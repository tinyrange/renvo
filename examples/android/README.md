# Android CompilerJIT target

[`android_arm64.rtg`](../../backends/android_arm64.rtg) is the compiler-side
Android port. It prepares through the
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
graphics port registers the NativeActivity window and input-queue lifecycle,
dispatches touch input, and renders at native resolution. `RendererAuto` uses
an EGL/OpenGL ES 2 frame backend when it is available, while
`RendererSoftware` retains the portable Renvo surface and
`ANativeWindow_lock`/`ANativeWindow_unlockAndPost` path. APK packaging is
provided by `cmd/renvoapk`.

Run the host-independent validation with:

```sh
go test ./internal/backendjit -run TestCompilerJITAndroidARM64NativeActivityImage
```

For a local compiler smoke test:

```sh
go run ./cmd/renvo \
  -backend backends/android_arm64.rtg \
  -t android/arm64 \
  -s -o sandbox/librenvo.so \
  internal/backendjit/testdata/mobile_entry.go
```

Compile the Forms hello example instead with:

```sh
go run ./cmd/renvo \
  -backend backends/android_arm64.rtg \
  -t android/arm64 \
  -s -o sandbox/librenvo-forms.so \
  ./examples/forms_hello
```

The full control gallery uses a 360 × 800 dp layout, requests OpenGL ES
explicitly, renders at the device's native density, and demonstrates touch
targets, text fields, selection controls, list scrolling, sliders, split-view
dragging, themes, and advanced controls:

```sh
go run ./cmd/renvo \
  -backend backends/android_arm64.rtg \
  -t android/arm64 \
  -s -o sandbox/librenvo-controls.so \
  ./examples/forms_controls
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

This flow has been verified through EGL/OpenGL ES 2 rendering, full TrueType
text, taps, list scrolling, and control dragging on a physical ARM64 Android 15
device. The explicit software renderer remains available for portability and
uses native-resolution buffer lock/post. Android uses the same retained Forms
controls as the desktop and browser targets.

See [`cmd/renvoapk/README.md`](../../cmd/renvoapk/README.md) for the config and
signing contracts.
