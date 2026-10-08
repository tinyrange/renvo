# RFE emulators

These are directly authored text `.rfe` packages. Their embedded Go and
instruction-lowering sections are the source of truth; builds extract temporary
files and do not require separately maintained emulator Go packages.

| Package | Role |
| --- | --- |
| `pdp11.rfe` | Shared PDP-11 CPU, FP11 floating point, MMU, interpreter and native lowering |
| `v7-user.rfe` | V7 a.out loader and syscall personality; requires `pdp11` |
| `pdp11-machine.rfe` | PDP-11 machine, console, clock and RK05 disks; requires `pdp11` |
| `aarch64.rfe` | Initial A64 integer CPU, interpreter, pure IR, native scalar memory and bounded native linking |
| `linux-arm64-user.rfe` | Initial static-ELF Linux AArch64 personality; requires `aarch64` |

From the Renvo checkout, with the Go toolchain installed:

```sh
go run ./cmd/renvofmt -w emulators
go run ./cmd/renvofmt -check emulators
go run ./cmd/renvoemu test emulators/pdp11.rfe
go run ./cmd/renvoemu test emulators/v7-user.rfe
go run ./cmd/renvoemu test emulators/pdp11-machine.rfe
go run ./cmd/renvoemu build -o sandbox/v7-user emulators/v7-user.rfe
go run ./cmd/renvoemu build -o sandbox/pdp11-machine emulators/pdp11-machine.rfe
```

Run a host a.out file directly, or import a V7 filesystem from an RK05 image:

```sh
sandbox/v7-user ./guest.aout argument
sandbox/v7-user -disk /path/to/v7.dsk /bin/echo hello
sandbox/v7-user -disk /path/to/v7.dsk /bin/sh -c 'echo hello | /bin/cat'
sandbox/pdp11-machine -script examples/pdp11/tests/v7.script /path/to/v7.dsk
```

Guest filesystem and disk writes are private to the process. The input disk
image and host directory imported with `-root` are never written back.
`-steps` bounds guest execution; `-engine interpreter`, `-engine ir`, and
`-engine native` select execution, with native the default. `-stats` reports
native coverage and cache use. The clock measures abstract instruction ticks,
not cycle-accurate PDP-11 timing.

Cross-build, for example:

```sh
go run ./cmd/renvoemu build -target windows/amd64 -o sandbox/v7-user.exe emulators/v7-user.rfe
```

Native block execution supports macOS arm64 and Linux/Windows amd64 and arm64.
The emulator executable is currently built by host Go; its hot guest blocks
are emitted by Renvo's backend. Portable IR and interpreter modes do not need
executable memory. This does not yet make the complete emulator self-hosted.

The V7 personality supports 0407, 0410 and 0411 executables, cooperative
fork/exec/wait, pipes, basic files/directories, and V7 signals. Handlers receive
the V7 PC/PSW stack frame and return through the guest trampoline's RTI.
`kill`, `alarm`, `pause`, ignored signals, default termination, signal wait
status and `EINTR` for blocked reads, host/pipe writes and waits are implemented.
Host SIGINT, SIGQUIT and SIGTERM are forwarded to the guest foreground group.
Alarms use deterministic virtual time at one million scheduler ticks per
second; idle waits advance to the next alarm. `time` and `times` use that same
clock. This is not wall-clock timing, and idle timer advancement counts against
the execution budget.

FP11 supports F (24-bit significand) and D (56-bit significand) arithmetic,
rounding/chopping, integer and precision conversions, six accumulators, FPS,
FEC/FEA and floating exceptions. These operations execute in the architectural
interpreter in all engine modes; integer register blocks can still run natively.
The implementation retains D's low bits without converting through host
`float64`. Full-system mode dispatches floating traps through vector 0244;
user mode delivers SIGFPE.

It is not a complete Unix kernel: job control, core-file generation, mounting
and arbitrary device ioctls remain unsupported. Unsupported syscalls return
EINVAL. Full-system mode runs the guest V7 kernel's syscall and signal code.
FP11 timing and model-specific maintenance modes are not emulated; the separate
FIS instruction option of other PDP-11 models is not part of this PDP-11/45.

Tests normally use synthetic images. To additionally exercise historical V7
executables and boot the guest kernel, supply a decompressed
[TUHS V7 RK05 image](https://www.tuhs.org/Archive/Distributions/Boot_Images/v7_rk05_1145.gz):

```sh
RENVO_PDP11_V7_IMAGE=/absolute/path/to/v7.dsk go test ./internal/rfe -run TestEmulatorSourcePackages -count=1
```

The optional tests run shell pipelines, create/read/remove a guest file, catch
shell signals, wake `sleep` through an alarm, and run `awk` arithmetic and its
square-root, exponential and logarithm library routines. They also check that
disk input remains unchanged. No test downloads an operating system.
The CPU tests compare generated Go and IR execution for every accepted lowering
encoding with all condition-code combinations, and use literal architectural
vectors to check their shared ALU rules. They also check native/IR agreement, MMU
boundaries, traps, code modification and instruction budgets.
Floating tests cover literal DEC-format results, the 56th significand bit,
rounding ties, conversions, addressing, exception masks and kernel trap entry.
Signal tests cover handler return, disposition inheritance across fork/exec,
bad stacks, interrupted host reads without lost input, blocked host writes with
stable buffers, and trap-to-signal mapping.

See [the RFE format and runtime](../docs/rfe.md) for authoring and extension.

## AArch64 user mode

The initial `aarch64.rfe` / `linux-arm64-user.rfe` pair implements an integer
subset and static ET_EXEC Linux user mode. It provides interpreter, portable IR
and native pure-block tiers, including multiply, conditional select/compare and
expanded integer operations. Exact instruction budgets and versioned code-page
guards preserve invalidation and execute permissions; custom memories without
version stamps retain instruction-byte revalidation. A pinned freestanding
CoreMark port now passes CRC checks in all tiers; guarded interpreter-only
entries reduce redundant fetches. It is **not yet a complete AArch64 emulator
or a demonstrated QEMU performance win**. QEMU is far ahead in current timings.

```sh
go run ./cmd/renvoemu test emulators/linux-arm64-user.rfe
go run ./cmd/renvoemu build -o sandbox/linux-arm64-user emulators/linux-arm64-user.rfe
sandbox/linux-arm64-user -engine native -stats ./static-aarch64-guest argument
```

Dynamic linking, TLS startup, FP/ASIMD, guest signals and threads are not yet
supported. Guest memory is capped at 64 MiB; only a documented syscall slice is
implemented, and unsupported syscalls return ENOSYS. The default retirement
ceiling is 100 million instructions, configurable with `-steps`. Real clocks
are used for elapsed timing. Guest syscall numbers are never passed through to
the host kernel. These checks are not a security sandbox.

See [the detailed AArch64 design and milestone plan](../docs/aarch64-user.md)
for instruction coverage, ABI limits, optimizer architecture and the CoreMark
benchmark contract. See [validation evidence and outstanding workflows](../docs/aarch64-validation.md)
for the pinned port, raw CoreMark diagnostics and remaining performance gates.


On Linux, `-stats` also names native blocks and the bounded dispatcher in a
private, exclusively created `/tmp/perf-PID.map` for perf report symbolization.
The map is retained after process exit; `profile_errors` counts nonfatal output
failures. AArch64's native path now uses SSA register allocation, compact memory
guards, immediate/variable-shift/conditional-move selection and register-resident
dispatch. Legacy calls retain a 64-instruction ceiling; supported Linux/amd64+cgo
foreign sessions have a separate ceiling of 65,536 instructions. See
`../docs/aarch64-validation.md` for diagnostic measurements and limitations.
