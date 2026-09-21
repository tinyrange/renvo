# M5Stack Tab5

This target brings up the ESP32-P4 in the M5Stack Tab5 without linking ESP-IDF
application code. It uses Renvo's shared RV32IM backend, the ESP32-P4 flash and
pre-v3 internal-SRAM map, watchdog handoff, and native USB Serial/JTAG output.

The connected development unit uses the ST7121 integrated display and touch
controller. Tab5 wiring and framebuffer support live in
`device/board` behind the `m5tab5` target tag; the reusable controller protocol packages live under
`device/display/st7121` and `device/input/st7121`.

## Build and run the startup probe

```sh
go build -o sandbox/renvo ./cmd/renvo
sandbox/renvo \
  -backend backends/esp32p4.rtg \
  -t esp32p4/riscv32 -tags m5tab5 \
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
example `./examples/device/forms_demo`.

## Simulate Forms display traffic

The host-side Tab5 simulator exercises the real 720 by 1280 RGB565 Forms demo,
its retained two-generation buffer algorithm, and pointer dispatch without
initializing physical hardware. The acceptance test drags the slider for 60
consecutive UI frames and reports paint latency plus four separate quantities:

- changed bytes are RGB565 pixels whose value differs from the displayed frame;
- damage bytes are the union of regions Forms repainted, including overdraw;
- cache writeback bytes count the backend's coalesced whole-row maintenance
  ranges, rounded to 64-byte cache lines (actual bus traffic depends on dirty
  cache lines); and
- scanout bytes are the continuous video-mode framebuffer payload.

Run it with:

```sh
go test -tags m5tab5 -run TestTab5SliderDragAt60FPS -v \
  ./examples/device/forms_demo
```

The Tab5 timing programmed by the backend is 720 by 1280 active pixels inside
an 802 by 1544 raster at an 80 MHz pixel clock. That is 64.61 Hz. Each RGB565
frame is 1,843,200 bytes, so the display continuously reads and transmits about
119.08 MB/s of active pixel payload even when the UI is unchanged. During the
active part of a line the payload rate is 160 MB/s. The two 1040 Mbps DSI lanes
have 260 MB/s of aggregate raw capacity before protocol overhead.

The test requires the host-rendered p95 paint time to stay below the 16.67 ms
60 FPS budget and verifies that all 60 automated updates are presented. This is
a software regression gate, not a physical ESP32-P4 timing claim: it precisely
models pixel volume and buffer generations, but not P4 cache misses, PSRAM
latency, DMA arbitration, or DSI underruns. `FramebufferStats` and the demo's
on-screen FPS counter remain the authoritative on-device checks for those
effects.
