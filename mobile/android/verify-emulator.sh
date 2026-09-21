#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
mkdir -p bin/android-preview
trap 'adb logcat -d -s AndroidRuntime:E > bin/android-preview/emulator-crashes.txt || true' EXIT
gradle -p mobile/android --no-daemon connectedDebugAndroidTest
adb pull /sdcard/Download/headscale-keyboard.png bin/android-preview/emulator-keyboard.png
adb pull /sdcard/Download/headscale-overview.png bin/android-preview/emulator-overview.png
# Gradle's managed test runner removes its installed packages after the tests.
# Install the same verified APK again to capture an ordinary user launch.
adb install -r -g bin/android-preview/headscaleclient-0.2.1-android.2.apk
adb shell am start -W -n com.bimcc.headscaleclient.preview/com.bimcc.headscaleclient.MainActivity
sleep 3
adb shell 'run-as com.bimcc.headscaleclient.preview sh -c "echo retained > files/upgrade-sentinel"'
adb install -r -g bin/android-preview/headscaleclient-0.2.1-android.2.apk
test "$(adb shell run-as com.bimcc.headscaleclient.preview cat files/upgrade-sentinel | tr -d '\r')" = retained
test "$(adb shell pm list packages com.bimcc.headscaleclient | tr -d '\r')" = package:com.bimcc.headscaleclient.preview
adb shell am start -W -n com.bimcc.headscaleclient.preview/com.bimcc.headscaleclient.MainActivity
sleep 3
adb shell dumpsys meminfo com.bimcc.headscaleclient.preview > bin/android-preview/emulator-memory.txt
