# MS-DOS system demo

This MZ program reports DOS date, time, version, drive, and directory data. It
also exercises the BIOS clock, conventional-memory segment access, serial
status, and printer status APIs.

```sh
go run ./cmd/renvo -backend backends/msdos.rtg \
  -t msdos/8086-mz -s -o sandbox/system.exe examples/msdos-system
```
