#!/usr/bin/env python3
"""Check native APK packaging and 16 KiB ELF segment alignment."""
import pathlib
import platform
import subprocess
import sys
import tempfile
import zipfile

apk, ndk = map(pathlib.Path, sys.argv[1:])
host = "darwin-x86_64" if platform.system() == "Darwin" else "linux-x86_64"
readelf = ndk / "toolchains/llvm/prebuilt" / host / "bin/llvm-readelf"
badging = subprocess.check_output([ndk.parent.parent / "build-tools/36.0.0/aapt", "dump", "badging", apk], text=True)
for expected in ["package: name='com.olivierh.bubblebobble'", "sdkVersion:'23'", "targetSdkVersion:'36'", "native-code: 'arm64-v8a'", "launchable-activity: name='com.olivierh.bubblebobble.MainActivity'"]:
    assert expected in badging, f"Missing APK metadata: {expected}"
with zipfile.ZipFile(apk) as archive, tempfile.TemporaryDirectory() as work:
    libraries = [item for item in archive.infolist() if item.filename.endswith(".so")]
    assert libraries, "APK has no native libraries"
    for item in libraries:
        assert item.filename.startswith("lib/arm64-v8a/"), item.filename
        assert item.compress_type == zipfile.ZIP_STORED, "Native libraries must be uncompressed"
        local = pathlib.Path(work) / "library.so"
        local.write_bytes(archive.read(item))
        elf = subprocess.check_output([readelf, "-Wl", local], text=True)
        loads = 0
        for line in elf.splitlines():
            parts = line.split()
            if not parts:
                continue
            if parts[0] == "LOAD":
                loads += 1
                assert int(parts[-1], 16) >= 16384, line
                assert int(parts[1], 16) % 16384 == int(parts[2], 16) % 16384, line
            if parts[0] == "GNU_RELRO":
                assert (int(parts[2], 16) + int(parts[5], 16)) % 16384 == 0, line
        assert loads, "Missing ELF load segments"
        print(f"Verified ARM64 / 16 KiB: {item.filename}")
