#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
mkdir -p bin/android-preview
trap 'adb logcat -d -s AndroidRuntime:E > bin/android-preview/emulator-crashes.txt || true' EXIT
gradle -p mobile/android --no-daemon connectedDebugAndroidTest
adb pull /sdcard/Download/headscale-keyboard.png bin/android-preview/emulator-keyboard.png
# Gradle's managed test runner removes its installed packages after the tests.
# Install the same verified APK again to capture an ordinary user launch.
adb install -r -g bin/android-preview/headscaleclient-0.2.1-android.2.apk
adb shell am start -W -n com.bimcc.headscaleclient.preview/com.bimcc.headscaleclient.MainActivity
sleep 3
adb exec-out screencap -p > bin/android-preview/emulator-overview.png
adb shell dumpsys meminfo com.bimcc.headscaleclient.preview > bin/android-preview/emulator-memory.txt
