#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export PATH="$(go env GOPATH)/bin:$PATH"
export ANDROID_NDK_HOME="${ANDROID_NDK_HOME:-$ANDROID_HOME/ndk/28.2.13676358}"
go mod download
# Build the pinned x/mobile tools with this module's locked x/tools version.
# Installing with @version would discard these dependency selections and pull
# 2024 x/tools, which is incompatible with the Go 1.26 token.FileSet layout.
mkdir -p android/app/libs android/app/src/main/assets/licenses
if [[ ! -f android/app/libs/engine.aar || "${REBUILD_CORE:-0}" == 1 ]]; then
  go install golang.org/x/mobile/cmd/gomobile golang.org/x/mobile/cmd/gobind
  gomobile bind -target=android/arm64,android/amd64 -androidapi=26 -ldflags='-s -w' -o android/app/libs/engine.aar ./engine github.com/tailscale/tailscale-android/libtailscale
fi
cp -R ../frontend/dist/. android/app/src/main/assets/
cp ../THIRD_PARTY_NOTICES.md ../LICENSE android/app/src/main/assets/licenses/
upstream_dir="$(go list -m -f '{{.Dir}}' github.com/tailscale/tailscale-android)"
cp "$upstream_dir/LICENSE" android/app/src/main/assets/licenses/TAILSCALE-ANDROID-LICENSE.txt
tailscale_dir="$(go list -m -f '{{.Dir}}' tailscale.com)"
cp "$tailscale_dir/LICENSE" android/app/src/main/assets/licenses/TAILSCALE-LICENSE.txt
mobile_dir="$(go list -m -f '{{.Dir}}' golang.org/x/mobile)"
cp "$mobile_dir/LICENSE" android/app/src/main/assets/licenses/GO-MOBILE-LICENSE.txt
wireguard_dir="$(go list -m -f '{{.Dir}}' github.com/tailscale/wireguard-go)"
cp "$wireguard_dir/LICENSE" android/app/src/main/assets/licenses/WIREGUARD-GO-LICENSE.txt
gradle -p android --no-daemon assembleDebug lintDebug
mkdir -p ../bin/android-preview
cp android/app/build/outputs/apk/debug/app-debug.apk ../bin/android-preview/headscaleclient-0.2.1-android.1.apk
"$ANDROID_HOME/build-tools/35.0.0/apksigner" verify --verbose ../bin/android-preview/headscaleclient-0.2.1-android.1.apk
sha256sum ../bin/android-preview/headscaleclient-0.2.1-android.1.apk
