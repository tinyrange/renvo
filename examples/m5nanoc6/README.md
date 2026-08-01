# M5NanoC6 blue LED

This example is the first freestanding microcontroller target built entirely
through an external RTG definition. The target combines the shared RV32IM
machine definition with the ESP32-C6 memory map, watchdog handoff, and ELF app
image contract. It drives the M5NanoC6 blue LED on GPIO7 without ESP-IDF.

The target is intentionally loaded with `-backend`: it exercises the same JIT
preparation path available to custom boards and does not advertise itself as a
compiled-in host target.

## Emulation oracle

Clone `tinyrange/renvo_emu` beside this repository, or set
`RENVO_EMU_ROOT`, then run:

```sh
./examples/m5nanoc6/test-emulator.sh
```

The test compiles the oracle with Renvo and asks the ESP32-C6 emulator to stop
on the first rising edge of `board.esp32c6.chip_gpio.pin7`. It deliberately
does not request a bus log because long bus logs are expensive and the signal
edge is the assertion under test.

## Build and flash

Build Renvo and the blinking application:

```sh
go build -o sandbox/renvo ./cmd/renvo
sandbox/renvo \
  -backend examples/m5nanoc6/esp32c6.rtg \
  -t esp32c6/riscv32 \
  -o sandbox/m5nanoc6-blink.elf \
  ./examples/m5nanoc6/blink
```

Convert the ELF with esptool and write only the factory application partition:

```sh
esptool.py --chip esp32c6 elf2image \
  --flash-mode dio --flash-freq 80m --flash-size 4MB \
  -o sandbox/m5nanoc6-blink.bin sandbox/m5nanoc6-blink.elf
esptool.py --chip esp32c6 --port /dev/ttyACM0 write-flash \
  0x10000 sandbox/m5nanoc6-blink.bin
```

Do not pass `elf2image --use-segments`: esptool must see the dedicated
`.flash.appdesc` section so it can place the ESP application descriptor at
image offset `0x20`. The commands above preserve the bootloader, partition
table, NVS, and eFuses, but they do replace the application at `0x10000`; back
up the board before flashing if its factory application matters.

After flashing, disconnect and reconnect USB power. A serial reset can leave
the ESP32-C6 USB-JTAG/serial controller in download mode even though the app
partition was written successfully.
