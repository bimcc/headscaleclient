#!/bin/sh

set -u

# Debian passes "upgrade" and RPM passes "1" while replacing an installed
# package. The new package keeps ownership of the same unit in those cases.
case "${1:-}" in
  upgrade|failed-upgrade|1)
    exit 0
    ;;
esac

if command -v systemctl >/dev/null 2>&1; then
  systemctl disable --now headscaleclient-tailscaled.service >/dev/null 2>&1 || true
fi

exit 0
