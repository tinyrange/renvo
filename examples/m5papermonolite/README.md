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
