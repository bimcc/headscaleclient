//go:build linux

package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/headscaleclient/headscaleclient/internal/domain"
)

const (
	linuxExternalUnit   = "tailscaled.service"
	linuxManagedUnit    = "headscaleclient-tailscaled.service"
	linuxManagedPayload = "/usr/lib/headscaleclient/daemon/tailscaled"
	linuxLocalAPISocket = "/run/tailscale/tailscaled.sock"
)

type linuxCommandRunner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

type osLinuxCommandRunner struct{}

func (osLinuxCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

type platformManager struct {
	runner       linuxCommandRunner
	systemctl    string
	pkexec       string
	payloadPaths []string
	socketPath   string
}

type linuxUnitStatus struct {
	name       string
	loadState  string
	active     string
	sub        string
	executable string
	managed    bool
}

func newPlatformManager() Manager {
	payloadPaths := []string{linuxManagedPayload}
	if executable, err := os.Executable(); err == nil {
		payloadPaths = append(payloadPaths, filepath.Join(filepath.Dir(executable), "daemon", "tailscaled"))
	}
	return &platformManager{
		runner:       osLinuxCommandRunner{},
		systemctl:    firstExecutable("/usr/bin/systemctl", "/bin/systemctl", "systemctl"),
		pkexec:       firstExecutable("/usr/bin/pkexec", "/bin/pkexec", "pkexec"),
		payloadPaths: payloadPaths,
		socketPath:   linuxLocalAPISocket,
	}
}

func (m *platformManager) Inspect(ctx context.Context) (domain.EngineStatus, error) {
	status, _, err := m.inspect(ctx)
	return status, err
}

func (m *platformManager) EnsureInstalled(ctx context.Context) error {
	status, unit, err := m.inspect(ctx)
	if err != nil {
		return err
	}
	if status.Service == domain.EngineServiceRunning || status.Service == domain.EngineServiceStarting {
		return nil
	}
	if unit == "" {
		if status.PayloadAvailable {
			return domain.NewError(domain.ErrorDaemonMissing, "The managed systemd service is not installed.").
				WithDetail("Install the HeadscaleClient DEB, RPM, or Arch package as an administrator.")
		}
		return domain.NewError(domain.ErrorDaemonMissing, "The bundled network service payload is unavailable.").
			WithDetail("Reinstall HeadscaleClient using a service-bearing Linux package.")
	}

	if output, startErr := m.runner.Run(ctx, m.systemctl, "start", unit); startErr != nil {
		if ctx.Err() != nil {
			return classifyLinuxContextError(ctx.Err())
		}
		output, startErr = m.runner.Run(ctx, m.pkexec, m.systemctl, "start", unit)
		if startErr != nil {
			detail := strings.TrimSpace(string(output))
			return domain.WrapError(domain.ErrorPermissionDenied, "Administrator approval is required to start the network service.", startErr).
				WithDetail(detail).WithRetryable(true)
		}
	}

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		status, _, inspectErr := m.inspect(ctx)
		if inspectErr == nil && status.Service == domain.EngineServiceRunning {
			return nil
		}
		select {
		case <-ctx.Done():
			return classifyLinuxContextError(ctx.Err())
		case <-ticker.C:
		}
	}
}

func (m *platformManager) inspect(ctx context.Context) (domain.EngineStatus, string, error) {
	if err := ctx.Err(); err != nil {
		return domain.EngineStatus{}, "", classifyLinuxContextError(err)
	}
	payloadPath, payloadAvailable := firstExistingFile(m.payloadPaths...)
	status := domain.EngineStatus{
		Ownership:        domain.EngineOwnershipUnknown,
		Service:          domain.EngineServiceUnknown,
		BundledVersion:   BundledVersion,
		PayloadAvailable: payloadAvailable,
	}

	external, externalErr := m.inspectUnit(ctx, linuxExternalUnit, "")
	managed, managedErr := m.inspectUnit(ctx, linuxManagedUnit, payloadPath)
	if externalErr != nil && managedErr != nil {
		if linuxFileExists(m.socketPath) {
			status.Ownership = domain.EngineOwnershipExternal
			status.Service = domain.EngineServiceRunning
			return status, "", nil
		}
		if payloadAvailable {
			status.Ownership = domain.EngineOwnershipPrepared
			status.Service = domain.EngineServiceMissing
			return status, "", nil
		}
		return status, "", domain.WrapError(domain.ErrorDaemonUnavailable, "The Linux network service could not be inspected.", errors.Join(externalErr, managedErr)).WithRetryable(true)
	}

	selected, found := chooseLinuxUnit(external, managed)
	if !found {
		status.Service = domain.EngineServiceMissing
		if payloadAvailable {
			status.Ownership = domain.EngineOwnershipPrepared
		} else {
			status.Ownership = domain.EngineOwnershipMissing
		}
		return status, "", nil
	}

	status.Service = mapLinuxServiceState(selected.active)
	status.CanStart = status.Service == domain.EngineServiceStopped
	if selected.managed {
		status.Ownership = domain.EngineOwnershipManaged
	} else {
		status.Ownership = domain.EngineOwnershipExternal
	}
	return status, selected.name, nil
}

func (m *platformManager) inspectUnit(ctx context.Context, name, managedPayload string) (linuxUnitStatus, error) {
	unit := linuxUnitStatus{name: name}
	output, err := m.runner.Run(ctx, m.systemctl, "show", name,
		"--property=LoadState", "--property=ActiveState", "--property=SubState", "--property=ExecStart", "--no-pager")
	properties := parseSystemdProperties(string(output))
	unit.loadState = properties["LoadState"]
	unit.active = properties["ActiveState"]
	unit.sub = properties["SubState"]
	unit.executable = systemdExecPath(properties["ExecStart"])
	if managedPayload != "" {
		unit.managed = sameLinuxExecutable(unit.executable, managedPayload)
	}
	if err != nil && unit.loadState == "" {
		return unit, fmt.Errorf("inspect %s: %w: %s", name, err, strings.TrimSpace(string(output)))
	}
	return unit, nil
}

func chooseLinuxUnit(external, managed linuxUnitStatus) (linuxUnitStatus, bool) {
	loadedExternal := external.loadState == "loaded"
	loadedManaged := managed.loadState == "loaded"
	if loadedExternal && external.active == "active" {
		return external, true
	}
	if loadedManaged && managed.active == "active" {
		return managed, true
	}
	if loadedExternal {
		return external, true
	}
	if loadedManaged {
		return managed, true
	}
	return linuxUnitStatus{}, false
}

func parseSystemdProperties(output string) map[string]string {
	properties := make(map[string]string)
	for line := range strings.SplitSeq(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if ok {
			properties[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return properties
}

func systemdExecPath(value string) string {
	if index := strings.Index(value, "path="); index >= 0 {
		value = value[index+len("path="):]
		if end := strings.IndexAny(value, " ;"); end >= 0 {
			value = value[:end]
		}
		return strings.Trim(strings.TrimSpace(value), `"`)
	}
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[0], `"`)
}

func sameLinuxExecutable(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	leftPath, leftErr := filepath.Abs(left)
	rightPath, rightErr := filepath.Abs(right)
	return leftErr == nil && rightErr == nil && filepath.Clean(leftPath) == filepath.Clean(rightPath)
}

func mapLinuxServiceState(active string) domain.EngineServiceState {
	switch active {
	case "active":
		return domain.EngineServiceRunning
	case "activating":
		return domain.EngineServiceStarting
	case "deactivating":
		return domain.EngineServiceStopping
	case "inactive", "failed":
		return domain.EngineServiceStopped
	default:
		return domain.EngineServiceUnknown
	}
}

func firstExistingFile(paths ...string) (string, bool) {
	for _, path := range paths {
		if linuxFileExists(path) {
			return path, true
		}
	}
	return "", false
}

func firstExecutable(paths ...string) string {
	for _, path := range paths {
		if filepath.IsAbs(path) && linuxFileExists(path) {
			return path
		}
		if resolved, err := exec.LookPath(path); err == nil {
			return resolved
		}
	}
	return paths[len(paths)-1]
}

func linuxFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func classifyLinuxContextError(err error) error {
	if errors.Is(err, context.Canceled) {
		return domain.WrapError(domain.ErrorCancelled, "The network service operation was cancelled.", err)
	}
	return domain.WrapError(domain.ErrorTimeout, "The network service did not reach the requested state in time.", err).WithRetryable(true)
}
