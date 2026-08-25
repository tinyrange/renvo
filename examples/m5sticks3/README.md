# M5StickS3 example

This target combines Renvo's shared Xtensa LX7 call0 backend with the
ESP32-S3 memory map, watchdog handoff, USB Serial/JTAG output, and ESP
application-image contract. The backend is loaded from RTG at compile time;
it does not depend on ESP-IDF-generated application code.

The application expects a standard ESP32-S3 second-stage bootloader at flash
offset `0x0`. Its small IRAM bootstrap accepts the bootloader's windowed-ABI
handoff, selects the chip's normal instruction-cache mode through the ESP32-S3
ROM, clears the complete final BSS, and then enters flash-mapped Renvo code.

The emulator oracle compiles and runs the complete freestanding single-file
frontend suite. That suite covers integer semantics, control flow, arrays,
slices, strings, runes, structs, pointers, methods, closures, function values,
interfaces, unsafe operations, and builtins. Its only system operation is
`print`, and success is exactly `PASS\n` on USB Serial/JTAG.

The `forms_menu` application renders the ordinary Forms controls through the
built-in 135x240 display. It uses the target's `tiny` asset profile and a 3x
logical surface to fit comfortably in internal RAM without changing the Forms
API. The side button advances through its scrolling list and the front button
selects the highlighted item.

## Emulator oracle

Clone `tinyrange/renvo_emu` beside this repository, or set
`RENVO_EMU_ROOT`, then run:

```sh
./examples/m5sticks3/test-emulator.sh
```

`RENVO` and `RENVO_EMU` may point at prebuilt executables. The test runs the
ESP32-S3 Xtensa core and the modeled hardware USB Serial/JTAG registers; it
does not use the emulator's compiler-only UART facade.

## Build and flash

Build Renvo, then compile the same suite for the board:

```sh
go build -o sandbox/renvo ./cmd/renvo
sandbox/renvo \
  -backend backends/esp32s3.rtg \
  -t esp32s3/xtensa_lx7 -tags m5sticks3 \
  -o sandbox/m5sticks3-suite.elf \
  ./frontend_tests/single_file_microcontroller
```

Convert the ELF and replace only the factory application at flash offset
`0x10000`:

```sh
./examples/m5sticks3/flash.sh sandbox/m5sticks3-suite.elf /dev/ttyACM0
```

Set `ESPTOOL` if the command is not named `esptool.py`. The helper preserves
the existing bootloader, partition table, NVS, and eFuses. It does replace the
application at `0x10000`; back up the board first if its factory application
matters. Holding the side reset button until its green indicator flashes puts
the StickS3 into ROM download mode when automatic reset is unavailable.
The helper requires esptool 5 or newer because it uses the ESP32-S3-aware
watchdog reset to leave the USB flasher and start the application reliably.

To build and flash the interactive Forms example, use the same commands with
`sandbox/m5sticks3-forms.elf` as the output and
`./examples/device/forms_menu` as the input package.
