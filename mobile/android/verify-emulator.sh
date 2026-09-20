#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
mkdir -p bin/android-preview
gradle -p mobile/android --no-daemon connectedDebugAndroidTest
adb shell am start -W -n com.bimcc.headscaleclient.preview/com.bimcc.headscaleclient.MainActivity
sleep 3
adb exec-out screencap -p > bin/android-preview/emulator-overview.png
adb shell dumpsys meminfo com.bimcc.headscaleclient.preview > bin/android-preview/emulator-memory.txt
