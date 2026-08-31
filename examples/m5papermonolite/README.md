# M5Stack PaperMono-Lite bring-up oracle

This Phase 1 oracle verifies the existing ESP32-S3 startup, application-only
flash, USB serial, and PaperMono-Lite board selection. Board initialization
places the two active-low user-button pins in input mode, but deliberately does
not power or communicate with the e-paper panel, I/O expander, PMIC, or touch
controller yet.

## Build, back up, and flash

Build the compiler and flasher, then compile the oracle with the shared
ESP32-S3 backend and the PaperMono-Lite board tag:

```sh
go build -o sandbox/renvo ./cmd/renvo
go build -o sandbox/renvoflash ./cmd/renvoflash
sandbox/renvo \
  -backend backends/esp32s3.rtg \
  -t esp32s3/xtensa_lx7 -tags m5papermonolite \
  -s -o sandbox/m5papermonolite-oracle.elf \
  ./examples/m5papermonolite/oracle
```

Before the first write, enter ROM download mode and make a whole-flash backup.
Hold the power button for about two seconds until the red LED flashes, then
release it. `renvoflash` detects this board's 16 MiB SPI NOR from its JEDEC ID
and prints a device-side MD5; compare that digest with `md5` on macOS or
`md5sum` on Linux before treating the backup as restorable.

```sh
sandbox/renvoflash --backup sandbox/papermonolite-factory.bin /dev/cu.usbmodem101
md5 sandbox/papermonolite-factory.bin
```

Replace the example port with the detected USB serial device. Flashing the ELF
writes only the factory application partition at `0x10000`; it preserves the
bootloader, partition table, NVS, other data partitions, and eFuses.

```sh
sandbox/renvoflash sandbox/m5papermonolite-oracle.elf /dev/cu.usbmodem101
```

After reset, the application emits exactly one line:

```text
RENVO PAPERMONO-LITE PASS
```

## Phase 2 power and button oracle

The separate `power_oracle` keeps the one-line startup oracle stable. It first
performs a read-only identity check for the M5PM1 at `0x6e` and M5IOE1 at
`0x4f` over the shared 100 kHz software-I2C bus on GPIO47/GPIO48. It then:

1. establishes a known shutdown state in case expander latches survived a
   battery shutdown;
2. drives EPD chip select inactive before applying display power;
3. holds the EPD and touch controllers in reset while enabling their rails;
4. releases both resets after the documented delay and verifies all four
   output latches;
5. asserts reset, removes both rails, and verifies the shutdown latches.

No SPI command, framebuffer data, or refresh request is sent, and the microSD
power output remains untouched. Build and flash it through the same
application-only path:

```sh
sandbox/renvo \
  -backend backends/esp32s3.rtg \
  -t esp32s3/xtensa_lx7 -tags m5papermonolite \
  -s -o sandbox/m5papermonolite-power-oracle.elf \
  ./examples/m5papermonolite/power_oracle
sandbox/renvoflash sandbox/m5papermonolite-power-oracle.elf /dev/cu.usbmodem101
```

Successful startup prints:

```text
RENVO PAPERMONO-LITE PHASE2 IDENTIFY PASS
RENVO PAPERMONO-LITE PHASE2 POWER PASS
RENVO PAPERMONO-LITE BUTTONS READY
```

The application leaves the display and touch rails off, then reports
`BUTTON A DOWN`/`UP` and `BUTTON B DOWN`/`UP` transitions with 10 ms polling.
Its software-I2C controller bounds every clock-stretch wait and attempts bus
recovery during initialization; every partial power-sequence failure attempts
the complete shutdown path before returning an error.

## Phase 3 display oracle

The `display_oracle` owns the panel through the board package and uses the
reusable `device/display/ssd1677` protocol driver. It configures SPI2 mode 0 at
20 MHz on GPIO14/GPIO15, keeps chip select and data/command on GPIO16/GPIO17,
and bounds every SPI completion wait and GPIO18 BUSY poll. It uses only the
SSD1677's built-in OTP waveforms; no custom LUT is uploaded.

The oracle performs one full monochrome baseline refresh, ten differential
partial refreshes, an automatically promoted recovery full refresh, and one
full four-gray refresh. It then asserts display reset and removes the display
and touch rails. The final retained image is four equal quadrants: white,
light gray, dark gray, and black.

```sh
sandbox/renvo \
  -backend backends/esp32s3.rtg \
  -t esp32s3/xtensa_lx7 -tags m5papermonolite \
  -s -o sandbox/m5papermonolite-display-oracle.elf \
  ./examples/m5papermonolite/display_oracle
sandbox/renvoflash sandbox/m5papermonolite-display-oracle.elf /dev/cu.usbmodem101
```

A complete successful run prints one full-mono pass, ten partial passes, and
then recovery, four-gray, and shutdown passes. The panel can remain unchanged
after shutdown because e-paper retains its optical state without power.

The controller-native packed surface is 800x480 pixels, or 48,000 bytes per
1-bit plane; its axes are rotated relative to the visible 480x800 panel. The
current ESP32-S3 target uses internal RAM and a 128 KiB default managed arena.
The oracle therefore allocates one static monochrome plane and streams each
four-gray plane through a 100-byte packed row instead of keeping the official
demo's three simultaneous planes. The driver also accepts two resident packed
planes on targets with enough memory. Richer graphics should keep using a
packed/streamed surface, or wait for explicit 8 MiB octal-PSRAM support rather
than assuming ordinary allocations land in PSRAM.

Partial refresh is rejected until a successful full monochrome refresh has
established both controller RAM planes. Four-gray output and panel power-off
invalidate that baseline. After ten partial updates, the next request is
automatically changed to a full refresh to limit ghosting and DC imbalance.
All public refresh operations finish in SSD1677 Deep Sleep Mode 1; callers must
use `board.Display.Shutdown` when RAM retention is no longer needed.

The board contains 16 MiB SPI flash and 8 MiB octal PSRAM, but this bring-up
uses neither factory data partitions nor PSRAM. The later touch driver must
normalize the FT6336G's documented active area of X=5..475 and Y=5..795; the
unused edge coordinates should not be treated as valid panel positions.

## Restore the factory application

Keep the verified whole-flash backup outside version control. The factory
partition table assigns `0xF00000` bytes to the application at `0x10000`.
Extract that partition and pass it to `renvoflash`, which continues to write
only the application region:

```sh
dd if=sandbox/papermonolite-factory.bin \
  of=sandbox/papermonolite-factory-app.bin \
  bs=65536 skip=1 count=240
sandbox/renvoflash sandbox/papermonolite-factory-app.bin /dev/cu.usbmodem101
```

Do not pass the whole-flash backup to the ordinary flash command: that command
always starts its input at application offset `0x10000`.
