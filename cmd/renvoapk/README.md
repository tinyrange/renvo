# renvoapk

`renvoapk` turns either the Android CompilerJIT `librenvo.so` or a
JVM-backend-generated `classes.dex` into an installable APK. The tool is itself
compiled with Renvo and does not invoke Gradle, AAPT, D8, the Android SDK, or
the NDK.

It generates:

- a binary `AndroidManifest.xml` declaring `android.app.NativeActivity`;
- `lib/arm64-v8a/librenvo.so` as an uncompressed native library;
- a deterministic ZIP32 APK container; and
- an APK Signature Scheme v2 development signature using RSA/SHA-256.

Build and run it with:

```sh
go run ./cmd/renvo -t linux/amd64 -s \
  -o sandbox/renvoapk ./cmd/renvoapk

sandbox/renvoapk \
  -so path/to/librenvo.so \
  -config path/to/app.conf \
  -o path/to/app.apk
```

For the Java-backed Android target, emit DEX directly and select `-dex` instead:

```sh
go run ./cmd/renvo -backend backends/jvm.rbe \
  -t android/vm32 -s -o classes.dex path/to/program

sandbox/renvoapk \
  -dex classes.dex \
  -config path/to/app.conf \
  -o path/to/app.apk
```

The config format is one `key=value` pair per line:

```text
package=dev.renvo.example
name=Renvo Example
version_code=1
version_name=1.0
min_sdk=24
target_sdk=35
orientation=portrait
```

`min_sdk` cannot be lower than 24 because the packager deliberately emits the
modern v2 signature without the older JAR/v1 signature. The current payload is
AArch64-only. `orientation` is optional and may be `portrait` or `landscape`.

## Development signing key

The built-in certificate is intentionally a public development identity. Its
SHA-256 fingerprint is:

```text
D9:A1:C7:30:73:A4:58:32:8C:E9:90:A2:57:56:DE:2D:
D6:A4:66:22:E3:AD:2E:79:68:25:CD:83:E3:14:05:E7
```

This makes local builds deterministic and lets `adb install -r` update an
earlier development build with the same package name. It is not a private
release identity and must not be used to publish applications. Configurable
production keys are a separate signing milestone.

## Validation

The tests independently verify the v2 signing block, certificate/public-key
match, RSA signature, whole-APK content digest, ZIP entries, and typed binary
manifest. They also compile and execute `renvoapk` with Renvo and require its
output to match the host builder byte-for-byte:

```sh
go test ./internal/apk ./internal/backendjit \
  -run 'TestBuildProducesVerifiedV2NativeActivityAPK|TestRenvoBuiltAPKPackagerMatchesHostBuilder'
```

The generated APK has been installed and launched on a physical ARM64 Android
15 device. To repeat the device check:

```sh
adb install -r path/to/app.apk
adb shell am start -W -n dev.renvo.example/android.app.NativeActivity
```

The shared Forms example under `examples/forms_hello` paints through the
NativeActivity-backed Android graphics port. Build `librenvo-forms.so` with the
commands in that example's README, then pass it to `renvoapk` as the `-so`
payload.
