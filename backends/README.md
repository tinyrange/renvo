# RTG backends

This directory contains complete external backend entrypoints accepted by
`renvo -backend`. Architecture and operating-system fragments shared by those
entrypoints remain in [`backend/definitions`](../backend/definitions), while
frontend target profiles remain in [`systems`](../systems).

The available external backends are:

- `android_arm64.rtg`
- `c89.rtg`
- `esp32c6.rtg`
- `esp32c6_jtag.rtg`
- `esp32p4.rtg`
- `esp32s3.rtg`
- `ios_arm64.rtg`
- `msdos.rtg` (`msdos/8086` COM, `msdos/8086-mz` MZ EXE, and the
  `bios/8086` raw boot disk imported from `bios_8086.rtg`)
- `freestanding_amd64.rtg` (the embeddable `freestanding/amd64`
  memory-image target)
- `bios_multistage.rtg` (the optional catalog combining `bios/8086` with
  `freestanding/amd64` for multi-architecture boot programs)
