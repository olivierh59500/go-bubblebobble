#!/bin/sh
set -eu
project_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$project_dir/scripts/build-android.sh"
android_sdk=${ANDROID_HOME:-${ANDROID_SDK_ROOT:-/opt/homebrew/share/android-commandlinetools}}
adb_path="$android_sdk/platform-tools/adb"
[ -x "$adb_path" ] || { printf '%s\n' 'Set ANDROID_HOME to the Android SDK containing adb.' >&2; exit 1; }
count=$("$adb_path" devices | awk '$2 == "device" { count++ } END { print count+0 }')
[ "$count" = 1 ] || { printf '%s\n' 'Connect and authorize exactly one Android device. The APK is already built.' >&2; exit 1; }
"$adb_path" install -r "$project_dir/dist/android/bubblebobble-android-arm64.apk"
"$adb_path" shell am force-stop com.olivierh.bubblebobble
"$adb_path" shell am start -n com.olivierh.bubblebobble/.MainActivity
