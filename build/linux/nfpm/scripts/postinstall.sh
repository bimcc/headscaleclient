#!/bin/sh

set -u

managed_unit="headscaleclient-tailscaled.service"
external_unit="tailscaled.service"

if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload >/dev/null 2>&1 || true
  external_load_state="$(systemctl show "$external_unit" --property=LoadState --value 2>/dev/null || true)"

  if systemctl is-active --quiet "$managed_unit"; then
    if ! systemctl try-restart "$managed_unit"; then
      echo "Warning: the managed HeadscaleClient network service could not be restarted." >&2
    fi
  elif [ "$external_load_state" = "loaded" ]; then
    echo "HeadscaleClient will reuse the existing tailscaled.service."
  elif [ -d /run/systemd/system ]; then
    if ! systemctl enable --now "$managed_unit"; then
      echo "Warning: the managed HeadscaleClient network service could not be enabled and started." >&2
    fi
  elif ! systemctl enable "$managed_unit"; then
    echo "Warning: the managed HeadscaleClient network service could not be enabled." >&2
  fi
else
  echo "Warning: systemd is unavailable; install and start tailscaled separately." >&2
fi

# Update desktop database for .desktop file changes
# This makes the application appear in application menus and registers its capabilities.
if command -v update-desktop-database >/dev/null 2>&1; then
  echo "Updating desktop database..."
  update-desktop-database -q /usr/share/applications
else
  echo "Warning: update-desktop-database command not found. Desktop file may not be immediately recognized." >&2
fi

# Update MIME database for custom URL schemes (x-scheme-handler)
# This ensures the system knows how to handle your custom protocols.
if command -v update-mime-database >/dev/null 2>&1; then
  echo "Updating MIME database..."
  update-mime-database -n /usr/share/mime
else
  echo "Warning: update-mime-database command not found. Custom URL schemes may not be immediately recognized." >&2
fi

exit 0
