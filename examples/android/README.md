# Android CompilerJIT target

`android_arm64.rtg` is the compiler-side Android port. It prepares through the
same external CompilerJIT path as the ESP32 targets and emits a position-
independent AArch64 `librenvo.so`; compiling it does not require the Android
NDK.

The image contract is deliberately small:

- ELF64 `ET_DYN` for AArch64 with 16 KiB-compatible load segments
- separate non-writable code and non-executable data segments
- a non-executable stack
- `librenvo.so` as the SONAME
- `ANativeActivity_onCreate` as a global function export
- `android`, `linux`, `unix`, `aarch64`, and `arm64` frontend build tags

The exported NativeActivity entry runs `appMain()` and returns to Android. The
graphics port registers NativeActivity window-created/window-destroyed
callbacks, renders into a Renvo software surface, and presents it through
`ANativeWindow_lock`/`ANativeWindow_unlockAndPost`. APK packaging is provided by
`cmd/renvoapk`.

Run the host-independent validation with:

```sh
go test ./internal/backendjit -run TestCompilerJITAndroidARM64NativeActivityImage
```

For a local compiler smoke test:

```sh
go run ./cmd/renvo \
  -backend examples/android/android_arm64.rtg \
  -t android/arm64 \
  -s -o sandbox/librenvo.so \
  internal/backendjit/testdata/mobile_entry.go
```

Compile the compact Forms example instead with:

```sh
go run ./cmd/renvo \
  -backend examples/android/android_arm64.rtg \
  -t android/arm64 \
  -s -o sandbox/librenvo-forms.so \
  ./examples/android/forms_hello
```

Build the SDK-free packager with Renvo, create a development-signed APK, and
install it with ADB:

```sh
go run ./cmd/renvo -t linux/amd64 -s \
  -o sandbox/renvoapk ./cmd/renvoapk

sandbox/renvoapk \
  -so sandbox/librenvo-forms.so \
  -config examples/android/app.conf \
  -o sandbox/renvo-forms.apk

adb install -r sandbox/renvo-forms.apk
adb shell am start -W \
  -n dev.renvo.example/android.app.NativeActivity
```

This flow has been verified through native buffer lock/post on a physical ARM64
Android 15 device. The Android Forms build is intentionally a compact first
profile: it provides the retained Form root, background invalidation, software
painting, and NativeActivity presentation. General controls, text rendering,
input dispatch, accessibility, and resize/redraw event delivery remain future
Android layers; the full desktop and browser Forms profiles are unchanged.

See [`cmd/renvoapk/README.md`](../../cmd/renvoapk/README.md) for the config and
signing contracts.
