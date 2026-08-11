# M5NanoC6 examples

This example is the first freestanding microcontroller target built entirely
through an external RTG definition. The target combines the shared RV32IM
machine definition with the ESP32-C6 memory map, watchdog handoff, and ELF app
image contract. It drives the M5NanoC6 blue LED on GPIO7 without ESP-IDF.

The `button_rgb` example also drives the board's GPIO20 WS2812 through the
ESP32-C6 RMT peripheral. Pressing the active-low GPIO9 button chooses a new
color from the ESP32-C6 hardware RNG, mixed with the timing of the press.

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

The C6 has no general USB controller, but its USB Serial/JTAG block documents a
raw PHY test interface. The software-USB probe first runs the portable packet
SIE and USB device stack through a complete synthetic low-speed HID
enumeration: bus reset, 8-byte endpoint-zero descriptor segmentation, address
assignment, configuration, endpoint data toggles, and one interrupt report. It
then takes over the raw PHY interface, selects the low-speed D- pull-up, and
transmits that report with NRZI, bit stuffing, CRC16, and EOP before restoring
USB Serial/JTAG:

```sh
./examples/m5nanoc6/test-usb-emulator.sh
```

Set `ESPTOOL` to an esptool 5 executable to convert the ELF and ask the
emulator to validate the resulting application image against the direct ELF in
the same run:

```sh
ESPTOOL=esptool ./examples/m5nanoc6/test-usb-emulator.sh
```

The emulator accepts output only when the report returned by the emulated SIE
is exact and the complete transmitted packet is wire-valid at 102--111
instruction ticks per bit cell. Unit tests separately round-trip data, token,
and handshake waveforms; the emulator's independent oracle accepts no packet
with a malformed PID, CRC, bit-stuffing sequence, EOP, or transmit cadence.
This qualifies the Go protocol state, target code generation, packet encoder,
C6 register takeover/restore path, deterministic transmit cadence, and the
watchdog-reset/retained-marker recovery sequence. It does not prove electrical
behavior, receiver sampling or arbitration jitter, clock calibration, host
turnaround deadlines, USB re-enumeration latency, or modem-control behavior on
the real USB-Serial-JTAG programming connection; those remain hardware tests.

The first hardware raw-PHY test is guarded by a twenty-second Timer Group 0
watchdog. If the attempt wedges, the C6 resets with USB Serial/JTAG restored and
holds a five-second recovery window before trying again. Start the recovery
flasher before resetting the board so it can automatically replace the image:

```sh
./sandbox/renvoflash --recover sandbox/m5nanoc6-blink.elf /dev/ttyACM0
```

## Build and flash

Build Renvo and the blinking application:

```sh
go build -o sandbox/renvo ./cmd/renvo
sandbox/renvo \
  -backend examples/m5nanoc6/esp32c6.rtg \
  -t esp32c6/riscv32 \
  -o sandbox/m5nanoc6-blink.elf \
  ./examples/m5nanoc6/blink
```

To build the button and RGB example instead, change the output name and final
package argument:

```sh
sandbox/renvo \
  -backend examples/m5nanoc6/esp32c6.rtg \
  -t esp32c6/riscv32 \
  -o sandbox/m5nanoc6-button-rgb.elf \
  ./examples/m5nanoc6/button_rgb
```

The SGP30 air-quality example is built in the same way:

```sh
sandbox/renvo \
  -backend examples/m5nanoc6/esp32c6.rtg \
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
