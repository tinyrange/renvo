# BIOS hello

This example builds a bootable legacy PC disk image, writes through the BIOS
teletype service, checks INT 13h extensions, observes the firmware timer, and
waits for a keyboard key.

```sh
go run ./cmd/renvo -backend backends/msdos.rtg \
  -t bios/8086 -s -o sandbox/renvo-bios.img examples/bios-hello
```

Boot `sandbox/renvo-bios.img` as a hard disk on an IBM-PC-compatible machine or
run `tools/bios/test-qemu` for the automated SeaBIOS smoke test.
