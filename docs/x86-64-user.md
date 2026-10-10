# x86-64 Linux user mode

`emulators/amd64.rfe` supplies a scalar integer x86-64 CPU and lowering to the
shared RFE interpreter, IR and native tiers. `linux-amd64-user.rfe` adds the
static-ELF loader and bounded Linux-user services.

```sh
go run ./cmd/renvoemu test emulators/amd64.rfe
go run ./cmd/renvoemu test emulators/linux-amd64-user.rfe
go run ./cmd/renvoemu build -o sandbox/linux-amd64-user emulators/linux-amd64-user.rfe
sandbox/linux-amd64-user -engine native -steps 100000000 -stats ./static-guest argument
```

The CLI defaults to native execution. `-engine interpreter` and `-engine ir`
select the other tiers. `-steps` is an absolute guest-instruction limit, including
interpreter fallbacks; `-stats` reports execution and compilation counters.

## Supported scope

This is an incomplete scalar subset, not general Linux binary compatibility.
It supports integer register widths, legacy/REX prefixes, ModRM/SIB and
RIP-relative addressing, moves/extensions, LEA, arithmetic/logical operations,
shifts/rotates, comparisons and conditional operations, branches, calls and
stack operations. Selected multiply/divide and carry operations fall back to
the interpreter; byte multiply/divide and most signed-division widths are not
implemented. Unsupported encodings fail explicitly. Instruction fetch respects
executable permissions and the 15-byte length limit.

The personality translates only write, exit/exit_group, clock_gettime,
getpid/gettid, brk, mmap, munmap and mprotect to the shared services. Unknown
syscall numbers cannot alias the shared service numbering. The loader accepts
static x86-64 ELF images; it does not provide dynamic linking, TLS, signals,
threads, arbitrary filesystem calls, SIMD or x87. This is **not a security
sandbox**. Guest memory and native execution require serialized ownership.

## Shared native optimizations

Bounded branch observations guide region selection; later promotions can grow
an existing region without modifying published native code. Shared IR adds
explicit parity and arithmetic-status operations, bit-field simplification,
low-word arithmetic and constant shifts. Loop compilation defers exit-only
arithmetic, preserves partial-state inputs, and allocates cold values separately
from the hot body. Fault checkpoints and retirement budgets remain precise.

Checked memory remains the default. Native loops retain full-width address,
permission and access-width checks while reusing proven page facts. Only
explicitly registered code-versioned memory contexts may omit data-store epoch
updates. Executable writes still exit native code; mapping and protection
transitions establish fresh code generations. Generic memory contexts retain
ordinary generation updates and cannot reuse specialized linked sessions.

## Experimental direct memory

`-direct-memory` opts into fault-assisted mappings on Linux/amd64 with cgo;
unsupported hosts fall back to checked memory. It reserves up to eight 64 MiB
windows, separately from the unchanged physical guest-memory limit. Host-service
and JIT views share backing; holes are inaccessible and executable guest pages
are not writable through the JIT view. Only registered scalar native access
sites can recover host faults. Paired and unaligned stores use checked access.
The embedding application must stop engines before calling `Process.Close()`.
Signal-handler ownership changes must be serialized with initialization.

This mode remains off by default: local CoreMark measurements were slower than
the checked implementation. It is an experiment, not a recommended fast path.

## Local performance evidence

On Linux/amd64 (Intel Core i7-13620H), the unchanged scalar CoreMark workload
at 100,000 iterations per seed produced these microsecond-derived rates:

| Guest | Performance iter/s | Validation iter/s |
| --- | ---: | ---: |
| x86-64, final native engine | 7,985.29 | 7,969.63 |
| AArch64, final native engine | 6,816.51 | 6,818.96 |

Both seed CRC sets passed and all four measured seed runs exceeded ten seconds.
These are local measurements, not portable performance guarantees or a QEMU
comparison. AArch64 here is a **guest on an x86-64 host**, not ARM hardware.
The adjacent `x86-64-coremark-merge-results.json` and
`aarch64-coremark-matrix-followup-results.json` preserve the trial outputs.

The latest cold-input/low-word-copy changes improved x86 throughput about
1.6% over their interleaved control (7,860.12 / 7,848.21). Removing only those
two changes gave AArch64 6,820.18 / 6,836.78: no demonstrated AArch64 throughput
gain from that last step. The cumulative AArch64 result must not be attributed
to those two changes alone. Benchmark algorithms, ports and scored guest flags
were not changed by these optimizations; compiler resource gates are unchanged.
