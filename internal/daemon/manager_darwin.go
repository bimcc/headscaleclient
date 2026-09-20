package daemon

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/headscaleclient/headscaleclient/internal/domain"
	"github.com/headscaleclient/headscaleclient/internal/macos"
	"tailscale.com/client/local"
)

type darwinRunner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

type osDarwinRunner struct{}

func (osDarwinRunner) Run(ctx context.Context, command string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, command, args...).CombinedOutput()
}

type platformManager struct {
	runner darwinRunner
}

func newPlatformManager() Manager { return &platformManager{runner: osDarwinRunner{}} }

func (m *platformManager) Inspect(ctx context.Context) (domain.EngineStatus, error) {
	status := domain.EngineStatus{BundledVersion: BundledVersion,
		Ownership: domain.EngineOwnershipMissing, Service: domain.EngineServiceMissing}
	if err := ctx.Err(); err != nil {
		return status, err
	}
	_, err := os.Lstat(macos.LaunchDaemonPath)
	if os.IsNotExist(err) {
		// GUI-only builds can reuse upstream's macOS socket/TCP discovery.
		if _, queryErr := (&local.Client{}).Status(ctx); queryErr == nil {
			status.Ownership, status.Service = domain.EngineOwnershipExternal, domain.EngineServiceRunning
		} else if _, appErr := os.Stat("/Applications/Tailscale.app"); appErr == nil {
			status.Ownership, status.Service = domain.EngineOwnershipExternal, domain.EngineServiceStopped
		}
		return status, nil
	}
	if err != nil {
		return status, err
	}
	if err := validateDarwinInstall(); err != nil {
		return status, domain.WrapError(domain.ErrorPermissionDenied, "The macOS network service installation needs repair.", err).
			WithDetail("Reinstall the HeadscaleClient PKG installer.")
	}
	status.Ownership = domain.EngineOwnershipManaged
	status.PayloadAvailable = true
	output, printErr := m.runner.Run(ctx, "/bin/launchctl", "print", "system/"+macos.ServiceLabel)
	if err := ctx.Err(); err != nil {
		return status, err
	}
	status.Service = darwinServiceState(string(output), printErr)
	status.CanStart = status.Service == domain.EngineServiceStopped
	return status, nil
}

func (m *platformManager) EnsureInstalled(ctx context.Context) error {
	status, err := m.Inspect(ctx)
	if err != nil {
		return err
	}
	if status.Service == domain.EngineServiceRunning {
		return nil
	}
	if status.Ownership == domain.EngineOwnershipExternal {
		return domain.NewError(domain.ErrorDaemonStopped, "Start the existing Tailscale application or service, then retry.")
	}
	if status.Ownership != domain.EngineOwnershipManaged {
		return domain.NewError(domain.ErrorDaemonMissing, "Install the HeadscaleClient macOS PKG to set up the network service.")
	}
	// The helper is installed by Installer.app, root-owned, and has no
	// frontend-controlled command/path parameters. Never elevate the GUI itself.
	if err := validateDarwinInstall(); err != nil {
		return err
	}
	var output []byte
	if os.Geteuid() == 0 {
		output, err = m.runner.Run(ctx, "/bin/sh", macos.ControlPath, "start")
	} else {
		script := "do shell script " + strconv.Quote("/bin/sh '"+macos.ControlPath+"' start") + " with administrator privileges"
		output, err = m.runner.Run(ctx, "/usr/bin/osascript", "-e", script)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return domain.WrapError(domain.ErrorPermissionDenied, "The network service could not be started. Approve the macOS administrator prompt and retry.", err).
			WithDetail(strings.TrimSpace(string(output))).WithRetryable(true)
	}
	// The application layer waits for LocalAPI before reporting success.
	return nil
}

func darwinServiceState(output string, printErr error) domain.EngineServiceState {
	if printErr == nil {
		for line := range strings.SplitSeq(output, "\n") {
			key, value, ok := strings.Cut(strings.TrimSpace(line), " = ")
			if ok && key == "state" && value == "running" {
				return domain.EngineServiceRunning
			}
		}
	}
	return domain.EngineServiceStopped
}

func validateDarwinInstall() error {
	for _, path := range []string{
		"/Library", "/Library/Application Support", "/Library/Application Support/BIMCC",
		macos.ServiceRoot, macos.ServiceRoot + "/daemon", "/Library/LaunchDaemons",
		macos.LaunchDaemonPath, macos.DaemonPath, macos.ControlPath,
	} {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != 0 || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 {
			return fmt.Errorf("unsafe service installation permissions: %s", path)
		}
	}
	return nil
}
