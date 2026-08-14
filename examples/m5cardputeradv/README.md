# M5Stack Cardputer Adv example

The Cardputer Adv board package supports its 240x135 ST7789V2 display and its
56-key keyboard. The display uses the ESP32-S3 SPI3 peripheral with GDMA. The
keyboard driver talks to the onboard TCA8418 over its internal I2C bus and
translates the controller's 7x8 wiring into the printed 4x14 layout, including
Shift and Fn layers.

The hardware oracle draws a landscape border, alignment grid, and color bars.
Each keyboard transition colors its physical cell in a miniature 4x14 keyboard
map and writes the translated printable character over USB serial. This makes
display bounds, orientation, color order, matrix remapping, Shift/Fn handling,
and keyboard event delivery independently observable.

The `terminal` application is a 20-column by 6-row local terminal with a
one-line key inspector. Printable keys use the Shift layer, Ctrl+letter emits
the corresponding C0 byte, and Fn supplies Escape, Delete, function keys, and
cursor arrows. It implements Backspace/Delete, Tab, CR, LF, vertical tab, form
feed, bell indication, scrolling, caret notation for otherwise unhandled C0
characters, and the common ANSI cursor and erase sequences.

## Build and flash

Build Renvo and compile the oracle with the shared ESP32-S3 target:

```sh
go build -o sandbox/renvo ./cmd/renvo
sandbox/renvo \
  -backend backends/esp32s3.rtg \
  -t esp32s3/xtensa_lx7 \
  -o sandbox/m5cardputeradv-oracle.elf \
  ./examples/m5cardputeradv/oracle
```

Replace the final package with `./examples/m5cardputeradv/terminal` and choose
an output such as `sandbox/m5cardputeradv-terminal.elf` to build the terminal.

Convert the ELF and replace only the factory application at flash offset
`0x10000`:

```sh
./examples/m5cardputeradv/flash.sh sandbox/m5cardputeradv-oracle.elf /dev/ttyACM0
```

Set `ESPTOOL` if the command is not named `esptool.py`. The helper preserves
the existing bootloader, partition table, NVS, and eFuses, but replaces the
factory application. Holding BtnG0 while resetting the Cardputer Adv enters ROM
download mode if automatic reset is unavailable.
