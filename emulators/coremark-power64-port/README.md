# Freestanding Linux PPC64le CoreMark port

Original ELFv2 integration code for the unchanged EEMBC CoreMark sources at
revision `d5fad6bd094899101a4e5fd53af7298160ced6ab`. The upstream benchmark is
not vendored here. Preserve its LICENSE.md, README.md and copyright notices
with the source. No normal test downloads a benchmark or operating system.

This ELF runs as `/init` inside the real Linux guest. `sc` invokes Linux's
PPC64 system-call handler; it is not a host syscall personality. The port uses
write, clock_gettime(CLOCK_MONOTONIC), uname and reboot. It handles PPC's
CR0.SO/positive-errno convention and performs orderly Linux/RTAS poweroff.

The startup runs the standard 2000-byte, single-context stack workload twice:
performance seeds `(0,0,0x66)`, then validation seeds `(0x3415,0x3415,0x66)`.
The volatile seeds, algorithms, CRC checks and duration checks are unchanged.
The wrapper prints PASS only after upstream validates both complete runs;
upstream's `main` return value by itself is not an acceptance check.

## Explicit build

Provision the five `core_*.c` files and `coremark.h` from the pinned revision
unchanged in a development directory. Copy this port's `.c`, `.h` and `.S`
files into its `port/` subdirectory. From that source directory:

```sh
powerpc64le-linux-gnu-gcc-13 -O2 -static -nostdlib -fno-builtin \
  -fno-stack-protector -fno-pie -no-pie -mcpu=power8 -mno-altivec \
  -mno-vsx -msoft-float -fno-tree-vectorize -I. -Iport \
  -DITERATIONS=2000 -DPERFORMANCE_RUN=1 -Wl,-e,_start \
  -Wl,--build-id=none core_list_join.c core_main.c core_matrix.c \
  core_state.c core_util.c port/core_portme.c port/start.S -o coremark.elf
```

Every translation unit uses the same flags. `-O2` is deliberate; vector/VSX,
FP, libc and dynamic runtime dependencies are outside the current POWER RFE
scope. Do not change upstream code to avoid an unsupported instruction: fix
its architectural implementation or report the gap.

Use the Linux kernel/config described in `docs/power64-linux.md`:

```sh
sandbox/power64-linux/machine -kernel /path/to/linux/vmlinux \
  -init-elf /absolute/path/to/coremark.elf -engine native -clock realtime -stats
```

Timing markers are outside each measured interval. `Total ticks` are
microseconds. Integer-mode seconds and rates truncate; calculate diagnostic
rates from ticks, not the printed integer rate. Each seed set must run for at
least ten seconds and validate. Short runs, deterministic-clock runs, partial
output, errors, a boot banner or process exit zero alone are not valid results.
Increase iterations explicitly if a faster host cannot satisfy the duration;
never add a sleep to the timed workload or fabricate clock ticks.

The deterministic mode remains useful for small instruction/CRC diagnostics,
not performance reporting. The DTB's synthetic processor-frequency property
is not a host CPU frequency and must not be used to report CoreMark/MHz.
The results measure the selected hosted execution tier plus guest kernel and
emulator overhead, not physical POWER8 hardware. Use `-engine interpreter` for
the interpreter control and `-engine native -stats` for full-stack execution;
`RENVO_POWER_ENGINE=native` makes the fixture gate require native memory,
linked dispatch and regions as well as correct CRCs and timing.

Expected seed/list/matrix/state CRCs are:

- Performance: `e9f5 / e714 / 1fd7 / 8e3a`.
- Validation: `18f2 / e3c1 / 0747 / 8d84`.

The current interpreter performance fixture uses 2,000 iterations per seed,
with final CRCs `4983` / `0cac` independently verified under QEMU on the identical
ELF. The historical 1,000-iteration profile has final CRCs `d340` / `26c2`.
Set `RENVO_POWER_COREMARK_ITERATIONS=2000` alongside the explicit ELF/kernel
paths when running `TestLinuxCoreMark`; its two-billion-instruction bound and
ten-second host/guest duration checks remain unchanged.

The final CRC depends on iterations. Compare it using the identical ELF under
an independent emulator, without rebuilding between runs. Keep independent
QEMU control logs separate from RFE execution proof. A short QEMU CRC control
is not a QEMU performance score.
