# Alpine Linux boot fixtures

`prepare` creates direct-boot inputs for the RFE Linux machines without root
privileges. The checked-in `inputs.json` pins Alpine 3.24.2 minirootfs and virt
ISO bytes for AArch64 and x86-64 using the official release SHA-256 values.
The kernel comes from the verified ISO, not the mutable netboot directory.

```sh
./tools/rfe-linux/prepare all --download
# Subsequent preparation is offline by default:
./tools/rfe-linux/prepare all
python3 -B -m unittest discover -s tools/rfe-linux -p 'test_*.py'
```

Requirements: Python 3.11 or later and `xorriso`. Downloads happen only when
`--download` is present. `--out DIR` selects a different output/cache directory.
Normal package tests never download an operating system.

Each `sandbox/rfe-linux/ARCH/` directory contains:

- `kernel`: the x86 bzImage or decompressed raw AArch64 Linux Image.
- `kernel.original`: the unmodified kernel extracted from the pinned ISO.
- `initramfs.cpio.gz`: Alpine minirootfs plus the checked-in `/init` script.
- `cmdline`: serial console and `/init` settings for direct boot.
- `manifest.json`: input/output hashes, architecture, version and success marker.

The tool reads the tar archive without extracting it to the host filesystem.
It rejects unsafe paths and non-directory parents, preserves Unix modes and
numeric ownership, normalizes timestamps, assigns stable inodes, sorts paths,
and emits Linux newc hardlinks with the data in the final link record. Gzip has
no timestamp or filename. No Alpine package scripts run on the host.

PID 1 mounts dev/proc/sys/tmp, checks the Alpine version and architecture,
executes a child shell, creates/reads/removes a file, and runs a pipe. It prints
`RENVO-RFE-LINUX: INIT`, `EXEC PASS`, `FILE PASS`, `PIPE PASS` and finally
`RENVO-RFE-LINUX: PASS` (each with the same prefix), then powers off. Add
`rfe.shell=1` to the kernel command line to open a shell after the checks.

The optional RFE integration gate is `./tools/check rfe-linux` or
`./tools/rfe-linux/check`. It explicitly downloads missing pinned inputs and
requires both RFE machines to execute the guest checks and shut down.

For an independent fixture check, QEMU can boot exactly the same files:

```sh
qemu-system-aarch64 -machine virt,accel=tcg -cpu cortex-a57 -m 512M -smp 1 \
  -display none -monitor none -serial stdio -no-reboot \
  -kernel sandbox/rfe-linux/aarch64/kernel \
  -initrd sandbox/rfe-linux/aarch64/initramfs.cpio.gz \
  -append 'console=ttyAMA0,115200 rdinit=/init panic=-1 nokaslr'
qemu-system-x86_64 -machine pc,accel=tcg -cpu qemu64 -m 512M -smp 1 \
  -display none -monitor none -serial stdio -no-reboot \
  -kernel sandbox/rfe-linux/x86_64/kernel \
  -initrd sandbox/rfe-linux/x86_64/initramfs.cpio.gz \
  -append 'console=ttyS0,115200 rdinit=/init panic=-1 nokaslr'
```

These commands validate the Alpine inputs under QEMU. They are not evidence
of RFE execution. Keep their logs named `qemu-reference.log` separately from
the RFE gate's `rfe-boot.log`.

## Cached system CoreMark measurement

`python3 -B tools/rfe-linux/measure ARCH` measures one existing, hash-verified
100,000-iteration-per-seed comparison fixture. It never downloads, rebuilds, or
modifies benchmark inputs. It requires a fresh `rfe.build` binary/stamp and the
saved comparison metadata under `sandbox/rfe-linux/coremark-system-compare-v1`.
It fixes affinity to logical CPU 0, RAM to 256 MiB, cache to balanced32, and
performance-only limits to 900 seconds / one trillion instructions. Original
Linux correctness gates keep their own unchanged limits.

The tool timestamps serial START/STOP markers using the host monotonic clock,
validates both seed/final CRC sets and minimum durations, checks unchanged
inputs, and writes unique serial/stderr/result files beneath the fixture runs
directory. A nonzero exit or invalid/incomplete output is not a score.
`--check` validates archived parsing and rejection cases without launching a
guest. The approved agent interface is `rfe.benchmark(arch, check=False)`.

## Hosted versus deterministic clocks

The machines accept `-clock deterministic` (legacy default) and `-clock realtime`.
Realtime uses a shared host-monotonic oscillator, excluding vCPU-stopped cold
translation and arena reclamation. Time is independent of retirement. Counter
reads have native scheduling boundaries, and idle waits do not retire work.
Choose the clock before JIT construction.

The cached CoreMark helper explicitly selects realtime and retains its original
balanced32/CPU0/256 MiB inputs and performance-only limits. The default-cache
Linux correctness gate remains unchanged. Realtime with the small legacy
kernel cache has not passed the original boot deadline. See
`docs/system-clock-repair.md` for measured gains and exact supported scope.

`rfe.kernel_code(address)` reads a bounded code window and available symbols
from the checksum-verified cached x86 kernel. It does not execute a kernel,
invoke disassemblers, or accept file paths or commands.
