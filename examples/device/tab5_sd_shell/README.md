# Tab5 SD shell

A small filesystem terminal for the M5Stack Tab5 and Tab5 Keyboard. It starts
in landscape; **Ctrl+O** switches portrait/landscape using the native rotated
Forms surface. **Ctrl+L** clears, **Ctrl+C** cancels input, and Up/Down recall
the last 16 commands. Enter executes once, including keyboards sending CRLF.

Insert an already formatted FAT32 SDHC/SDXC card before starting. Mounting and
browsing do not write to the card. There is no format or raw-write shell command.
Back up important cards before testing filesystem writes.

```text
help
ls
ls -a
cd music
ls
cd ..
cat config.txt
head playlist.txt
hex config.txt
stat config.txt
mkdir notes
write notes/hello.txt "Hello from Tab5"
cat notes/hello.txt
touch notes/empty.txt
rm notes/empty.txt
sync
```

`write` creates/replaces the entire file with its text arguments and a newline;
`touch` creates an empty file only if missing. `rm` removes files, and `rmdir`
removes empty directories. Relative and absolute paths, `.` and `..`, quotes,
and backslash escapes work. This is bash-ish navigation, not Bash: no external
programs, pipes, redirects, globbing, variables, scripts, or completion. Input
is ASCII, append/backspace editing, limited to 256 bytes. File control bytes
are escaped on display; CR, LF and CRLF are normalized to terminal CRLF.
`head` displays up to 20 lines; file display consumes at most 64 KiB of input.

Plain `ls` omits dot names (including macOS `._` sidecars) and entries with the
FAT hidden attribute. `ls -a` or `ls -A` includes these entries, without synthetic
`.`/`..` rows; `-l` is accepted because listings already include sizes. Hidden
paths remain accessible when named explicitly. Use `--` before a pathname
beginning with `-`. Names containing spaces, quotes or backslashes are quoted
so their displayed spelling can be used in commands.

## Storage layers

- `device/block.Device`: synchronous, bounds-checked whole 512-byte block reads,
  writes and `Sync`. `board.OpenSD()` returns a Tab5 implementation.
- `device/fat32`: `Mount`, `Stat`, `ReadDir`, streaming `Open`/`File.Read`,
  `WriteFile`, `Mkdir`, `Remove`, and `Sync`. `File.Size` returns bytes; reads
  return EOF after the file. Readers need no Close because they own no handles.
- `device/shell`: reusable command parsing/execution with an `io.Writer` output,
  separate from the hardware keyboard and terminal event loop in `main.go`.

Each card/volume has one synchronous owner. Do not mutate a volume while reading
an open file, share it between concurrent callers, or remove the card while
running. There is no hot-plug recovery. A failed block transfer disables that
card instance; a write/sync failure disables further filesystem writes on that
mount. Resolve the media error and restart/remount before trying again.

Supported layouts are FAT32 superfloppy or the first primary MBR FAT32 partition
(types 0x0b/0x0c), with 512-byte sectors. FAT12/16, exFAT, GPT, extended partitions,
and legacy SDSC cards are unsupported. Existing VFAT long names can be read,
overwritten, or removed, with their short aliases also accepted. **New names
must be portable ASCII 8.3**; Unicode case folding and long-name creation are
not implemented. Newly created entries use the placeholder date 1980-01-01.
There is no rename, seek, append stream, recursive delete, or formatter.

The filesystem updates enabled FAT copies and invalidates FSInfo allocation
hints. Replacing a file builds and syncs a new chain before publishing its
directory entry and freeing the old chain. This is not a journal or a
power-failure guarantee: interrupted writes can leak clusters or tear metadata.
`Sync` waits for the card to finish writes; keep power connected until it returns.
Corrupt-chain checks are not a full filesystem consistency checker; damaged or
cross-linked volumes should be repaired on a computer before writing.

## Why native SDMMC

The [Tab5 pin map](https://docs.m5stack.com/en/core/Tab5) connects the card to
ESP32-P4 slot 0's native IOMUX: DAT0–3 on GPIO39–42, CLK43, CMD44. The driver uses
this dedicated **4-bit SDMMC** bus, not SPI: identification at 400 kHz followed
by default-speed 20 MHz SDR, without UHS voltage switching.

Reads/writes use the peripheral's internal descriptor DMA, batching up to
32 KiB per command, with automatic stop for multi-block transfers. Descriptors
and PSRAM bounce buffers are 64-byte aligned; explicit cache writeback and
invalidation make arbitrary caller buffers safe. Completion checks include both
card-data and DMA completion, followed by card-ready status. Commands and data
waits are bounded; ambiguous writes are not automatically retried.

FAT and directory metadata have separate one-sector caches. File reads and
writes batch whole sectors within each cluster; partial sectors use scratch
storage. Files are streamed for display, not loaded in their entirety. This is
an optimized native-bus implementation, not a measured maximum-throughput claim.

Register/clock/DMA details were checked against Espressif's
[SDMMC host documentation](https://docs.espressif.com/projects/esp-idf/en/stable/esp32p4/api-reference/peripherals/sdmmc_host.html),
[P4 low-level definitions](https://github.com/espressif/esp-idf/blob/v5.5/components/hal/esp32p4/include/hal/sdmmc_ll.h),
and [transaction driver](https://github.com/espressif/esp-idf/blob/v5.5/components/esp_driver_sdmmc/src/sdmmc_transaction.c).

## Build and test

Select **Tab5 SD shell** in the board example catalog, or build with:

```sh
go build -o sandbox/renvo ./cmd/renvo
sandbox/renvo -backend backends/esp32p4.rtg -t esp32p4/riscv32 \
  -tags m5tab5 -o sandbox/tab5-sd-shell.elf ./examples/device/tab5_sd_shell
```

Flash only the application at `0x10000`, retaining the existing bootloader and
partition table. During development the repository ROM flasher failed on a
larger image; Espressif's esptool stub/compressed path flashed and verified the
converted application image successfully. Do not use an unverified partial flash.

Host tests cover FAT32 read/write/remount, replacement failure, FAT mirroring,
FSInfo hints, MBR preservation, directory growth, long-name validation/deletion,
cycle rejection, shell quoting, and CR/LF handling:

```sh
go test ./frontend_tests -run TestFAT32 -count=1
./tools/check frontend fat32_sd_shell
./tools/check backend 'returned_slice_dma_alias|prepared_fixed_interface_equal'
./tools/check preflight
```

Hardware validation on the connected Tab5 detected 31,116,288 sectors and read
the existing root entries. With permission, the test created `/RV260907`, wrote
and remount-verified 65,567 patterned bytes, replaced and verified the file, and
removed that temporary data file. `/RV260907/RESULT.TXT` records the successful
test. The MBR remained byte-identical; existing user files were not modified.
