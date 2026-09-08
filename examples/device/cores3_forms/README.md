# CoreS3-SE forms demo

A touch-operated Forms demo with a tap counter, theme switch, slider, and
progress bar. Forms renders directly into a 160x120 RGB565 surface, scaled to
the 320x240 LCD. Its 38,400-byte framebuffer fits without PSRAM initialization.
The LCD accepts its native pixel format: presentation only duplicates pixels
and orders bytes for SPI. Other-format images are converted when drawn into
the native surface; untinted RGB565 images have a direct packed copy path.

The `m5cores3se` adapter configures the AXP2101 LCD supply/backlight, AW9523B
LCD/touch reset pins, ILI9342C display (SPI3, GDMA channel 0), and FT6336U
polling input. It owns the internal I2C bus on SDA12/SCL11. SPI uses MOSI37,
SCK36, CS3, and DC35; SD CS4 stays inactive because SD shares that bus.

Build from the repository root:

```sh
mkdir -p sandbox/bin
go build -o sandbox/bin/renvo ./cmd/renvo
sandbox/bin/renvo -backend backends/esp32s3.rtg \
  -t esp32s3/xtensa_lx7 -tags m5cores3se -s \
  -o sandbox/cores3-forms.elf ./examples/device/cores3_forms
```

Preserve the original flash before the first installation. Convert the ELF
with `renvoflash --convert` and install the application at the factory
partition offset recorded in the board's partition table. This demo relies on
the existing ESP32-S3 bootloader. Serial output reports initialization errors,
`CoreS3-SE: forms ready`, and first-frame paint/transfer timings.

Hardware references:

- https://docs.m5stack.com/en/core/M5CoreS3%20SE
- https://github.com/m5stack/M5GFX/blob/master/src/M5GFX.cpp
- https://github.com/m5stack/M5GFX/blob/master/src/lgfx/v1/panel/Panel_ILI9342.hpp

Host touch-decoding checks: `go test -tags m5cores3se ./device/board`.
Compile acceptance check: `go test ./frontend_tests -run '^TestFrontendCoreS3SEForms$'`.

On the tested CoreS3-SE, the factory application partition begins at `0x10000`
and is 7 MB. After making and verifying a backup, installation with esptool is:

```sh
go run ./cmd/renvoflash --convert sandbox/cores3-forms.elf sandbox/cores3-forms.bin
esptool --chip esp32s3 --port /dev/cu.usbmodem1101 --after watchdog-reset \
  write-flash 0x10000 sandbox/cores3-forms.bin
```

Use the serial port for your board. This command replaces only the application.

Hardware validation: the flashed demo reached `CoreS3-SE: forms ready` and
reported live FT6336U contacts and a successful Forms button click. The
application write was verified by esptool's flash hash check.

The adapter promotes an 80 MHz PLL boot clock to 160 MHz before peripheral
initialization. APB remains 80 MHz, SPI writes run at 40 MHz, and the system
timer remains 16 MHz. It preserves a boot clock already at 160/240 MHz.
240 MHz promotion is not implemented: that requires calibrated RTC/digital
voltage-bias setup as well as the CPU divider change.

On the connected CoreS3-SE, the original first frame took 116 ms to paint and
270 ms to convert/transfer. With 160 MHz CPU and native RGB565 rendering,
those times were approximately 83 ms and 59 ms; framebuffer memory fell
from 76,800 to 38,400 bytes.
These are first-frame measurements, not an interactive frame-rate guarantee;
subsequent updates transfer only the dirty rectangle.
