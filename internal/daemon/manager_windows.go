//go:build windows

package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/headscaleclient/headscaleclient/internal/domain"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const windowsServiceName = "Tailscale"

type platformManager struct{}

func newPlatformManager() Manager {
	return &platformManager{}
}

func (m *platformManager) Inspect(ctx context.Context) (domain.EngineStatus, error) {
	if err := ctx.Err(); err != nil {
		return domain.EngineStatus{}, classifyContextError(err)
	}
	payloadPath, payloadAvailable := windowsPayload()
	status := domain.EngineStatus{
		Ownership:        domain.EngineOwnershipUnknown,
		Service:          domain.EngineServiceUnknown,
		BundledVersion:   BundledVersion,
		PayloadAvailable: payloadAvailable,
	}

	serviceStatus, binaryPath, err := inspectWindowsService()
	if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
		status.Service = domain.EngineServiceMissing
		if payloadAvailable {
			status.Ownership = domain.EngineOwnershipPrepared
			status.CanInstall = true
		} else {
			status.Ownership = domain.EngineOwnershipMissing
		}
		return status, nil
	}
	if err != nil {
		return status, domain.WrapError(domain.ErrorDaemonUnavailable, "The Windows network service could not be inspected.", err).WithRetryable(true)
	}

	status.Service = mapWindowsServiceState(serviceStatus.State)
	if matchesManagedWindowsExecutable(binaryPath, payloadPath, legacyWindowsPayload()) {
		status.Ownership = domain.EngineOwnershipManaged
	} else {
		status.Ownership = domain.EngineOwnershipExternal
	}
	status.CanStart = status.Service == domain.EngineServiceStopped &&
		(status.Ownership == domain.EngineOwnershipExternal || payloadAvailable)
	return status, nil
}

func (m *platformManager) EnsureInstalled(ctx context.Context) error {
	status, err := m.Inspect(ctx)
	if err != nil {
		return err
	}
	if status.Service == domain.EngineServiceRunning || status.Service == domain.EngineServiceStarting {
		if status.Service == domain.EngineServiceStarting {
			return waitForWindowsService(ctx, true)
		}
		return nil
	}
	payloadPath, payloadAvailable := windowsPayload()
	if status.Service == domain.EngineServiceMissing {
		if !payloadAvailable {
			return domain.NewError(domain.ErrorDaemonMissing, "The bundled network service payload is unavailable.").WithDetail("Reinstall HeadscaleClient using the full machine installer.")
		}
		return configureManagedWindowsService(ctx, payloadPath, false)
	}
	if status.Service == domain.EngineServiceStopping {
		if err := waitForWindowsService(ctx, false); err != nil {
			return err
		}
		return m.EnsureInstalled(ctx)
	}
	if status.Service != domain.EngineServiceStopped {
		return domain.NewError(domain.ErrorDaemonUnavailable, "The Windows network service is in an unsupported state.").WithRetryable(true)
	}
	if status.Ownership == domain.EngineOwnershipManaged {
		if !payloadAvailable {
			return domain.NewError(domain.ErrorDaemonMissing, "The managed network service payload is unavailable.").WithDetail("Reinstall HeadscaleClient using the full machine installer.")
		}
		return configureManagedWindowsService(ctx, payloadPath, true)
	}

	if err := startWindowsService(); err != nil {
		if !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return domain.WrapError(domain.ErrorDaemonStopped, "The external network service could not be started.", err).
				WithDetail(fmt.Sprintf("Windows service error: %v", err)).WithRetryable(true)
		}
		scPath := filepath.Join(os.Getenv("SystemRoot"), "System32", "sc.exe")
		if err := shellExecuteElevated(scPath, "start "+windowsServiceName, filepath.Dir(scPath)); err != nil {
			return domain.WrapError(domain.ErrorPermissionDenied, "Administrator approval is required to start the network service.", err).WithRetryable(true)
		}
	}
	return waitForWindowsService(ctx, true)
}

func configureManagedWindowsService(ctx context.Context, payloadPath string, repair bool) error {
	if !fileExists(payloadPath) {
		return domain.NewError(domain.ErrorDaemonMissing, "The managed network service executable is missing.").
			WithDetail("Reinstall HeadscaleClient using the full machine installer.")
	}
	powerShellPath := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	if !fileExists(powerShellPath) {
		return domain.NewError(domain.ErrorDaemonUnavailable, "Windows PowerShell is required to repair the network service.")
	}
	quotedPayload := "'" + strings.ReplaceAll(payloadPath, "'", "''") + "'"
	commands := []string{"$ErrorActionPreference = 'Stop'"}
	if repair {
		commands = append(commands,
			"& "+quotedPayload+" uninstall-system-daemon",
			"if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }",
		)
	}
	commands = append(commands,
		"& "+quotedPayload+" install-system-daemon",
		"if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }",
		"Start-Service -Name '"+windowsServiceName+"'",
	)
	parameters := `-NoProfile -NonInteractive -ExecutionPolicy Bypass -Command "` + strings.Join(commands, "; ") + `"`
	if err := shellExecuteElevated(powerShellPath, parameters, filepath.Dir(payloadPath)); err != nil {
		return domain.WrapError(domain.ErrorPermissionDenied, "Administrator approval is required to install or repair the network service.", err).WithRetryable(true)
	}
	if err := waitForWindowsManagedService(ctx, payloadPath); err != nil {
		if problem := domain.ProblemFromError(err); problem != nil && problem.Code == domain.ErrorCancelled {
			return err
		}
		return domain.WrapError(domain.ErrorDaemonStopped, "The managed network service could not be started.", err).
			WithDetail("Close any manually started tailscaled.exe process, then use Start and repair again.").WithRetryable(true)
	}
	return nil
}

func inspectWindowsService() (svc.Status, string, error) {
	manager, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return svc.Status{}, "", err
	}
	defer windows.CloseServiceHandle(manager)

	handle, err := windows.OpenService(manager, windows.StringToUTF16Ptr(windowsServiceName), windows.SERVICE_QUERY_STATUS|windows.SERVICE_QUERY_CONFIG)
	if err != nil {
		return svc.Status{}, "", err
	}
	service := &mgr.Service{Name: windowsServiceName, Handle: handle}
	defer service.Close()

	status, err := service.Query()
	if err != nil {
		return svc.Status{}, "", err
	}
	config, err := service.Config()
	if err != nil {
		return svc.Status{}, "", err
	}
	return status, config.BinaryPathName, nil
}

func startWindowsService() error {
	manager, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return err
	}
	defer windows.CloseServiceHandle(manager)

	handle, err := windows.OpenService(manager, windows.StringToUTF16Ptr(windowsServiceName), windows.SERVICE_START|windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return err
	}
	defer windows.CloseServiceHandle(handle)
	err = windows.StartService(handle, 0, nil)
	if errors.Is(err, windows.ERROR_SERVICE_ALREADY_RUNNING) {
		return nil
	}
	return err
}

func waitForWindowsService(ctx context.Context, running bool) error {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		status, _, err := inspectWindowsService()
		if err == nil {
			if running && status.State == svc.Running {
				return nil
			}
			if !running && status.State == svc.Stopped {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return classifyContextError(ctx.Err())
		case <-ticker.C:
		}
	}
}

func waitForWindowsManagedService(ctx context.Context, payloadPath string) error {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		status, binaryPath, err := inspectWindowsService()
		if err == nil && status.State == svc.Running && sameExecutable(binaryPath, payloadPath) {
			return nil
		}
		select {
		case <-ctx.Done():
			return classifyContextError(ctx.Err())
		case <-ticker.C:
		}
	}
}

func windowsPayload() (string, bool) {
	executable, err := os.Executable()
	if err != nil {
		return "", false
	}
	dir := filepath.Join(filepath.Dir(executable), "daemon")
	daemonPath := filepath.Join(dir, "tailscaled.exe")
	if fileExists(daemonPath) && fileExists(filepath.Join(dir, "wintun.dll")) {
		return daemonPath, true
	}
	return daemonPath, false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func legacyWindowsPayload() string {
	programFiles := os.Getenv("ProgramW6432")
	if programFiles == "" {
		programFiles = os.Getenv("ProgramFiles")
	}
	if programFiles == "" {
		return ""
	}
	return filepath.Join(programFiles, "BIMCC., Ltd.", "HeadscaleClient", "daemon", "tailscaled.exe")
}

func matchesManagedWindowsExecutable(binaryPath, payloadPath, legacyPayloadPath string) bool {
	return sameExecutable(binaryPath, payloadPath) ||
		(legacyPayloadPath != "" && sameExecutable(binaryPath, legacyPayloadPath))
}

func sameExecutable(left, right string) bool {
	left = strings.Trim(strings.TrimSpace(left), `"`)
	right = strings.Trim(strings.TrimSpace(right), `"`)
	leftPath, leftErr := filepath.Abs(left)
	rightPath, rightErr := filepath.Abs(right)
	return leftErr == nil && rightErr == nil && strings.EqualFold(filepath.Clean(leftPath), filepath.Clean(rightPath))
}

func mapWindowsServiceState(state svc.State) domain.EngineServiceState {
	switch state {
	case svc.Stopped:
		return domain.EngineServiceStopped
	case svc.StartPending:
		return domain.EngineServiceStarting
	case svc.Running:
		return domain.EngineServiceRunning
	case svc.StopPending:
		return domain.EngineServiceStopping
	default:
		return domain.EngineServiceUnknown
	}
}

func shellExecuteElevated(file, parameters, workingDirectory string) error {
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	filePointer, err := windows.UTF16PtrFromString(file)
	if err != nil {
		return err
	}
	parametersPointer, err := windows.UTF16PtrFromString(parameters)
	if err != nil {
		return err
	}
	directoryPointer, err := windows.UTF16PtrFromString(workingDirectory)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, verb, filePointer, parametersPointer, directoryPointer, windows.SW_HIDE)
}

func classifyContextError(err error) error {
	if errors.Is(err, context.Canceled) {
		return domain.WrapError(domain.ErrorCancelled, "The network service operation was cancelled.", err)
	}
	return domain.WrapError(domain.ErrorTimeout, "The network service did not reach the requested state in time.", err).WithRetryable(true)
}
