# Experimental microcontroller backends

This directory contains source RBE backends for every microcontroller exposed
by `tinyrange/renvo_emu` at revision
`37a63e5343d4c77ae9eb5bc568ccfaf45b844a5c` (2026-08-29). They are deliberately
kept outside the generated production backend set while their architecture and
device APIs mature.

| renvo_emu target | RBE target | CPU selected by the RBE | Image |
| --- | --- | --- | --- |
| `ch32v003` | `experimental/ch32v003` | QingKe V2 RV32E | ELF32 |
| `ch32v006` | `experimental/ch32v006` | QingKe V2 RV32E | ELF32 |
| `rp2040` | `experimental/rp2040` | Cortex-M0+ | ELF32 |
| `rp2350` | `experimental/rp2350` | Hazard3 RV32 | ELF32 |
| `esp32s3` | `experimental/esp32s3` | Xtensa LX7 | ELF32 |
| `esp32c6` | `experimental/esp32c6` | RV32IMAC | ELF32 |
| `atsamd21e18` | `experimental/atsamd21e18` | Cortex-M0+ | ELF32 |
| `stm32l432kc` | `experimental/stm32l432kc` | Cortex-M4 | ELF32 |
| `r7fa4m1ab3cfm` | `experimental/r7fa4m1ab3cfm` | Cortex-M4 | ELF32 |
| `atmega328pb` | `experimental/atmega328pb` | enhanced AVR8 | ELF32 |
| `msp430fr2433` | `experimental/msp430fr2433` | MSP430X | ELF32 |
| `pic16f15376` | `experimental/pic16f15376` | enhanced mid-range PIC16 | Intel HEX |
| `efm8bb52f32g` | `experimental/efm8bb52f32g` | MCS-51 | Intel HEX |

RP2350 has both Arm and RISC-V cores; this definition intentionally selects
the Hazard3 core so RP2040 and RP2350 also exercise different Renvo emitters.
The 32-bit direct-load targets use renvo_emu's compiler UART and exit facades.
The four 8/16-bit targets initialize and use their native UART peripherals.

Each RBE embeds a small `renvo.dev/device/remu` package with the selected target
name, memory ranges, and GPIO count. Build a source program directly from an RBE:

```sh
go run ./cmd/renvo \
  -backend backends/experimental/ch32v003.rbe \
  -t experimental/ch32v003 \
  -o program.elf program.go

remu run --target ch32v003 --elf program.elf --max-instructions 100000
```

For PIC16 and EFM8, use a `.hex` output name and pass `--hex` to `remu`.

The narrow cores currently expose Renvo's native values as 16-bit values. The
`load.u32`, `store.u32`, and 32-bit arithmetic bindings therefore preserve the
low 16 bits only. Their scalar call, arithmetic, static-data, and console paths
are executable, but aggregate locals, dynamic indexing, and some signed
control-flow combinations do not yet pass the full frontend corpus. These are
explicit bootstrap limitations, not claims that the physical CPUs provide
32-bit semantics.
