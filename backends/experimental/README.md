# Experimental backends

## Linux user-mode QEMU

The `linux_*.rbe` entrypoints provide experimental native Linux ELF targets.
They compile parsed source directly to machine instructions and static ELF
images, without a host assembler, linker, guest compiler, or guest sysroot.
Each Linux `.rbe` is independently maintained and self-contained: the file
contains its selected CPU encoder, calling convention, Linux runtime, and ELF
writer, with no `@import` dependencies. Copy one file to add or adapt that
backend; changes to it do not affect another target's emitter. The four
maintained Linux targets remain in `backend/definitions/`.

The smoke is opt-in: its module lives under `frontend_tests/manual/`, outside
the automatically discovered corpus tiers. CI and `tools/check` do not invoke
the QEMU runner. The focused compiler regression programs remain in the normal
backend corpus.

Run the checked-in frontend smoke on all 33 QEMU user emulators:

```sh
./tools/qemu-smoke/run
./tools/qemu-smoke/run --target mips --target mipsel
./tools/qemu-smoke/run --list
```

The runner builds `cmd/renvo` once, copies each experimental RBE to an isolated
temporary location with a different filename, and compiles the same source
module for each selected target. This verifies single-file portability as well
as comparing stdout byte-for-byte with its `expected.txt`.
Nonzero exits, timeouts, stderr, missing emulators, and output mismatches fail
the run. It uses a temporary directory under ignored `sandbox/` for binaries
and guest working files, disables core dumps, and removes its temporary files.
It never invokes host Go as an output oracle. `--compiler` reuses an existing
Renvo binary; `--timeout` controls each compile/run timeout (default 60 seconds).

The complete mapping lives in
[`tools/qemu-smoke/targets.json`](../../tools/qemu-smoke/targets.json):

| QEMU emulator suffix | Implementation |
| --- | --- |
| aarch64, arm, i386, x86_64 | Maintained Linux backends |
| aarch64_be, armeb | Big-endian data variants of the maintained emitters |
| mips, mipsel, mipsn32, mipsn32el, mips64, mips64el | MIPS32 and MIPS64, O32/N32/N64 Linux ABIs |
| m68k | Imported Motorola 68000 emitter |
| sh4, sh4eb | Imported SuperH emitter |
| riscv32, riscv64 | Maintained RV32 architecture and experimental RV64 port |
| alpha, hexagon, hppa, loongarch64 | Independent RBE with its own ISA encoder |
| microblaze, microblazeel, or1k | Independent RBE with its own ISA encoder |
| ppc, ppc64, ppc64le, s390x | Independent RBE with its own ISA encoder |
| sparc, sparc32plus, sparc64 | Independent RBE with its own ISA encoder |
| xtensa, xtensaeb | Linux ports of the maintained Xtensa CALL0 emitter |

`qemu-xtensaeb` explicitly selects `-cpu test_kc705_be`, which supplies the
integer divide instructions used by the emitter. The other rows use QEMU's
default CPU. The PowerPC64 images use ELFv2 and Renvo's internal calling
convention. HPPA reserves a private 4 MiB stack in BSS for that convention.

The smoke source is
[`qemu_native_smoke`](../../frontend_tests/manual/qemu_native_smoke/cmd/app/main.go).
It checks signed quotient/remainder identities, high-bit unsigned division,
oversized shifts, indirect function calls,
recursion, eight arguments including stack arguments, a 512-element sieve,
xorshift-generated insertion sorting, CRC32, initialized globals, linked heap
records, aggregate returns, growing and aliased slices, overlapping copies,
signed and unsigned narrow loads, and string slicing/equality. Success prints
only `PASS\n`.

These are integer and memory smoke tests, not self-hosting or full frontend
conformance. The experimental Linux entrypoints currently accept the zero
entry-argument form; they do not advertise argv/environment support. Filesystem
operation bindings exist but are not covered by this smoke. Floating point,
64-bit arithmetic on narrow targets, and the full standard library are not
validated by this matrix. Experimental code size and speed are not substitutes
for the maintained compiler performance gates.

## Imported reusable architectures

The seven CPU-only files `mips32.rtg`, `m68k.rtg`, `superh.rtg`, `z80.rtg`,
`lr35902.rtg`, `mos6502.rtg`, and `wdc65816.rtg` were imported from
`tinyrange/renvo_console` at commit
`6d9c7de8bd30aa207a5e0eba3ea5838488f1a0fd` under Apache-2.0. Console image,
firmware, peripheral, and SDK entrypoints were not imported. Z80, LR35902,
MOS6502, and WDC65816 have no corresponding QEMU user emulator; their presence
here does not imply validation by the Linux matrix.

The MIPS, SuperH, and m68k host trap bindings were adapted for Linux. The m68k
scratch register no longer overlaps the string comparison argument registers.
The standalone Linux RBEs contain their own copies of the relevant CPU code;
they do not import these architecture fragments. MIPS64 and RV64 are native-width
ports of the reusable MIPS32 and RV32 sources. There is no shared Linux encoder
or image-helper file, and no CPU selector dispatching across unrelated ISAs.
The ARM, AArch64, and Xtensa variants originate in this repository's maintained
architecture definitions. Big-endian Xtensa keeps canonical instruction fields
until relocation, then converts those fields to the big-endian instruction
layout; data retains the target byte order throughout.

The additional ISA encoders were authored here and checked with LLVM MC where
supported and QEMU execution. Processor encodings and syscall ABI facts were
also checked against the QEMU `target/` and `linux-user/` sources and Linux
architecture syscall tables. No QEMU or Linux implementation source is included
in these backends. See the upstream
[console provenance](https://github.com/tinyrange/renvo_console/blob/6d9c7de8bd30aa207a5e0eba3ea5838488f1a0fd/PROVENANCE.md),
[QEMU architecture sources](https://github.com/qemu/qemu/tree/master/target), and
[Linux architecture sources](https://github.com/torvalds/linux/tree/master/arch).

## Microcontroller backends

This directory contains source RBE backends for the microcontrollers exposed
by `tinyrange/renvo_emu` that do not already have maintained Renvo backends.
They are deliberately kept outside the generated production backend set while
their architecture and device APIs mature.

| renvo_emu target | RBE target | CPU selected by the RBE | Image |
| --- | --- | --- | --- |
| `ch32v003` | `experimental/ch32v003` | QingKe V2 RV32E | ELF32 |
| `ch32v006` | `experimental/ch32v006` | QingKe V2 RV32E | ELF32 |
| `atsamd21e18` | `experimental/atsamd21e18` | Cortex-M0+ | ELF32 |
| `stm32l432kc` | `experimental/stm32l432kc` | Cortex-M4 | ELF32 |
| `r7fa4m1ab3cfm` | `experimental/r7fa4m1ab3cfm` | Cortex-M4 | ELF32 |
| `atmega328pb` | `experimental/atmega328pb` | enhanced AVR8 | ELF32 |
| `msp430fr2433` | `experimental/msp430fr2433` | MSP430X | ELF32 |
| `pic16f15376` | `experimental/pic16f15376` | enhanced mid-range PIC16 | Intel HEX |
| `efm8bb52f32g` | `experimental/efm8bb52f32g` | MCS-51 | Intel HEX |

ESP32-C6, ESP32-S3, RP2040, and RP2350 are intentionally absent here. Use the
maintained `backends/esp32c6.rtg`, `backends/esp32s3.rtg`, `backends/rp2.rtg`,
and `backends/rp2350.rtg` definitions for those devices. The 32-bit
experimental targets use renvo_emu's compiler UART and exit facades. The four
8/16-bit targets initialize and use their native UART peripherals.

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

### Microcontroller source conventions

- Do not add an experimental target when a maintained backend already covers
  the device.
- Keep architecture, emulator runtime, and image helpers in separate `.rtg`
  imports, and retain a shared helper only while at least two RBEs use it.
- Format embedded Go and `@stdlib` sources with `gofmt`; use the spacing and
  layout conventions from `backend/definitions/` for RTG declarations.
