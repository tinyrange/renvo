# PDP-11 / Unix V7 host emulator

A C PDP-11 integer emulator compiled by **Renvo**, with a small Go host adapter
and the existing Renvo graphical terminal. The first target is a usable Unix V7
boot, not an exhaustive hardware simulator. No Arduino, JavaScript engine, host
C compiler, or external target runtime is involved in the Renvo build.

## Build and run

From the repository root on macOS ARM64:

```sh
go build -o sandbox/renvo ./cmd/renvo
sandbox/renvo cc -t darwin/arm64 -o sandbox/pdp11-host examples/pdp11/main.c
sandbox/pdp11-host /absolute/path/to/v7_rk05_1145.dsk
```

Close a running emulator before replacing its executable. The example opens a
960×640 native terminal window. Input is sent to the emulated console without
local echo; Unix controls echo, line editing, and newline conversion. Enter
sends CR once, Backspace sends DEL, Ctrl+letter sends the corresponding control
byte, and arrow keys send ANSI sequences. Guest output is also mirrored to host
stdout. Closing the window exits the emulator.

The source is intended to be portable, but **Darwin/ARM64 is the host tested so
far**. Select another supported host target explicitly and validate there before
claiming support for it. The [Tab5 ESP32-P4 adapter](../device/tab5_pdp11/README.md)
builds this same core for the device's keyboard, display, and SD card.

### Boot the tested image

Use TUHS's [v7_rk05_1145.gz](https://www.tuhs.org/Archive/Distributions/Boot_Images/v7_rk05_1145.gz),
decompressed to a local file. Images are not included in this repository, and
normal tests do not download them. Consult the archive's documentation and
applicable image terms separately from the emulator source.

At the first `@` prompt enter `boot`. At the second-stage `:` prompt enter
`rk(0,0)rkunix`. A successful boot reports `mem = 178816` and a root shell prompt.
This particular image initially uses uppercase terminal presentation; type
`STTY -LCASE` at the guest shell to disable that. These commands come from the
[TUHS boot-image instructions](https://www.tuhs.org/Archive/Distributions/Boot_Images/README).

```text
@boot
New Boot, known devices are hp ht rk rl rp tm vt
: rk(0,0)rkunix
mem = 178816
# echo hello
```

This RK05 image is a small V7 root filesystem, not a complete development
distribution. Do not interpret missing guest programs as missing CPU features.

## Disks and data safety

The positional image attaches to RK0. Additional images use `--rk1 path`
through `--rk7 path`. Each drive is an RK05: 203 cylinders, two surfaces,
12 sectors per surface, and 512 bytes per sector (2,494,464 bytes).

**All image files are read-only inputs.** Each mounted image is copied into a
private, full-sized RAM disk. Short, sector-aligned TUHS filesystem dumps are
zero-extended in RAM to supply the omitted swap area. Guest writes, file
creation, deletion, and swap operate on that working copy; they are discarded
on exit. There is currently no save/export or persistent overlay option. The
host never reformats or writes back to the supplied file.

Each mounted drive consumes approximately 2.4 MiB plus load-time allocation
overhead. The Tab5 adapter deliberately mounts only one RAM-backed RK05, loaded
once through SDMMC DMA. More drives or larger devices would need an explicit
cache/overlay design and persistent-write policy rather than assuming they fit.

## Architecture and implemented scope

- `cpu.c`: CPU registers, byte/word addressing modes, integer instruction
  decoding, EIS arithmetic, kernel/supervisor/user MMU page tables, separate
  instruction/data spaces, trap entry/return, interrupts, and 248 KiB RAM.
- `cpu.c` also contains the minimal devices: RK11/RK05 disk controller with
  DMA, console receive/transmit registers and interrupts, and KW11-L-style
  clock interrupts needed by Unix. Partial disk writes preserve untouched bytes.
- `pdp11.h`: the narrow host boundary. `host_disk` transfers sectors;
  `host_tx` emits terminal bytes. Input and clock events are fed through
  `pdp_receive` and `pdp_clock`. The core has no UI, filesystem, or board calls.
- `main.c`: bounded execution batches, clock injection, and host-event polling.
  It includes the core into one C translation unit; compile `main.c` only.
- `host.go`: `#pragma go` adapter using `device/terminal` and `std/graphics`,
  RAM-backed disk images, keyboard events, and optional scripted boot input.

The CPU executes 4,096 instruction slots between host polls. Input stays queued
until the emulated receiver can accept it. The current clock is deterministic,
one tick per 16,667 instruction slots: **guest time is not wall-clock accurate**,
and the emulator is not cycle accurate. Disk completion is synchronous.

This is the V7-tested integer subset of an 11/45-style machine. FP11/FIS,
22-bit/Unibus mapping, exhaustive MMU trap-on-access behaviour, stack-limit
red/yellow-zone recovery, full diagnostic-console registers, and exhaustive
hardware conformance are not implemented. Unsupported instructions trap rather
than pretending to execute. Double faults stop the emulator. Other operating
systems and devices (RL/RP, tape, networking, sound) are not claimed supported.

## Tests

```sh
go test ./frontend_tests -run '^TestPDP11Core$' -count=1
RENVO_PDP11_V7_IMAGE=/absolute/path/to/v7_rk05_1145.dsk \
  go test ./frontend_tests -run '^TestPDP11UnixV7Boot$' -count=1
./tools/check preflight
```

The ordinary core test compiles and runs at stage0 and self-hosted stage3. It
checks byte extension/flags, deferred operands, kernel-D trap vectors,
nonresident pages, console/clock interrupts, sector DMA, partial-write
preservation, and drive/cylinder decoding. It prints only `PASS` on success.

The optional boot test compiles the actual host executable with Renvo, boots
the external V7 image, and checks shell output from `echo` and `ls`. It is
bounded and does not invoke another emulator as an oracle.

For an unattended run:

```sh
sandbox/pdp11-host --headless --ticks 10000 \
  --script examples/pdp11/tests/v7.script /absolute/path/to/v7_rk05_1145.dsk
```

`--ticks` counts 4,096-slot host polls, not guest clock interrupts, and limits
headless runs (default 25,000). Script lines contain an output substring, a
literal tab, and text to send once that substring appears. `\r` and `\n` are
decoded in the send field. Blank lines and lines beginning `# ` are ignored.
A line beginning `#` followed by a tab matches the shell prompt. Scripts are
guest input only; they do not execute host shell commands.

Verified on the host: V7 boot, shell command execution, directory listing,
guest file creation/readback/removal, and an unchanged original disk checksum.
The decompressed test image SHA-256 is
`0c36ae193579d75c792f6983749b4933a2ea2e4727b4748faaa36ac495d62356`.

## Upstream acknowledgement

The architectural reference is Paul Nankervis's
[pdp11-js](https://github.com/paulnank/pdp11-js), pinned during development to
`605cc23eada3d831be54e537fe94307f1b80a85b`, particularly `pdp11.js` and
`iopage.js`. Their source headers permit free use provided the original author
is acknowledged in modified source. Paul Nankervis is acknowledged in the C
core as well as here. No source from the Cardputer C++ forks, their platform
libraries, or their bundled disk images was incorporated.
