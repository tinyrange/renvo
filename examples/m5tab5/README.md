# M5Stack Tab5

This target brings up the ESP32-P4 in the M5Stack Tab5 without linking ESP-IDF
application code. It uses Renvo's shared RV32IM backend, the ESP32-P4 flash and
pre-v3 internal-SRAM map, watchdog handoff, and native USB Serial/JTAG output.

The connected development unit uses the ST7121 integrated display and touch
controller. Display and multitouch support live in the `board` package and are
kept separate from the processor target so startup can be verified before
enabling the MIPI-DSI and PSRAM subsystems.

## Build and run the startup probe

```sh
go build -o sandbox/renvo ./cmd/renvo
sandbox/renvo \
  -backend examples/m5tab5/esp32p4.rtg \
  -t esp32p4/riscv32 \
  -o sandbox/m5tab5-hello.elf \
  ./examples/m5tab5/hello
sandbox/renvo -t darwin/arm64 -o sandbox/renvoflash ./cmd/renvoflash
sandbox/renvoflash sandbox/m5tab5-hello.elf
```

Successful startup prints `RENVO TAB5 PASS` over the native USB serial port.
The helper writes only the factory application partition at `0x10000`; it does
not write the bootloader, partition table, NVS, data partitions, or eFuses.
Restore the checked and verified whole-flash backup before relying on the
factory software again.

## Run the DSI link probe

Build `./examples/m5tab5/display_probe` with the same target. It first uses the
ESP32-P4 DSI host's color-bar generator to establish the PHY and cold-initialize
the ST7121, then switches to a bordered RGB565 color chart rendered in PSRAM
and supplied through the DSI bridge and DW-GDMA. A successful scanout prints
`TAB5 PSRAM FRAME PASS`.
