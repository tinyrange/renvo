# MS-DOS VGA demo

Build a relocatable MZ executable from the repository root:

```sh
go run ./cmd/renvo \
  -backend examples/msdos/msdos_com.rtg \
  -t msdos/8086-mz -s -o sandbox/vga.exe \
  examples/msdos-vga
```

The program selects VGA mode 13h, writes a palette and a generated framebuffer,
waits for a key through BIOS INT 16h, and restores the 80x25 text mode. The
device calls are implemented by `device/dos`; its target-specific primitives
are evaluated from `.rtgasm` against the selected MS-DOS backend.
