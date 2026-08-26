# MS-DOS input and PC hardware demo

This interactive MZ program plays a short PIT/PC-speaker arpeggio, detects and
queries the DOS mouse driver, reads a key through the BIOS, and echoes it with
the BIOS teletype service.

```sh
go run ./cmd/renvo -backend backends/msdos.rtg \
  -t msdos/8086-mz -s -o sandbox/input.exe examples/msdos-input
```
