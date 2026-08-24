# renvoflash

`renvoflash` is available as both a host-Go bootstrap and a Renvo-built tool.
Its implementation lives in the reusable `internal/espflash` package. It
converts a Renvo ESP32-C6 or
ESP32-S3 ELF32 executable to the
Espressif application-image format, writes the factory application partition at
flash offset `0x10000`, resets the board, and briefly displays its serial output.
It can also accept an already converted ESP application `.bin` file. The
command invokes no external flashing or debug tool.

It can also make a read-only, whole-flash backup of an ESP32-C6, ESP32-S3, or
ESP32-P4. Before reading, backup mode refuses secure-download mode and flash
encryption, configures the detected flash geometry, and calculates a device-side
MD5. The printed digest must match the host file's MD5 before the backup is used.

```sh
sandbox/renvo -o sandbox/renvoflash ./cmd/renvoflash
./sandbox/renvoflash firmware.elf
./sandbox/renvoflash --backup factory.bin

# The host-Go bootstrap exposes the same library and command behavior.
go run ./cmd/renvoflash --probe-jtag
```

When no port is supplied, the command detects a single USB serial device under
`/dev`; it refuses an ambiguous match. An explicit Linux `/dev/ttyACM*` or
`/dev/ttyUSB*`, or macOS `/dev/cu.usbmodem*` or `/dev/cu.usbserial*`, may be
passed as the final argument. On Linux the user must have read/write permission
for the serial device (commonly through the `dialout` or `uucp` group). Use
`--convert ELF BIN` to produce an application image without flashing it.
On boards whose native USB port does not expose usable DTR/RTS reset control,
enter ROM download mode manually before running the command.

Only the application region is replaced. The bootloader, partition table, NVS,
and eFuses are not written.

## ESP32-C6 JTAG and hot reload

Linux hosts can claim the ESP32-C6 built-in USB/JTAG interface directly through
usbfs. No OpenOCD, GDB, libusb, or ESP-IDF installation is used. Probe it with:

```sh
go run ./cmd/renvoflash --probe-jtag
```

Hot-reload programs must use the SRAM-linked backend. An ordinary
`esp32c6/riscv32` ELF executes from mapped flash and is rejected by the JTAG
loader.

```sh
go build -o sandbox/renvo ./cmd/renvo
sandbox/renvo \
  -backend backends/esp32c6_jtag.rtg \
  -t esp32c6-jtag/riscv32 \
  -s -o sandbox/hotreload.elf \
  ./examples/m5nanoc6/hotreload

go run ./cmd/renvoflash --jtag sandbox/hotreload.elf
go run ./cmd/renvoflash --watch sandbox/hotreload.elf
```

The same direct transport provides basic debugger controls. Inspection commands
temporarily halt a running core and restore its prior running state; explicit
control and mutation commands leave the state shown in their output.

```sh
go run ./cmd/renvoflash --jtag-status
go run ./cmd/renvoflash --jtag-regs
go run ./cmd/renvoflash --jtag-read 0x40824000 64
go run ./cmd/renvoflash --jtag-halt
go run ./cmd/renvoflash --jtag-step
go run ./cmd/renvoflash --jtag-set-reg a0 0x1234
go run ./cmd/renvoflash --jtag-set-pc 0x40824100
go run ./cmd/renvoflash --jtag-blink 6 250
go run ./cmd/renvoflash --jtag-resume
```

The host-controlled LED demo halts the core and drives the M5NanoC6's GPIO7
indicator directly through JTAG. It turns the LED off and restores the core's
previous running state when complete:

```sh
./examples/m5nanoc6/host-blink.sh

# Use a Renvo-built host executable and an explicit usbfs device.
RENVOFLASH=./sandbox/renvoflash \
  ./examples/m5nanoc6/host-blink.sh 10 100 /dev/bus/usb/003/005
```

Memory reads are word aligned and limited to 4096 bytes per CLI invocation.
The reusable `USBJTAG` API also exposes `State`, `ReadPC`, `ReadRegister`,
`ReadRegisters`, `WriteRegister`, `ReadMemory`, and `Step`.

`--jtag` loads one image and exits. `--watch` keeps USB/JTAG claimed and retains
the previous image, polling the ELF every 10 ms and writing only changed
word-aligned ranges. The reusable frontend boundary is:

```go
debug, err := espflash.OpenESP32C6JTAG("")
session := espflash.NewHotReloadSession(debug)
report, err := session.Update(elf)
```

For an in-process edit loop, keep both the frontend/backend objects and JTAG
session alive. `BeginFSBuildSession` accepts the external `-backend` option, so
the frontend can retain package artifacts while compiling directly into the
session:

```go
build := driver.BeginFSBuildSession(args, root, stdRoot, cache, fs, true)
for !build.Step() {}
compiled := driver.CompileBuildResult(build.Result(), backend)
report, err := session.Update(compiled.Binary)
```

The session halts the core, writes the changed SRAM words through RISC-V system
bus access, executes `fence.i`, sets `dpc` to the new entry, and resumes. A board
reset or power cycle requires `session.Reset()` so the next update writes the
complete image.

Device programs can compose libraries through `renvo.dev/device/app`:

```go
func main() {
    app.Run([]app.Component{display, sensors, controls})
}
```

Every component's `Setup` runs once in slice order, followed by repeated `Loop`
calls in the same order. A hot reload restarts the linked program and therefore
runs `Setup` again; it does not preserve BSS or heap state.

The SRAM-linked target deliberately trades a small amount of space for stable
edit deltas. It sorts newly reachable functions independently of source call
order and ends each function at a 256-byte boundary. A small function growth
can then overwrite its existing padding without moving later functions or
changing their call relocations. This layout policy is not used by persistent
flash images.

The sub-50 ms objective applies to a warm frontend cache plus a small JTAG
delta. Cold compilation, initial SRAM loading, USB enumeration, and full flash
erase/write are intentionally outside that latency budget.
