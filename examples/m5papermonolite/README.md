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
