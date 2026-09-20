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
plist=/Library/LaunchDaemons/io.headscaleclient.tailscaled.plist
release_version=$(node -p 'require("./frontend/package.json").version')
verify_installed_version() {
  test "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' /Applications/HeadscaleClient.app/Contents/Info.plist)" = "$release_version"
  cmp bin/headscaleclient.app/Contents/MacOS/headscaleclient /Applications/HeadscaleClient.app/Contents/MacOS/headscaleclient
  test "$(pkgutil --pkg-info io.headscaleclient.desktop.pkg | awk '/^version:/ {print $2}')" = "$release_version"
}

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

sudo installer -showChoicesXML -pkg "$pkg" -target / > bin/macos-installer-choices.xml
plutil -lint bin/macos-installer-choices.xml
sudo installer -pkg "$pkg" -target /
test "$(cat "$service_root/installation-mode")" = managed
verify_installed_version
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

# Installing while two engines exist must fail without stopping either engine.
managed_pid=$(sudo launchctl print system/io.headscaleclient.tailscaled | awk '/^[[:space:]]*pid =/ {print $3; exit}')
sudo mkdir /Applications/Tailscale.app
if sudo installer -pkg "$pkg" -target /; then
  echo 'Conflicting managed and official installations must be rejected.' >&2
  exit 1
fi
test "$(sudo launchctl print system/io.headscaleclient.tailscaled | awk '/^[[:space:]]*pid =/ {print $3; exit}')" = "$managed_pid"
sudo rmdir /Applications/Tailscale.app

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

# A detected but stopped official installation is reused, not overwritten.
sudo mkdir /Applications/Tailscale.app
sudo touch /Applications/Tailscale.app/preserved-fixture
sudo installer -pkg "$pkg" -target /
test "$(cat "$service_root/installation-mode")" = external
test ! -e "$plist"
sudo test -f /Applications/Tailscale.app/preserved-fixture
sudo /bin/sh "$service_root/uninstall-service"
sudo test -f /Applications/Tailscale.app/preserved-fixture
sudo rm /Applications/Tailscale.app/preserved-fixture
sudo rmdir /Applications/Tailscale.app

# Run an actual upstream daemon as a separate Homebrew-like launchd service.
# No real login, host routes or credentials: this fixture uses userspace mode.
fixture=$(mktemp -d /private/tmp/headscaleclient-external.XXXXXX)
external_plist=/Library/LaunchDaemons/homebrew.mxcl.tailscale.plist
external_job=system/homebrew.mxcl.tailscale
sudo install -m 755 "bin/daemon/darwin-$arch/tailscaled" "$fixture/tailscaled"
sudo install -m 755 "bin/daemon/darwin-$arch/tailscale" "$fixture/tailscale"
sudo /usr/libexec/PlistBuddy -c 'Add :Label string homebrew.mxcl.tailscale' "$external_plist"
sudo /usr/libexec/PlistBuddy -c 'Add :ProgramArguments array' "$external_plist"
sudo /usr/libexec/PlistBuddy -c "Add :ProgramArguments:0 string $fixture/tailscaled" "$external_plist"
sudo /usr/libexec/PlistBuddy -c 'Add :ProgramArguments:1 string --tun=userspace-networking' "$external_plist"
sudo /usr/libexec/PlistBuddy -c 'Add :ProgramArguments:2 string --socket=/var/run/tailscaled.socket' "$external_plist"
sudo /usr/libexec/PlistBuddy -c "Add :ProgramArguments:3 string --state=$fixture/state/tailscaled.state" "$external_plist"
sudo /usr/libexec/PlistBuddy -c 'Add :RunAtLoad bool true' "$external_plist"
sudo chmod 644 "$external_plist"
sudo mkdir -m 700 "$fixture/state"
sudo touch "$fixture/state/preserved-identity-fixture"
sudo launchctl bootstrap system "$external_plist"
for i in {1..30}; do
  if "$fixture/tailscale" status --json >/dev/null 2>&1; then break; fi
  sleep 1
done
"$fixture/tailscale" status --json >/dev/null
external_pid=$(sudo launchctl print "$external_job" | awk '/^[[:space:]]*pid =/ {print $3; exit}')
test -n "$external_pid"
external_hash=$(shasum -a 256 "$external_plist" "$fixture/tailscaled")
sudo "$fixture/tailscale" debug prefs > "$fixture/prefs-before.json"
sudo installer -pkg "$pkg" -target /
test "$(cat "$service_root/installation-mode")" = external
test ! -e "$plist"
test ! -S "$socket"
test "$(sudo launchctl print "$external_job" | awk '/^[[:space:]]*pid =/ {print $3; exit}')" = "$external_pid"
test "$(shasum -a 256 "$external_plist" "$fixture/tailscaled")" = "$external_hash"
sudo "$fixture/tailscale" debug prefs > "$fixture/prefs-after.json"
cmp "$fixture/prefs-before.json" "$fixture/prefs-after.json"
open /Applications/HeadscaleClient.app
sleep 5
pgrep -x headscaleclient >/dev/null
pkill -x headscaleclient
sleep 2
test ! -e "$plist"
sudo /bin/sh "$service_root/uninstall-service"
"$fixture/tailscale" status --json >/dev/null
sudo test -f "$fixture/state/preserved-identity-fixture"
test "$(shasum -a 256 "$external_plist" "$fixture/tailscaled")" = "$external_hash"

# Registered but stopped external service must also remain external.
sudo launchctl bootout "$external_job"
sudo installer -pkg "$pkg" -target /
test "$(cat "$service_root/installation-mode")" = external
test ! -e "$plist"
test "$(shasum -a 256 "$external_plist" "$fixture/tailscaled")" = "$external_hash"
sudo rm "$external_plist"
sudo rm -f /var/run/tailscaled.socket

# Explicit user removal followed by rerunning the SAME PKG enables managed mode.
sudo installer -pkg "$pkg" -target /
test "$(cat "$service_root/installation-mode")" = managed
"$cli" --socket="$socket" status --json >/dev/null
sudo test -f "$service_root/state/upgrade-check"
sudo /bin/sh "$service_root/uninstall-service"
echo 'Unified installer: stopped official fixture, live/stopped external daemon, conflict protection, unchanged external prefs/process/payload, removal and explicit migration passed.'

# Upgrade from the previously published split-package edition (immutable hashes).
case "$arch" in
  arm64) previous_sha=aeedc347da3829856b6b7a629285fedad5422d14638c59c8ff62e035ec804191 ;;
  amd64) previous_sha=9f71d0e3a4523a2b031f829ca043ce276d84e8f42d865ed2d893413c853ed496 ;;
esac
previous="$PWD/bin/previous-macos-$arch.pkg"
curl --fail --location --retry 2 --max-time 120 \
  "https://github.com/bimcc/headscaleclient/releases/download/v0.1.0-preview.20260920/headscaleclient-macos-$arch-installer.pkg" -o "$previous"
test "$(shasum -a 256 "$previous" | awk '{print $1}')" = "$previous_sha"
# Remove the newer GUI from the install location so this really starts at 0.1.0.
# Preserve it under the already-created disposable fixture directory.
sudo mv /Applications/HeadscaleClient.app "$fixture/HeadscaleClient-current.app"
sudo installer -pkg "$previous" -target /
test "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' /Applications/HeadscaleClient.app/Contents/Info.plist)" = 0.1.0
sudo test -f "$service_root/state/upgrade-check"
sudo installer -pkg "$pkg" -target /
verify_installed_version
test "$(cat "$service_root/installation-mode")" = managed
sudo test -f "$service_root/state/upgrade-check"
"$cli" --socket="$socket" status --json >/dev/null
sudo /bin/sh "$service_root/uninstall-service"
echo 'Upgrade from the published split-package preview passed with state preserved.'
