# AArch64 Linux user emulator

The authoritative guest adapters are `emulators/aarch64.rfe` (CPU, decoding and
IR lowering) and `emulators/linux-arm64-user.rfe` (AArch64 ELF/register/trap
conventions and host services). Shared tiering lives in `internal/rfe/engine`;
checked memory, ELF loading and Linux syscall services live in
`internal/rfe/linuxuser`. The archive suite exercises the adapters alongside
the shared packages' own tests.

## Running

```sh
go run ./cmd/renvoemu build -o sandbox/linux-arm64-user emulators/linux-arm64-user.rfe
sandbox/linux-arm64-user -engine native -stats guest.elf [args...]
```

`-engine` selects `interpreter`, `ir` or `native`. Native mode includes fallback
to the lower tiers. `-steps` is an absolute retirement ceiling (default 100 million).
`-region-instructions` bounds optimized regions (default/max 256);
`-native-instructions` bounds supported native sessions (default/max 65,536).
Cold discovery remains limited to 16 instructions.

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

## Guest architecture boundary

`engine.Architecture` supplies the state shape, PC slot, instruction alignment,
execute-checked fetch/decode and IR lowering. `engine.CPU` supplies stable views
of state, memory and retirement, plus the architectural interpreter. The shared
controller owns tier promotion, bounded caches, region discovery and dispatch.
Instructions carry explicit PCs and byte lengths; budgets count instructions,
not bytes. Code-page dependencies include every byte of the last instruction,
including page-straddling instructions. Unversioned validation uses the guest's
fetch callback, not an assumed byte order. Native admission retains full PC tags
and host entry alignment; guest instruction alignment stays with the adapter.

`linuxuser.ELFABI` describes machine identity, accepted flags and entry extent.
`linuxuser.SyscallABI` maps syscall registers and recognizes traps. The current
services implement the common Linux syscall-number slice used by this AArch64
personality; other ABIs must translate differing numbers and data layouts.
The host supplies secure startup entropy and realtime/monotonic clocks; the
portable services neither substitute weak randomness nor use instruction ticks.

Shared IR has total signed/unsigned division and remainder at 32/64-bit widths:
zero divisors produce all-ones quotients or the dividend remainder; signed
MIN/-1 produces MIN with remainder zero. Results are zero-extended bit patterns.
Guest adapters explicitly implement differing policies (AArch64 chooses a zero
quotient on division by zero) and any required result sign extension.

A synthetic two-register guest tests mixed two-/four-byte instructions,
halfword targets, straddling-page invalidation, precise faults and native region
budgets. This is preparation for another ISA, **not a RISC-V implementation**:
a new guest still needs its decoder, interpreter, lowering and ABI adapter.

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
