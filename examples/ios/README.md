# iOS CompilerJIT target

`ios_arm64.rtg` is the compiler-side iOS port. It reuses the shared AArch64 and
Darwin lowering through an external CompilerJIT definition, then specializes
the generated Mach-O contract for iOS. Preparing or validating the compiler
target does not require Xcode.

The current image contract provides:

- a PIE AArch64 Mach-O executable
- `LC_BUILD_VERSION` for iOS 13.0
- 16 KiB Mach-O segments without writable/executable overlap
- the ordinary `/usr/lib/libSystem.B.dylib` runtime binding
- an ad-hoc SHA-256 code directory refreshed after the iOS header is written
- `ios`, `darwin`, `unix`, `aarch64`, and `arm64` frontend build tags

Installing on a physical device still requires an application bundle,
entitlements, provisioning, and an Apple-issued signature. UIKit lifecycle and
the forms/event-loop adapter are also later layers; this target establishes and
tests the compiler artifact underneath them.

Run the host-independent validation with:

```sh
go test ./internal/backendjit -run TestCompilerJITIOSARM64MachOImage
```

For a local compiler smoke test:

```sh
go run ./cmd/renvo \
  -backend examples/ios/ios_arm64.rtg \
  -t ios/arm64 \
  -s -o sandbox/renvo-ios \
  internal/backendjit/testdata/mobile_entry.go
```
