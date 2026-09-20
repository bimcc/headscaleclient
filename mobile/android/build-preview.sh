#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export PATH="$(go env GOPATH)/bin:$PATH"
go mod download
go install golang.org/x/mobile/cmd/gomobile@v0.0.0-20240806205939-81131f6468ab
go install golang.org/x/mobile/cmd/gobind@v0.0.0-20240806205939-81131f6468ab
mkdir -p android/app/libs android/app/src/main/assets/licenses
gomobile bind -target=android/arm64,android/amd64 -androidapi=26 -o android/app/libs/engine.aar ./engine github.com/tailscale/tailscale-android/libtailscale
cp -R ../frontend/dist/. android/app/src/main/assets/
cp ../THIRD_PARTY_NOTICES.md ../LICENSE android/app/src/main/assets/licenses/
upstream_dir="$(go list -m -f '{{.Dir}}' github.com/tailscale/tailscale-android)"
cp "$upstream_dir/LICENSE" android/app/src/main/assets/licenses/TAILSCALE-ANDROID-LICENSE.txt
tailscale_dir="$(go list -m -f '{{.Dir}}' tailscale.com)"
cp "$tailscale_dir/LICENSE" android/app/src/main/assets/licenses/TAILSCALE-LICENSE.txt
gradle -p android --no-daemon assembleDebug lintDebug
mkdir -p ../bin/android-preview
cp android/app/build/outputs/apk/debug/app-debug.apk ../bin/android-preview/headscaleclient-0.2.1-android.1.apk
"$ANDROID_HOME/build-tools/35.0.0/apksigner" verify --verbose ../bin/android-preview/headscaleclient-0.2.1-android.1.apk
sha256sum ../bin/android-preview/headscaleclient-0.2.1-android.1.apk
