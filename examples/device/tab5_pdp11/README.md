# Unix V7 on M5Stack Tab5

Renvo compiles the shared [PDP-11 C core](../../pdp11/README.md) directly for
ESP32-P4/RISC-V. The Go adapter supplies SDMMC/FAT32, internal SRAM, a PSRAM disk, the Tab5 Keyboard
on Ext.Port1, and the existing native-rotation Forms terminal. No ESP-IDF or
Arduino runtime is linked into the application.

## Prepare and build

Put a decompressed TUHS `v7_rk05_1145.gz` image at `/RV260907/V7RK0.DSK` on a
FAT32 SDHC/SDXC card. Create the directory first if necessary. The image is
external and is not redistributed in the source or firmware; see the host
example's archive links and licensing notes. RL01/RL02 images are not RK05s.

From the repository root:

```sh
go build -o sandbox/renvo ./cmd/renvo
sandbox/renvo cc -backend backends/esp32p4.rtg -t esp32p4/riscv32 \
  -tags m5tab5 -o sandbox/tab5-pdp11.elf examples/device/tab5_pdp11/main.c
```

Use the existing Tab5 ELF-to-ESP image/flash workflow, writing only the app at
`0x10000`. Keep the installed bootloader, partition table, and NVS. Compile
`main.c` only: it includes the core and imports `board.go` through `#pragma go`.
The Go `main` runs package/board initializers before calling C's `pdp_run`;
this is important because the board establishes its PSRAM arena during init.

## Use

The example starts in landscape and automatically sends `boot` and
`rk(0,0)rkunix` at the disk bootstrap's prompts. It executes the real disk
bootstrap and kernel. After `mem = 178816`, the `#` prompt is Unix's root shell.
This small image is not a full V7 development distribution.

- Ctrl+O switches portrait/landscape without restarting the guest. The terminal
  redraws into its native-oriented buffer; no later pixel rotation is used.
- Enter sends one CR, even though the keyboard reports CRLF. Guest output is
  preserved exactly, and Unix supplies echo and line discipline.
- Backspace sends DEL; other controls and ANSI arrow sequences pass through.
  The old Unix TTY may use different erase/interrupt characters than modern
  shells. Ctrl+O is reserved locally and is not delivered to the guest.
- Touch-drag uses terminal scrollback. Serial output mirrors the guest display.
- Type `STTY -LCASE` if you want to disable the image's uppercase presentation.

## Memory and storage policy

Only RK0 is attached. The SD file is opened **read-only**, streamed in 32 KiB
batches through the board's 4-bit, 20 MHz SDMMC DMA driver, and zero-extended to
the RK05's 2,494,464-byte capacity in PSRAM.

The PDP-11's **248 KiB guest RAM stays entirely in internal SRAM**, at
`0x4ff41000..0x4ff7efff`. The board reservation checks the L2 cache partition,
never falls back to PSRAM, and keeps clear of BSS, startup/stack storage, and
display DMA descriptors. Reservations last until reset. Unsupported cache
layouts fail explicitly instead of overwriting cache storage.

All subsequent disk reads/writes—including swap and file creation—use that RAM
copy. **Reset/power loss discards guest changes.** There is no SD writeback or
save command. This keeps random guest sector traffic off the FAT32 path and
protects the source image. Multiple drives and persistent overlays are future
work, not silently enabled features.

Before peripheral initialization, the adapter promotes the bootloader's
CPLL/4 clock (90 MHz on the tested board) to CPLL/1 (360 MHz). It changes APB,
memory, then CPU dividers in the documented upscale order, preserving safe
180 MHz memory/90 MHz APB clocks. It does not overclock or recalibrate the PLL.
Unknown clock layouts fail explicitly. See Espressif's
[clock transition implementation](https://github.com/espressif/esp-idf/blob/v5.4.2/components/esp_hw_support/port/esp32p4/rtc_clk.c).

The CPU runs 1,024 instruction slots between host polls. Keyboard/display work
is limited to roughly 16 ms intervals, with a bounded input queue. The keyboard
IRQ avoids empty software-I2C transactions, with a 250 ms fallback poll. Clock
interrupts use a fractional 60 Hz board-timer accumulator; long pauses coalesce
ticks rather than generating a catch-up interrupt storm. This is not a
cycle-accurate emulator.

Instruction fetch uses a one-entry translation cache for complete readable RAM
pages. It is invalidated on MMU-register writes, includes CPU mode in its tag,
and always fetches current RAM bytes (self-modifying code is preserved). Partial
pages, faults, I/O mappings and odd addresses retain the checked path.

## Hardware smoke test

Add `-DPDP_AUTOTEST` to the build to send a one-shot V7 shell test after boot:
echo `RENVO_P4_V7_OK`, list `/`, create/read/delete `/renvo` in the RAM-backed
guest filesystem. Verify actual output lines (not just echoed commands),
including `RENVO_RAM_WRITE_OK`. Normal builds send no shell commands.

For on-board instruction/MMU and SRAM-allocation tests without touching SD,
build `examples/device/tab5_pdp11/tests/main.c` with the same target flags. The
serial transcript must contain `PASS` followed by `PASS SRAM bounds and alignment`.
This also verifies the internal pool rejects invalid alignment and exhaustion.

The shared host CPU tests and opt-in V7 boot test are documented beside the
core. They do not substitute for running the prepared RISC-V build on hardware.

Measured on the connected rev1.3 board: the initial 90 MHz/PSRAM-guest build ran
about 12–19k instruction slots/s during boot; full clock, internal guest SRAM
and cached instruction translation reached about 60–63k/s. Loading the 2,048,512
byte SD image took about 0.44 s. V7 boot still takes roughly a minute; UI/input
servicing accounted for about 1.2 s over the first minute. The remaining cost is
primarily interpretation, not disk I/O. These are workload measurements, not
cycle-accurate PDP-11 speed claims.
