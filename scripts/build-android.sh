#!/bin/sh
set -eu

project_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
android_sdk=${ANDROID_HOME:-${ANDROID_SDK_ROOT:-}}
java_path=${JAVA_HOME:-}
if [ -z "$android_sdk" ] && [ -d /opt/homebrew/share/android-commandlinetools ]; then android_sdk=/opt/homebrew/share/android-commandlinetools; fi
if [ -z "$android_sdk" ] && [ -d "$HOME/Library/Android/sdk" ]; then android_sdk="$HOME/Library/Android/sdk"; fi
if [ -z "$java_path" ] && [ -d /opt/homebrew/opt/openjdk@17 ]; then java_path=/opt/homebrew/opt/openjdk@17; fi
if [ -z "$java_path" ] && [ -d "/Applications/Android Studio.app/Contents/jbr/Contents/Home" ]; then java_path="/Applications/Android Studio.app/Contents/jbr/Contents/Home"; fi
fail() { printf '%s\n' "$*" >&2; exit 1; }
[ -n "$android_sdk" ] || fail 'Set ANDROID_HOME to the Android SDK.'
[ -f "$android_sdk/platforms/android-36/android.jar" ] || fail 'Android SDK Platform 36 is required.'
[ -d "$android_sdk/ndk/28.2.13676358" ] || fail 'Android NDK 28.2.13676358 is required.'
[ -x "$java_path/bin/java" ] || fail 'Set JAVA_HOME to Java 17.'
"$java_path/bin/java" -version 2>&1 | awk '/version "17\./ { found=1 } END { exit !found }' || fail 'Java 17 is required.'
export JAVA_HOME="$java_path" ANDROID_HOME="$android_sdk" ANDROID_NDK_HOME="$android_sdk/ndk/28.2.13676358"
export PATH="$JAVA_HOME/bin:$PATH"
export CGO_LDFLAGS="${CGO_LDFLAGS:-} -Wl,-z,max-page-size=16384 -Wl,-z,common-page-size=16384"
cd "$project_dir"
ebiten_version=$(go list -m -f '{{.Version}}' github.com/hajimehoshi/ebiten/v2)
[ "$ebiten_version" = v2.10.2 ] || fail 'Update the Android tool version together with Ebitengine.'
mkdir -p android/app/libs dist/android
go run "github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@$ebiten_version" bind \
    -target android/arm64 -androidapi 23 -javapkg com.olivierh.bubblebobble \
    -o android/app/libs/bubblebobble.aar ./mobile
./android/gradlew -p android --console=plain clean assembleDebug lintDebug
apk=android/app/build/outputs/apk/debug/app-debug.apk
tools_dir="$android_sdk/build-tools/36.0.0"
"$tools_dir/apksigner" verify --verbose "$apk"
"$tools_dir/zipalign" -c -P 16 4 "$apk"
"$tools_dir/aapt" dump badging "$apk" > dist/android/APK-INFO.txt
cp "$apk" dist/android/bubblebobble-android-arm64.apk
python3 scripts/check-android.py dist/android/bubblebobble-android-arm64.apk "$ANDROID_NDK_HOME"
(cd dist/android && shasum -a 256 bubblebobble-android-arm64.apk > SHA256SUMS)
printf '%s\n' 'Built and verified dist/android/bubblebobble-android-arm64.apk (debug signed).' 'No device was accessed.'
