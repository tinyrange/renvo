# PC BIOS devices

`renvo.dev/device/bios` is the firmware API for raw `bios/8086` boot images.
It exposes video, keyboard, disk, serial, timer, port-I/O, and segmented-memory
operations without assuming DOS. The generated image contains its own boot
sector and loads the Renvo program into a single real-mode segment.

Build the hello example with:

```sh
go run ./cmd/renvo -backend backends/msdos.rtg \
  -t bios/8086 -s -o sandbox/renvo-bios.img examples/bios-hello
```

The boot sector tries INT 13h extensions three times, then queries the drive
geometry and uses classic CHS reads when extensions are unavailable. Programs
can use either `ReadSectors` or `ReadCHS` directly. Run `tools/bios/test-qemu`
for the opt-in end-to-end smoke test of both boot paths.

`EnterLongMode` is the bridge for multi-target boot loaders. Pass it a
`uintptr` variable annotated with `//renvo:compile -t pc-longmode/amd64 entry`;
the compiler embeds that separately compiled program in the BIOS image and the
bridge builds identity-mapped page tables before transferring to its 64-bit
entrypoint. See `examples/bios-longmode` for an end-to-end image.
