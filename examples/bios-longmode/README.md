# BIOS to amd64 long mode

This example is one source package with two compiler entrypoints. The ordinary
`main` is a 16-bit `bios/8086` program. The `longModeMain` directive creates a
second unit for `freestanding/amd64`, embeds its in-place executable image, and
initializes `longModeEntry` to that image's entrypoint. Both units compile
`sharedMessage` for their own architecture.

```sh
go run ./cmd/renvo -backend backends/bios_multistage.rtg \
  -t bios/8086 -s -o sandbox/bios-longmode.img examples/bios-longmode
```

Run `tools/bios/test-longmode` to build the image and verify the transition with
QEMU's debug console.
