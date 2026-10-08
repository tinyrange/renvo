# AArch64 Linux user emulator

The authoritative sources are `emulators/aarch64.rfe` (CPU and tier controller)
and `emulators/linux-arm64-user.rfe` (memory, ELF loader and Linux personality).
Their embedded tests are exercised by the repository's RFE archive suite.

## Running

```sh
go run ./cmd/renvoemu build -o sandbox/linux-arm64-user emulators/linux-arm64-user.rfe
sandbox/linux-arm64-user -engine native -stats guest.elf [args...]
```

`-engine` selects `interpreter`, `ir` or `native`. Native mode includes fallback
to the lower tiers. `-steps` is an absolute retirement ceiling (default 100 million).
`-region-instructions` bounds optimized regions (default/max 256);
`-native-instructions` bounds supported native sessions (default/max 65,536).
Cold discovery remains limited to 16 instructions. `-native-chains` enables
experimental direct chaining and is off by default.

## Supported scope

- Little-endian A64 integer operations, flags, direct/indirect branches,
  scalar loads/stores and register pairs. Unsupported instructions trap.
- Static ET_EXEC AArch64 ELF with mapped program headers, bounded arguments,
  a 1 MiB initial stack, empty environment and conservative auxiliary vector.
- Guest mappings are capped at 64 MiB in a 48-bit address space.
- Syscalls: exit/exit_group; write to stdout/stderr; realtime/monotonic
  clock_gettime; virtual getpid/gettid; brk; anonymous private mmap without
  fixed addresses/hints; munmap; mprotect. Other syscalls return ENOSYS.
- No FP/ASIMD, atomics, dynamic linking, TLS startup, signals or guest threads.

The memory and engine are single-threaded. Mapping/permission changes and
executable writes invalidate native links through address-space identities and
page generations. Custom unversioned memories use instruction-byte validation.
Memory faults preserve the completed instruction prefix. Native failures and
resource exhaustion retain architectural fallback rather than bypassing checks.
RFE packages and generated host code are trusted; this is **not a security sandbox**.

## Native runtime and diagnostics

Linux/amd64+cgo supports pinned foreign sessions. Other supported native hosts,
no-cgo builds and aliasing contexts use conservative short calls. Native arm64
emission exists but has not been hardware-validated in this change.
See `rfe-native-abi.md` for ownership, frame and admission invariants.

`-stats` reports tier/cache counters and enables Linux perf-map symbols.
Profiling files are private, exclusively created and retained for offline use;
failures increment `profile_errors`. Profiling does not alter guest semantics.

The original freestanding CoreMark integration and reproduction instructions
are in `emulators/coremark-port/` and `emulators/coremark-host-port/`.
Local benchmark observations belong in the PR, not in permanent development logs.
