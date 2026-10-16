# POWER64 Linux partition RFE

`emulators/power64.rfe` supplies the architectural CPU and the POWER adapter
for the shared recompiler: cached interpretation, scalar IR, native code,
checked native memory, linked dispatch and cyclic regions. It is shared by the
`linux-power64-machine.rfe` single-vCPU PAPR-style partition. Verified PPC64le
boots include Linux 6.6.1 with 4 KiB logical pages and the stock Alpine kernel
below with 64 KiB logical pages. Both use the hashed MMU with 4 KiB hardware
page-table entries and a freestanding ELFv2 PID 1. The CPU retains explicit
big- and little-endian state; a big-endian Linux boot is not yet verified.

No QEMU execution is hidden behind the RFE. QEMU is an independent reference
for the boot inputs only. The partition constructs the device tree, firmware
trampoline and optional newc root directly in Go, in memory. Guest `sc` enters
the real Linux syscall handler; only partition `sc 1` calls reach the virtual
hypervisor. Console messages from PID 1 are guest execution, not host emulation
of Linux syscalls.

## Verified boot

A fresh RFE machine has booted the unmodified Linux 6.6.1 kernel, executed the
freestanding PID 1 through Linux, reported `Linux 6.6.1 ppc64le`, printed
`RENVO-POWER-LINUX: PASS`, and powered off through Linux/RTAS. The explicit
`TestLinuxBoot` gate passed again after the privilege and cross-page fixes.
The same kernel/root fixture also passed an independent QEMU POWER8 boot.

The test uses 256 MiB RAM, a 4 MiB hypervisor-owned hash table (pft-size 22),
256 MiB segments and 4 KiB pages. A larger hash table than the initial 1 MiB
prototype leaves space for mappings beyond the 65,536 bolted RAM pages.
The original interpreter boot remains a correctness control. A separate fresh
native-engine boot and real Linux/CoreMark run now pass the full-stack gates
below. There is no QEMU delegation in either RFE execution.

### Stock Alpine fixture

Alpine v3.20's `main/ppc64le` package `linux-virt-6.6.142-r0` provides an
unmodified POWER8 kernel with release `6.6.142-0-virt`:

- APK: <https://dl-cdn.alpinelinux.org/alpine/v3.20/main/ppc64le/linux-virt-6.6.142-r0.apk>
- APK SHA-256: `095e93ea6cbe6f89688fcf77c0af6e0a18bfeb0d3aa08f800bed759f066a6f5c`
- Payload: `boot/vmlinuz-virt`, a direct ELF64 little-endian POWER executable.
- Payload SHA-256: `a6ccc1e0f23b37c637df7aa486e95d1d1a1311721eaeb2afae2b29490870d35e`

TinyRangeX's native APK reader supplies this payload directly in memory.
Its 64 KiB Linux logical pages use the partition's advertised 4 KiB hash
page-table entries. The kernel enables AltiVec; its `copypage_power7` path
needs `lvx` and `stvx` when loading the larger CoreMark ELF. These instructions
retain architectural vector registers in the CPU and execute through the
checked memory bus. The scalar native ABI remains 38 words.

This stock kernel passed the RFE tiny-init boot and poweroff proof with zero
compilation failures or arena resets. The same kernel and 80,000-iteration
CoreMark ELF passed independent QEMU POWER8 reference validation for both
seeds and all ten CRCs, then powered off. A fresh strict RFE 80,000-iteration
run also passed the CRCs, host/guest duration checks and poweroff with zero
compilation failures or arena resets. It failed the performance requirement:
the first seed scored 1,737.602 iterations/s against the required 4,000. These
observations use a new GCC 11.2.1 fixture and Alpine/Go 1.25.10 environment;
they do not replace the historical Linux 6.6.1 results below. Normal tests
still require explicitly supplied fixtures and do not download packages.

## Build the Linux 6.6.1 development fixture

Normal tests do not download or build an operating system. Build this fixture
explicitly with a POWER cross toolchain and the ordinary Linux build tools.
Linux 6.6.1 is intentionally fixed, not a request for the latest kernel:

- Source: `https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-6.6.1.tar.xz`
- SHA-256: `da1ed7d47c97ed72c9354091628740aa3c40a3c9cd7382871f3cedbd60588234`
- Configuration: `tools/power64-linux/minimal.config`
- Tested kernel compiler: `powerpc64le-linux-gnu-gcc-13` (GCC 13).
  GCC 15 failed the unmodified Linux 6.6.1 `fs/readdir.c` build with invalid
  DS-form displacements; do not change the emulator to work around that.

After independently obtaining and verifying the source archive, run in its
source checkout (substitute an absolute path to this repository's config):

```sh
make ARCH=powerpc CROSS_COMPILE=powerpc64le-linux-gnu- \
  KCONFIG_ALLCONFIG=/path/to/renvo/tools/power64-linux/minimal.config allnoconfig
make -j2 ARCH=powerpc CROSS_COMPILE=powerpc64le-linux-gnu- \
  CC=powerpc64le-linux-gnu-gcc-13 vmlinux
```

The configuration supports up to two CPUs because that is Linux's minimum for
this SMP platform; the device tree exposes only CPU 0. Keep the generated
`vmlinux` unmodified. The fixture uses only the documented Linux and guest-init
sources.

From the Renvo checkout, build the guest PID 1:

```sh
mkdir -p sandbox/power64-linux
powerpc64le-linux-gnu-gcc-13 -O2 -static -nostdlib -fno-builtin \
  -fno-stack-protector -fno-pie -no-pie -mcpu=power8 -mno-altivec \
  -mno-vsx -msoft-float -fno-tree-vectorize -Wl,-e,_start \
  -Wl,--build-id=none tools/power64-linux/start.S \
  tools/power64-linux/init.c -o sandbox/power64-linux/init

go run ./cmd/renvoemu build -o sandbox/power64-linux/machine \
  emulators/linux-power64-machine.rfe
sandbox/power64-linux/machine -kernel /path/to/linux/vmlinux \
  -init-elf sandbox/power64-linux/init
```

`-init-elf` creates a complete newc root in memory, including `/dev/console`.
Alternatively `-initrd` accepts an explicitly supplied complete initramfs.
The two options are mutually exclusive. `-initramfs-out OUTPUT` with
`-init-elf` writes a standalone boot archive and exits without booting; this
allows an independent emulator to consume exactly the same root image.

The guest smoke must print `RENVO-POWER-LINUX: INIT`, report the real kernel's
`uname` results, print `RENVO-POWER-LINUX: PASS`, and power off through Linux
and RTAS. A banner, timeout, unsupported instruction, or panic is not a pass.

## Current native result: above 4,000 iterations/s

At the stopping point, a fresh, unprofiled strict run with the stock Alpine
kernel and unchanged **80,000-iteration** scalar GCC 11.2.1 ELF measured:

| Seed | Guest seconds | Host marker interval | Iterations/second | Final CRC |
|---|---:|---:|---:|---|
| Performance | 17.268410 | 17.271804123 | **4,632.737** | `cc42` |
| Validation | 17.139243 | 17.142491030 | **4,667.651** | `35f3` |

Both complete CRC sets, minimum ten-second durations, host/guest time
agreement, actual native execution and orderly Linux poweroff passed.
Compilation failures and arena resets were zero. The unchanged extended gate
requires at least 4,000 iterations/second for each seed and passed; the
subsequent **6,000 iterations/second goal was not reached**.

These measurements use a one-vCPU Linux/amd64 Alpine workbench with Go
1.25.10, a 256 MiB POWER guest, 65,536-instruction native sessions,
4,096-instruction monotonic clock polling, 64-instruction static regions and
the explicitly configured 128 MiB POWER native-code cache. They are not a
same-resource comparison with the historical 8 MiB/64-instruction polling
measurements below. Checked memory remains the default. Profiling was disabled
for the strict proof; sampled diagnostic runs are not performance results.

The guest ELF SHA-256 is
`318e506bf767aa8f5cb44d7fcc0ea2395228a0c38cd585cb376b42d3b888cbbe`.
The kernel hash and pinned upstream CoreMark revision are recorded above and
below. No workload, clock accounting, CRC reference, minimum duration or gate
threshold was changed during the final optimization pass.

Reproduce with explicitly supplied fixtures (normal tests do not download them):

```sh
RENVO_POWER_ENGINE=native \
RENVO_POWER_LINUX_KERNEL=/absolute/path/to/alpine-vmlinux \
RENVO_POWER_LINUX_RELEASE=6.6.142-0-virt \
RENVO_POWER_COREMARK_EXTENDED_ELF=/absolute/path/to/coremark80000.elf \
  go run ./cmd/renvoemu test -run '^TestLinuxCoreMarkExtendedPerformance$' \
  emulators/linux-power64-machine.rfe
```

The final pass reduces checked successor dispatch and translation overhead,
preserves warmed regions at short scheduling tails, admits bounded pure
conditional regions, and simplifies scalar/condition lowering. Countdown loops
reuse exact arithmetic zero flags and derive completed iterations from the
original instruction budget only after architectural state restoration.
Regression coverage includes full-width predicates, raw private-body progress,
permissions, root changes, fault prefixes, timer expiry, exact budgets and
mixed leaf/region transitions.

## Tests and diagnostics

```sh
go run ./cmd/renvofmt -check emulators/power64.rfe \
  emulators/linux-power64-machine.rfe
go run ./cmd/renvoemu test emulators/linux-power64-machine.rfe
RENVO_POWER_LINUX_KERNEL=/path/to/linux/vmlinux \
RENVO_POWER_LINUX_INIT=/absolute/path/to/init \
  go run ./cmd/renvoemu test -run TestLinuxBoot emulators/linux-power64-machine.rfe
```

Fixture tests expect the exact kernel release `6.6.1` by default. When selecting
another explicitly supplied kernel, set `RENVO_POWER_LINUX_RELEASE` to its exact
`uname -r` value, for example `6.6.142-0-virt` for the Alpine kernel fixture.
This applies to `TestLinuxBoot` and every Linux CoreMark gate. An explicitly
empty value is rejected; the release must be one printable ASCII token of at
most 64 bytes. The guest must print exactly one matching uname report. Changing
the expected release does not change CRC references, timing checks, required
scores, instruction limits or native resources. Native fixture proofs require
zero compilation failures. Arena resets emit a warning and remain visible in
the native statistics; recycling costs remain part of the measured runtime.

`-steps` bounds execution. `-progress N` emits a compact diagnostic every N
instructions. `-trace-after N -trace-count M` enables a bounded trace on
stderr, not an unbounded trace file. An unsupported instruction stops with PC,
MSR, and the actual instruction word. It must be implemented from architecture
semantics, not skipped or identified by a guest address.

`-jit-profile` explicitly enables native symbol maps and monotonic JIT-dump
support for diagnostic sampling. It is disabled by default and independent of
`-stats`; profiled timings are not performance proofs.

## CoreMark inside Linux

The unchanged EEMBC CoreMark revision
`d5fad6bd094899101a4e5fd53af7298160ced6ab` has run as PID 1 inside this guest.
Both standard seed sets pass upstream validation at 1,000 iterations with the
2000-byte, single-context stack dataset. The observed realtime intervals were
25.780918 seconds (performance) and 25.806727 seconds (validation), about
38.8 iterations/second on the hosted interpreter, not physical POWER hardware.
Final CRCs `d340` / `26c2` match the independent identical-ELF QEMU control.
That short QEMU control is not a QEMU performance score.

See `emulators/coremark-power64-port/README.md` for the exact GCC 13.4.0
scalar ELFv2 build and `power64-coremark-results.json` for fixture hashes and
measured output. No upstream algorithm or validation/duration check is changed.
The guest uses real Linux write/clock_gettime/uname/reboot syscalls. The port
prints PASS only after both complete upstream validations succeed.

```sh
sandbox/power64-linux/machine -kernel /path/to/linux/vmlinux \
  -init-elf /absolute/path/to/coremark.elf -clock realtime
RENVO_POWER_LINUX_KERNEL=/path/to/linux/vmlinux \
RENVO_POWER_COREMARK_ELF=/absolute/path/to/coremark.elf \
RENVO_POWER_COREMARK_ITERATIONS=2000 \
  go run ./cmd/renvoemu test -run TestLinuxCoreMark emulators/linux-power64-machine.rfe
```

The legacy references support 1,000 and 2,000 iterations with their original
2B-attempt limit. They may finish too quickly on the optimized native tier:
that is an **invalid score**, not permission to relax the duration gate. The
separately selected 20,000-iteration performance proof below keeps enough real
work for valid timing and requires at least 1,000 iterations/s for both seeds.
Never change the duration check or use deterministic
time to obtain a score. The gate checks all five CRCs
for each seed, ordered START/STOP markers, guest and host durations of at least
ten seconds, agreement within 5% or 100 ms, and orderly guest poweroff. It
rejects bad final CRCs, short runs, duplicate/incomplete output and false-success
exit codes. Normal tests do not download or build these fixtures.

`-clock deterministic` remains the compatibility default for correctness
work. `-clock realtime` uses the shared monotonic oscillator independently of
retirement; TB reads, decrementer expiry and H_CEDE all use that same clock.
Instruction/resource limits remain independent. The synthetic DTB processor
frequency must not be used for CoreMark/MHz or physical hardware comparisons.

## Measured interpreter improvement

A sequential, unprofiled comparison on the same one-CPU Linux/amd64 development
sandbox used the identical 2,000-iteration ELF, Linux kernel, 256 MiB partition,
Go 1.26.0 and GCC 13.4.0 toolchain, with realtime clock mode:

| Seed | Before (`cf3860a3`) | Fast paths | Throughput before → after | Speedup |
| --- | ---: | ---: | ---: | ---: |
| Performance | 50.874347 s | 19.254827 s | 39.31 → 103.87 iterations/s | 2.64× |
| Validation | 53.025505 s | 19.286204 s | 37.72 → 103.70 iterations/s | 2.75× |

Both complete guest runs validated and powered off. The identical ELF under
QEMU POWER8 independently matched every seed/list/matrix/state/final CRC,
including the 2,000-iteration final CRCs `4983` / `0cac`. That QEMU run was below
ten seconds and is **only a CRC control**, not a performance score. The fresh
RFE gate also checks host/guest elapsed-time agreement and the ten-second
minimum, with the original two-billion-instruction limit unchanged.

`power64-interpreter-performance.json` records the comparison and fixture
hashes. These results are for the hosted interpreter, not physical POWER
hardware or parity with the ARM64/x86 native tiers. The original 1,000-iteration
measurement above remains historical evidence, not the before side of this
comparison.

## Interpreter fast paths and profiling

Realtime interrupt polling is bounded by 64 attempted instructions. A transition
that enables MSR.EE refreshes the oscillator before the next interruptible
instruction; timebase/decrementer reads and writes and H_CEDE sample it
synchronously. Polling does not create elapsed time or advance the oscillator
from instruction retirement. The deterministic mode is unchanged.

A bounded positive translation cache skips repeated SLB/PTEG searches, not
protection checks. Every hit checks the selected live SLB/HPTE and mutation
epochs, then uses live privilege, key, PP0/PP, NX, RPN and reference/change bits.
SLB instructions and hash-table hypercalls invalidate lookup precedence. RAM
accesses remain checked, cross-page accesses retain precise translation faults,
and fetches read current instruction bytes (there is no decoded-code cache).
Rotate masks use constant-time edge masks instead of per-bit loops.

`-cpuprofile OUTPUT` optionally writes a Go CPU profile of guest execution. For
example, add it to the realtime command above, then inspect the standalone
profile with `go tool pprof -top /path/to/machine OUTPUT`. Profiling runs are
**diagnostics**, not before/after performance comparisons: compare identical
ELFs, kernel, clock mode, host and toolchain with profiling disabled.

## Full shared recompiler stack

The machine CLI defaults to `-engine native`; `-engine ir` and
`-engine interpreter` select the lower-tier controls. On a host without a native
emitter the shared engine can fall back to its lower tiers, but an explicit
native fixture proof requires `NativeAvailable()` and real native retirement.
`New`/`Step` remain interpreter-compatible APIs; `EnableJIT` plus `RunQuantum`
provides bounded accelerated execution. `Close` releases executable storage.
The CPU library exposes `Architecture`, `NewEngine`, and a stable `TierCPU`
state ABI with explicit import/export instead of unsafe CPU-layout assumptions.

Scalar integer/CR/XER, branches/CTR/LR, shifts/rotates, multiplication/division,
loads/stores, update forms and byte reversals lower to shared IR. The machine
uses `systemmemory.RAM` for checked descriptors and code versions. Full-width
addresses are bank-guarded into reversible 48-bit keys; cross-bank data accesses
exit before the access. With MSR.DR clear, memory keys first apply the same
real-mode physical alias mask as the interpreter; full architectural EAs remain
unchanged, and enabling DR cannot reuse that real-mode specialization.
Endian-separated code identities permit guarded endian
specialization. Native refills probe permissions **without** setting HPTE R/C;
first accesses and cleared bits return to architectural execution. SLB/HPT
mutations invalidate mappings; CPU, boot and RTAS writes use physical code
barriers, including aliases. External raw RAM edits must call
`NativeRAM.BeforeWrite` before mutation; external HPT/SLB edits must call
`InvalidateMappings`. The trusted, single-threaded native-memory contract does
not authorize arbitrary unnotified changes to public backing arrays.

On Linux/amd64 with the supported foreign boundary, native state and validated
pins stay resident for at most **65,536 instructions**, while real monotonic
timer deadlines are checked before entry and at intervals of at most **4,096
retired guest instructions** inside admitted native sessions. This polling
cadence is independent of the static region width and exact retirement ledger.
The deadline is the earlier of DEC expiry and a 1 ms host
refresh boundary. This avoids millions of Go/cgo re-entries without making the
clock retirement-based. Custom clocks, aliases and unsupported hosts retain
the original short boundary. Slow instructions, exceptions and bank
transitions synchronize immediately. Deterministic slices stop at decrementer
expiry. Realtime counter accesses and cede remain synchronous, and the POWER
oscillator includes host compilation time rather than pausing it out of scores.
Regions retain their 64-instruction static size bound; cold kernel native
admission is conservative. The POWER machine explicitly configures a **128 MiB
native-code cache**. The shared engine default remains 8 MiB; instruction,
session, frame, guest RAM and duration budgets are unchanged. The separately
selected performance input has its own explicit resource profile, not a larger legacy/default limit.

Use the explicit fixtures above to require actual full-stack execution:

```sh
sandbox/power64-linux/machine -kernel /path/to/linux/vmlinux \
  -init-elf /absolute/path/to/coremark.elf -engine native -clock realtime -stats
RENVO_POWER_ENGINE=native \
RENVO_POWER_LINUX_KERNEL=/path/to/linux/vmlinux \
RENVO_POWER_LINUX_INIT=/absolute/path/to/init \
RENVO_POWER_COREMARK_PERFORMANCE_ELF=/absolute/path/to/coremark20000.elf \
  go run ./cmd/renvoemu test emulators/linux-power64-machine.rfe
```

The fixture gates require positive native, native-memory, linked, region and
loop counters and zero compilation failures, in addition to the existing CRC,
ordered output, host/guest time agreement, duration and orderly shutdown checks.
Differential tests cover scalar edge cases and register aliases in interpreter,
IR and native code; native BE/LE memory, unaligned/update/reverse forms, code
identity and aliased code writes, R/C/permissions, precise cross-page exits,
reservation slow paths and fixed timer/resource boundaries.

## Native performance: above 1,000 iterations/s

The initial full-stack integration (`2dea23f0`) measured only 83.00/90.82
iterations/s. Those historical measurements remain in
`power64-recompiler-results.json`. Profiling identified short native sessions,
pointer-graph setup and tail-budget returns as the dominant costs. Bounded
foreign timer sessions and real-mode physical alias lowering remove those costs
without changing CoreMark, skipping guest operations or inventing time.

A fresh, unprofiled Linux partition on a one-CPU Linux/amd64 development sandbox
ran **20,000 iterations per seed** with the unchanged 2000-byte single-context
scalar GCC 13.4.0 input:

| Seed | Guest realtime | Host marker interval | Iterations/s from ticks | Final CRC |
| --- | ---: | ---: | ---: | --- |
| Performance | 16.748164 s | 16.752516432 s | **1194.16** | `382f` |
| Validation | 14.891526 s | 14.892898941 s | **1343.05** | `d304` |

Both upstream validations passed and Linux powered off. Every seed/list/matrix/
state/final CRC matched an independent identical-ELF/kernel QEMU POWER8 control.
The short QEMU control is **CRC-only**, not a valid performance score. The native
run retired 13,340,410,854 native instructions (2,800,795,661 native memory),
13,340,403,350 linked instructions and 943,180,817 loop iterations, with zero
compilation failures. The executable arena stayed within its original 8 MiB cap.

The integer-only integration port rounds printed seconds and iterations/s.
The table and gate use **iterations / (Total ticks / 1,000,000)** instead, and
verify host marker timing independently. No printed, rounded score is accepted
in place of the duration/CRC/time-agreement proof.

Build the same port/algorithms using the README command, changing only
`-DITERATIONS=20000`, then select the explicit performance profile:

```sh
RENVO_POWER_ENGINE=native \
RENVO_POWER_LINUX_KERNEL=/path/to/linux/vmlinux \
RENVO_POWER_COREMARK_PERFORMANCE_ELF=/absolute/path/to/coremark20000.elf \
  go run ./cmd/renvoemu test -run '^TestLinuxCoreMarkPerformance$' \
  emulators/linux-power64-machine.rfe
# Equivalent bounded standalone run (default -steps remains 2B):
sandbox/power64-linux/machine -kernel /path/to/linux/vmlinux \
  -init-elf /absolute/path/to/coremark20000.elf -engine native -clock realtime \
  -steps 16000000000 -stats
```

This separately selected gate caps attempts at 16B and requires **>=1000
iterations/s for EACH seed**, all ten independent CRCs, host AND guest durations
>=10 seconds, agreement within max(100 ms, 5%), real native memory/link/region/
loop coverage, zero compilation failures and orderly shutdown. Existing legacy
1000/2000 fixtures, default 2B attempt limit and compiler performance/resource
gates are unchanged. The legacy 2000 interpreter gate and fresh native Linux init
also passed, as did focused deadline/GC/alias/revocation and POWER differential
regressions, formatting and Renvo preflight. Full validation remains with CI.

`power64-recompiler-performance.json` records provenance and measured counters.
These are hosted native-emulation measurements, not physical POWER hardware,
EEMBC certification or an ARM64/x86 throughput-parity claim.

### Historical 20,000-iteration follow-up

The earlier timer-frame optimization pass did not reach 4,000 iterations/s.
Its fresh, unprofiled strict 20,000-iteration gate measured:

| Seed | Guest realtime | Host marker interval | Iterations/s from ticks |
| --- | ---: | ---: | ---: |
| Performance | 14.840206 s | 14.841815123 s | **1347.69** |
| Validation | 13.461432 s | 13.462889796 s | **1485.73** |

These are about 12.9% / 10.6% above the preceding measurements, not a claimed
4,000-iteration result. Both upstream validations and all ten independent
identical-ELF/kernel QEMU CRC comparisons passed. The QEMU control remains
CRC-only: both of its roughly 3.1-second runs correctly fail upstream duration
validation and must not be reported as performance results.

This pass removes dominated immutable SSA guards and redundant shift/mask
operations, uses word-width compare flags directly, and keeps the generated
foreign dispatcher frame resident across genuine monotonic timer polls.
Prepared successor predictions now work across mixed leaf families, retaining
full-PC, fresh-epoch and exact-budget checks. Shared write facts and timer
budgets occupy disjoint frame words; the original total frame cap is unchanged.
Optional fault-protected RAM gains root-cache reuse and register-result direct
loops with precise recovery, but **checked Go RAM remains the default** because
the direct experiment did not improve both seed sets.

The final proof includes 13,337,566,714 native instructions, 2,799,194,749 native
memory operations and 942,475,216 loop iterations, with zero compilation
failures or arena resets. Fresh native Linux init, the unchanged 2000-iteration
interpreter gate, affected package and POWER differential tests, the focused
Linux backend regression, formatting and Renvo preflight passed. No algorithm,
clock, duration/CRC rule, default resource limit or compiler performance policy
was relaxed. `power64-recompiler-followup.json` records the complete counters
and provenance; the preceding result is retained for comparison.

## Current boundaries

The scalar CPU now uses the full shared IR/native stack. Vector/VSX, floating
point and transactional memory remain unsupported. Privileged operations,
interrupts, atomics/reservations, devices and precise memory faults retain
architectural slow paths; unsupported instructions still fail explicitly.
The partition has a checked RAM bus, a hypervisor-owned hash page table,
256 MiB segments with 4 KiB pages, terminal polling, selectable deterministic/realtime timebase
and decrementer, and a small RTAS surface. It is not a general PowerVM
implementation, multiprocessor machine, network/storage platform, or security
sandbox. Unsupported partition calls report `H_FUNCTION`; unsupported CPU or
MMU features stop explicitly rather than silently succeeding.
