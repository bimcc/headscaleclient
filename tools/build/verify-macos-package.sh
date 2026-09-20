#!/bin/bash
# Disposable macOS CI host only: installs and removes the managed service.
set -euo pipefail
[[ ${CI:-} == true ]] || { echo 'Run installation verification on a disposable CI Mac.' >&2; exit 1; }
cd "$(dirname "$0")/../.."
arch=${1:?Specify architecture}
pkg="$PWD/bin/headscaleclient-macos-$arch-installer.pkg"
service_root='/Library/Application Support/BIMCC/HeadscaleClient'
socket='/var/run/headscaleclient-tailscaled.socket'
cli="$service_root/daemon/tailscale"

# Keep failures actionable on the disposable runner without leaking real state.
diagnose() {
  result=$?
  if [[ $result -ne 0 ]]; then
    sudo tail -n 80 /var/log/install.log || true
    sudo launchctl print system/io.headscaleclient.tailscaled || true
  fi
  exit "$result"
}
trap diagnose EXIT

sudo installer -pkg "$pkg" -target /
sudo "$cli" --socket="$socket" status --json | tee bin/macos-initial-status.json
node -e 'const s=JSON.parse(require("fs").readFileSync("bin/macos-initial-status.json")); if(s.BackendState!=="NeedsLogin") throw Error("unexpected fresh state: "+s.BackendState)'
sudo /bin/sh "$service_root/service-control" stop
sudo /bin/sh "$service_root/service-control" start
sudo "$cli" --socket="$socket" status --json >/dev/null

# Confirm the normal desktop user can read and mutate its daemon without sudo.
"$cli" --socket="$socket" status --json >/dev/null
"$cli" --socket="$socket" set --accept-dns=false --accept-routes=false

# Reinstall preserves state instead of resetting saved identities.
sudo touch "$service_root/state/upgrade-check"
sudo installer -pkg "$pkg" -target /
sudo test -f "$service_root/state/upgrade-check"
sudo "$cli" --socket="$socket" status --json >/dev/null

# Starting the desktop must not require elevation or end the network service.
open /Applications/HeadscaleClient.app
sleep 5
pgrep -x headscaleclient >/dev/null
pkill -x headscaleclient
sleep 2
sudo "$cli" --socket="$socket" status --json >/dev/null

sudo /bin/sh "$service_root/uninstall-service"
test ! -e /Library/LaunchDaemons/io.headscaleclient.tailscaled.plist
sudo test -f "$service_root/state/upgrade-check"
if sudo launchctl print system/io.headscaleclient.tailscaled >/dev/null 2>&1; then
  echo 'Owned service remains loaded after uninstall.' >&2
  exit 1
fi
echo 'Fresh install, normal-user LocalAPI, restart, upgrade, GUI launch and service uninstall passed.'
