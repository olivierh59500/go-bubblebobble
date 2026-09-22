# Android build and device validation

The Android application uses the same Go simulation, resources and frontend as the desktop version. Its package is `com.olivierh.bubblebobble`; it does not replace any other game. All saves are in the application's private files directory.

## Build without a phone

```sh
make android
```

Required tools: Go 1.26+, Java 17, Python 3, Android SDK Platform 36, Build Tools 36.0.0 and NDK 28.2.13676358. The checked-in Gradle wrapper selects Gradle 8.11.1 with Android Gradle Plugin 8.10.1. The build script detects the Homebrew SDK/JDK or accepts `ANDROID_HOME` and `JAVA_HOME`.

The script runs Ebitengine 2.10.2's `ebitenmobile bind` for `android/arm64`, generates `android/app/libs/bubblebobble.aar`, then performs a clean APK build and Android lint. It verifies the APK signature, ZIP alignment, uncompressed native packaging and 16 KiB ELF LOAD/RELRO alignment. Output:

```text
dist/android/bubblebobble-android-arm64.apk
dist/android/APK-INFO.txt
dist/android/SHA256SUMS
```

This is an installable **debug** build using Android's development signing key. Release builds must use a separate private signing key before distribution. Generated AARs, APKs, Gradle caches and machine-specific configuration are excluded from Git. No installation or ADB command is part of `make android`.

## Install later

With one authorized Android device attached:

```sh
./scripts/run-android.sh
```

For an existing APK, use the SDK's `adb` directly:

```sh
adb install -r dist/android/bubblebobble-android-arm64.apk
adb shell am start -n com.olivierh.bubblebobble/.MainActivity
```

## Touch and lifecycle checks

1. Test left, right and up on the pad, then hold up/right and BUBBLE with two fingers.
2. Release the fire finger while retaining direction. Release both and check that nothing stays held.
3. Rotate between both landscape orientations and check the camera cutout does not obscure controls.
4. Create a profile using the in-game keyboard, change its dragon and reopen the app to verify persistence.
5. Draw and erase tiles in the editor, save, play-test, return and reload the layout.
6. Pause, switch applications and lock/unlock the screen. The match must remain paused until resumed; controls must be released and audio must suspend/resume correctly.
7. Listen to the main theme through at least one full loop and check pitch, timing and absence of crackles.

## Bluetooth checks with two phones

The implementation uses Bluetooth Classic RFCOMM with a game-specific service UUID. The native shell handles Android permission prompts, optional discoverability, the device chooser and sockets. A short-lived secret protects the private Go/Java loopback connection; it is never transmitted to the other phone. Neither side exposes a LAN listener.

Android 12+ asks for CONNECT and either ADVERTISE (hosting) or SCAN (joining). Earlier Android releases need location permission for discovery; it is not used for gameplay or positioning. Bluetooth is optional, so solo play works on devices without an adapter or without these permissions.

Verify on actual devices:

1. Deny permissions, cancel pairing and cancel hosting; return to the menu and retry successfully.
2. Connect both previously paired and newly discovered phones. Green must respond only to host input and blue only to guest input.
3. Move, jump and shoot simultaneously on both phones. Pop each other's bubbles and collect bonuses independently.
4. Lose a life on one device and verify both display the same lives, enemies, score and round.
5. Pause or background either phone. Resume from either phone and check that neither dragon keeps an old input.
6. Win/lose a run, restart from the host, and verify both profiles record the completed result once.
7. Disable Bluetooth or move out of range. The connection must end cleanly; host/join again to start a new match.
8. Connect different builds and check that an incompatible protocol/campaign is rejected.

Automated Go tests cover packet framing, malformed peers, authoritative snapshots, stale input, pause commands, cancellation, final scores, independent players and race safety. They cannot establish real-world radio latency, Android pairing behavior or hardware audio quality. Those checks remain pending until two devices are available.

The loopback bridge can also be checked on the development machine with `go test -race -tags=integration ./internal/netplay`. Those tests open local TCP sockets and verify the native proxy token before any game data is accepted.

Reference APIs: [Ebitengine mobile](https://ebitengine.org/en/documents/mobile.html), [Android Bluetooth permissions](https://developer.android.com/develop/connectivity/bluetooth/bt-permissions), [RFCOMM connections](https://developer.android.com/develop/connectivity/bluetooth/connect-bluetooth-devices).
