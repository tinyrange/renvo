# M5NanoC6 examples

This example is the first freestanding microcontroller target built entirely
through an external RTG definition. The target combines the shared RV32IM
machine definition with the ESP32-C6 memory map, watchdog handoff, and ELF app
image contract. It drives the M5NanoC6 blue LED on GPIO7 without ESP-IDF.

The `button_rgb` example also drives the board's GPIO20 WS2812 through the
ESP32-C6 RMT peripheral. Pressing the active-low GPIO9 button chooses a new
color from the ESP32-C6 hardware RNG, mixed with the timing of the press.

The `blink_mixed` example demonstrates a bidirectional Go/C package. Its Go
entrypoint calls a C11 blink loop, and that C loop calls small Go adapters for
the typed board LED and clock capabilities. It is available in the web editor
as the `blink_mixed` NanoC6 example, where both source files can be edited and
flashed together.

The `air_quality` example reads an SGP30 from the Grove I2C connector once per
second. It displays TVOC on the RGB LED as a continuous
green-to-orange-to-red scale. Magenta means the sensor could not be initialized
or a measurement failed its I2C/CRC checks.

The target is intentionally loaded with `-backend`: it exercises the same JIT
preparation path available to custom boards and does not advertise itself as a
compiled-in host target.

## Emulation oracle

Clone `tinyrange/renvo_emu` beside this repository, or set
`RENVO_EMU_ROOT`, then run:

```sh
./examples/m5nanoc6/test-emulator.sh
```

The test compiles the oracle with Renvo and asks the ESP32-C6 emulator to stop
on the first rising edge of `board.esp32c6.chip_gpio.pin7`. It deliberately
does not request a bus log because long bus logs are expensive and the signal
edge is the assertion under test.

## Build and flash

Build Renvo and the blinking application:

```sh
go build -o sandbox/renvo ./cmd/renvo
sandbox/renvo \
  -backend backends/esp32c6.rtg \
  -t esp32c6/riscv32 \
  -o sandbox/m5nanoc6-blink.elf \
  ./examples/m5nanoc6/blink
```

Build the mixed C/Go version by changing the output and package paths:

```sh
sandbox/renvo \
  -backend backends/esp32c6.rtg \
  -t esp32c6/riscv32 \
  -o sandbox/m5nanoc6-blink-mixed.elf \
  ./examples/m5nanoc6/blink_mixed
```

To build the button and RGB example instead, change the output name and final
package argument:

```sh
sandbox/renvo \
  -backend backends/esp32c6.rtg \
  -t esp32c6/riscv32 \
  -o sandbox/m5nanoc6-button-rgb.elf \
  ./examples/m5nanoc6/button_rgb
```

The SGP30 air-quality example is built in the same way:

```sh
sandbox/renvo \
  -backend backends/esp32c6.rtg \
  -t esp32c6/riscv32 \
  -o sandbox/m5nanoc6-air-quality.elf \
  ./examples/m5nanoc6/air_quality
```

Convert the ELF, write only the factory application partition, and start it:

```sh
./examples/m5nanoc6/flash.sh sandbox/m5nanoc6-blink.elf /dev/ttyACM0
```

Set `ESPTOOL` if the command is not named `esptool.py`. Do not pass
`elf2image --use-segments`: esptool must see the dedicated
`.flash.appdesc` section so it can place the ESP application descriptor at
image offset `0x20`. The commands above preserve the bootloader, partition
table, NVS, and eFuses, but they do replace the application at `0x10000`; back
up the board before flashing if its factory application matters.

The helper uses an ESP32-C6 USB-aware reset after flashing. This gives the
application enough time to disable subsequent CDC-triggered resets, avoiding
the manual power replug otherwise needed with esptool's generic hard reset.
