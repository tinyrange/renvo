# POWER64 Linux partition RFE

`emulators/power64.rfe` supplies the architectural CPU and the POWER adapter
for the shared recompiler: cached interpretation, scalar IR, native code,
checked native memory, linked dispatch and cyclic regions. It is shared by the
`linux-power64-machine.rfe` single-vCPU PAPR-style partition. The verified
configuration boots a PPC64le Linux kernel with the hashed MMU, 4 KiB pages,
and a freestanding ELFv2 PID 1. The CPU retains explicit big- and little-endian
state; a big-endian Linux boot is not yet verified.

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

## Build the development fixture

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

## Tests and diagnostics

```sh
go run ./cmd/renvofmt -check emulators/power64.rfe \
  emulators/linux-power64-machine.rfe
go run ./cmd/renvoemu test emulators/linux-power64-machine.rfe
RENVO_POWER_LINUX_KERNEL=/path/to/linux/vmlinux \
RENVO_POWER_LINUX_INIT=/absolute/path/to/init \
  go run ./cmd/renvoemu test -run TestLinuxBoot emulators/linux-power64-machine.rfe
```

`-steps` bounds execution. `-progress N` emits a compact diagnostic every N
instructions. `-trace-after N -trace-count M` enables a bounded trace on
stderr, not an unbounded trace file. An unsupported instruction stops with PC,
MSR, and the actual instruction word. It must be implemented from architecture
semantics, not skipped or identified by a guest address.

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

The automated references support 1,000 and 2,000 iterations. Build the current
performance fixture with `-DITERATIONS=2000`, and supply the explicit iteration
environment variable above; the historical 1,000-iteration fixture may be too
short on a faster host. Never change the duration check or use deterministic
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
exit before the access. Endian-separated code identities permit guarded endian
specialization. Native refills probe permissions **without** setting HPTE R/C;
first accesses and cleared bits return to architectural execution. SLB/HPT
mutations invalidate mappings; CPU, boot and RTAS writes use physical code
barriers, including aliases. External raw RAM edits must call
`NativeRAM.BeforeWrite` before mutation; external HPT/SLB edits must call
`InvalidateMappings`. The trusted, single-threaded native-memory contract does
not authorize arbitrary unnotified changes to public backing arrays.

Native state stays resident across linked/region side exits only inside the
unchanged **64-attempt** timer window. Slow instructions, exceptions and bank
transitions synchronize immediately. Deterministic slices stop at decrementer
expiry. Realtime counter accesses and cede remain synchronous, and the POWER
oscillator includes host compilation time rather than pausing it out of scores.
Regions are bounded to the same schedulable window; cold kernel native admission
is conservative. The original instruction/arena/duration budgets are unchanged.

Use the explicit fixtures above to require actual full-stack execution:

```sh
sandbox/power64-linux/machine -kernel /path/to/linux/vmlinux \
  -init-elf /absolute/path/to/coremark.elf -engine native -clock realtime -stats
RENVO_POWER_ENGINE=native \
RENVO_POWER_LINUX_KERNEL=/path/to/linux/vmlinux \
RENVO_POWER_LINUX_INIT=/absolute/path/to/init \
RENVO_POWER_COREMARK_ELF=/absolute/path/to/coremark.elf \
RENVO_POWER_COREMARK_ITERATIONS=2000 \
  go run ./cmd/renvoemu test emulators/linux-power64-machine.rfe
```

The fixture gates require positive native, native-memory, linked, region and
loop counters and zero compilation failures, in addition to the existing CRC,
ordered output, host/guest time agreement, duration and orderly shutdown checks.
Differential tests cover scalar edge cases and register aliases in interpreter,
IR and native code; native BE/LE memory, unaligned/update/reverse forms, code
identity and aliased code writes, R/C/permissions, precise cross-page exits,
reservation slow paths and fixed timer/resource boundaries.

The final real guest retired **1,291,919,236 native instructions**, including
**272,920,788 native memory instructions**, with **1,291,883,631 linked
instructions**, **7,980 regions**, **86,570,758 loop iterations** and **zero
compilation failures**. Both standard seed sets validated at 2,000 iterations;
all ten CRCs matched an independent identical-ELF QEMU POWER8 control. The
fresh full-stack fixture gates and Renvo preflight passed.

This is functional stack integration, not a claim of ARM64/x86 throughput
parity. In the same-fixture unprofiled comparison, inclusive-clock native runs
took 24.096318/22.021507 seconds (83.00/90.82 iterations/s); interpreter controls
took 24.113729/23.624208 seconds (82.94/84.66 iterations/s). The earlier
interpreter results are historical measurements on another fixture build.
Native dispatch/scheduling overhead remains significant; no large new speedup
is claimed. See `power64-recompiler-results.json` for provenance and counters.

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
