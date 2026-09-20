#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/../.."
root=$(pwd)
arch=${1:?Specify amd64 or arm64}
case "$arch" in amd64) machine_arch=x86_64 ;; arm64) machine_arch=arm64 ;; *) exit 2 ;; esac
[[ $(uname -s) == Darwin ]] || { echo 'Build this package on macOS.' >&2; exit 1; }
app="$root/bin/headscaleclient.app"
payload="$root/bin/daemon/darwin-$arch"
version=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$app/Contents/Info.plist")
for binary in "$app/Contents/MacOS/headscaleclient" "$payload/tailscale" "$payload/tailscaled"; do
  lipo "$binary" -verify_arch "$machine_arch"
  codesign --verify --strict "$binary"
done
codesign --verify --deep --strict "$app"
stage=$(mktemp -d "$root/.task/macos-package.XXXXXX")
trap 'rm -rf "$stage"' EXIT
install_root="$stage/root/Library/Application Support/BIMCC/HeadscaleClient"
mkdir -p "$stage/root/Applications" "$install_root/daemon/licenses" "$stage/scripts"
ditto "$app" "$stage/root/Applications/HeadscaleClient.app"
cp "$payload/tailscale" "$payload/tailscaled" "$payload/provenance.json" "$install_root/daemon/"
cp "$payload/licenses/TAILSCALE-LICENSE.txt" "$install_root/daemon/licenses/"
cp build/darwin/service/service-control build/darwin/service/service-policy build/darwin/service/uninstall-service "$install_root/"
# Do not ship an active LaunchDaemon in the payload: postinstall decides ownership.
cp build/darwin/service/io.headscaleclient.tailscaled.plist "$install_root/launchd-template.plist"
cp build/darwin/installer/preinstall build/darwin/installer/postinstall "$stage/scripts/"
cp build/darwin/service/service-policy "$stage/scripts/"
cp THIRD_PARTY_NOTICES.md LICENSE "$install_root/"
chmod 755 "$stage/scripts/"* "$install_root/service-control" "$install_root/service-policy" "$install_root/uninstall-service" "$install_root/daemon/tailscale" "$install_root/daemon/tailscaled"
chmod 644 "$install_root/launchd-template.plist"
plutil -lint "$install_root/launchd-template.plist"
# A fixed install location prevents Installer from updating an unrelated copied app.
pkgbuild --analyze --root "$stage/root" "$stage/components.plist"
/usr/libexec/PlistBuddy -c 'Set :0:BundleIsRelocatable false' "$stage/components.plist"
pkgbuild --root "$stage/root" --identifier io.headscaleclient.desktop.pkg --version "$version" \
  --install-location / --ownership recommended --scripts "$stage/scripts" \
  --component-plist "$stage/components.plist" "$stage/component.pkg"
sed -e "s/@ARCH@/$machine_arch/g" -e "s/@VERSION@/$version/g" \
  build/darwin/installer/Distribution.xml > "$stage/distribution.xml"
productbuild --distribution "$stage/distribution.xml" --package-path "$stage" --resources build/darwin/installer \
  "$root/bin/headscaleclient-macos-$arch-installer.pkg"
(cd bin && shasum -a 256 "headscaleclient-macos-$arch-installer.pkg" > "headscaleclient-macos-$arch-SHA256SUMS.txt")
