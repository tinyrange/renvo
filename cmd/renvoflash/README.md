# renvoflash

`renvoflash` is compiled by Renvo itself. It converts a Renvo ESP32-C6 or
ESP32-S3 ELF32 executable to the
Espressif application-image format, writes the factory application partition at
flash offset `0x10000`, resets the board, and briefly displays its serial output.
It can also accept an already converted ESP application `.bin` file. The
command invokes no external flashing tool and contains no host-Go implementation.

It can also make a read-only, whole-flash backup of an ESP32-C6, ESP32-S3, or
ESP32-P4. Before reading, backup mode refuses secure-download mode and flash
encryption, configures the detected flash geometry, and calculates a device-side
MD5. The printed digest must match the host file's MD5 before the backup is used.

```sh
sandbox/renvo -o sandbox/renvoflash ./cmd/renvoflash
./sandbox/renvoflash firmware.elf
./sandbox/renvoflash --backup factory.bin
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
