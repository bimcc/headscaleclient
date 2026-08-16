# Security Model

## Trust boundaries

- The WebView frontend is untrusted input to narrow Go service methods.
- The Go application trusts only validated LocalAPI responses and local config.
- A custom control server is remote and untrusted until TLS validation succeeds.
- `tailscaled` owns node keys and privileged network configuration.
- The machine installer and explicit service repair are privileged operations;
  the long-running Wails process is not.

## Rules

- Do not expose arbitrary filesystem, command execution, or HTTP methods to the frontend.
- Do not load remote pages or scripts inside the application WebView.
- Apply a restrictive Content Security Policy in production.
- Require HTTPS for non-loopback endpoints.
- Never implement "accept any certificate" as a persistent silent option.
- A custom CA is stored and referenced explicitly; certificate errors remain visible.
- URLs cannot contain usernames, passwords, or fragments.
- Auth keys are one-time inputs, redacted from logs, and not stored in normal config.
- Browser login tokens and daemon node keys are never copied into application storage.
- Diagnostic exports redact tokens, profile pictures where practical, and local paths.
- GUI and watcher goroutines run without administrator/root elevation.
- The frontend cannot submit service names, executable paths, shell commands,
  or arguments. It can request only the fixed `EnsureDaemon` operation.
- Daemon payloads must match committed source and runtime hashes. Windows also
  requires expected Authenticode signers; Linux requires pinned Go build info.
- Never replace or uninstall a service whose executable path is outside the
  active HeadscaleClient installation.
- Linux privilege escalation is limited to starting a fixed known systemd unit;
  no product-specific passwordless PolicyKit rule is installed.

## Frontend method policy

All Wails-bound inputs have length limits and are validated again in Go. The
frontend may improve usability but is not a security boundary. Structured
errors sent to the frontend exclude stack traces and secret-bearing payloads.

## Update policy

Production updates require signed artifacts and a signed update manifest.
Automatic update work remains disabled until signing keys and rollback behavior
are documented and tested on all supported platforms.
