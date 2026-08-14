# M5Stack Tab5

This target brings up the ESP32-P4 in the M5Stack Tab5 without linking ESP-IDF
application code. It uses Renvo's shared RV32IM backend, the ESP32-P4 flash and
pre-v3 internal-SRAM map, watchdog handoff, and native USB Serial/JTAG output.

The connected development unit uses the ST7121 integrated display and touch
controller. Tab5 wiring and framebuffer support live in
`device/board/m5tab5`; the reusable controller protocol packages live under
`device/display/st7121` and `device/input/st7121`.

## Build and run the startup probe

```sh
go build -o sandbox/renvo ./cmd/renvo
sandbox/renvo \
  -backend backends/esp32p4.rtg \
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

## Demos

The useful Tab5 demos are also published in the web editor:

- `forms_demo` exercises controls, cached TrueType glyphs, dragging, and the
  on-screen keyboard.
- `sgp30_demo` composes the board, I2C, SGP30, Forms, and graphics packages into
  a compact air-quality dashboard for a Unit connected to Port A.
- `terminal` mirrors `print` and `fmt.Printf` to a color terminal with
  scrollback and a touch keyboard. It streams ADXL345 readings from Port A.
- `terminal_stress` drives variable multi-line bursts through wrapping, ANSI
  rendition, DMA scrolling, stdout mirroring, touch input, and live display
  diagnostics while continuing to sample the ADXL345.
- `touch_trails` visualizes every filtered multitouch contact and is useful for
  validating a display after flashing.

Build any demo by replacing the final package in the startup command, for
example `./examples/m5tab5/forms_demo`.
