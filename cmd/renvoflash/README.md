# renvoflash

`renvoflash` is compiled by Renvo itself. It converts a Renvo ESP32-C6 or
ESP32-S3 ELF32 executable to the
Espressif application-image format, writes the factory application partition at
flash offset `0x10000`, resets the board, and briefly displays its serial output.
It can also accept an already converted ESP application `.bin` file. The
command invokes no external flashing tool and contains no host-Go implementation.

```sh
sandbox/renvo -o sandbox/renvoflash ./cmd/renvoflash
./sandbox/renvoflash firmware.elf /dev/ttyACM0
```

For the first ESP32-C6 raw-PHY tests, start recovery flashing before resetting
or powering the board:

```sh
./sandbox/renvoflash --recover known-good.elf /dev/ttyACM0
```

Recovery mode waits indefinitely while the experimental USB personality owns
the pins. A guarded firmware attempt watchdog-resets after twenty seconds, then
leaves the reset-default USB Serial/JTAG interface active for five seconds.
`renvoflash` catches that repeating window, enters the ROM loader, and replaces
the application automatically. Press Ctrl-C to stop waiting.

The default port is `/dev/ttyACM0`. On Linux the user must have read/write
permission for the serial device (commonly through the `dialout` or `uucp`
group). Use `--convert ELF BIN` to produce an application image without
flashing it.

Only the application region is replaced. The bootloader, partition table, NVS,
and eFuses are not written.
