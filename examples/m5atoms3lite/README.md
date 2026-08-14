# M5Stack AtomS3 Lite example

The `button_rgb` example uses the AtomS3 Lite button on GPIO41 to choose a
new color for its GPIO35 addressable LED. It runs without ESP-IDF on Renvo's
ESP32-S3 Xtensa target.

Build the compiler, the Renvo-native flasher, and the example on macOS with:

```sh
go build -o sandbox/renvo ./cmd/renvo
sandbox/renvo -t darwin/arm64 -o sandbox/renvoflash ./cmd/renvoflash
sandbox/renvo \
  -backend backends/esp32s3.rtg \
  -t esp32s3/xtensa_lx7 \
  -o sandbox/m5atoms3lite-button-rgb.elf \
  ./examples/m5atoms3lite/button_rgb
```

Flash the factory application region while preserving the bootloader,
partition table, NVS, and eFuses:

```sh
./sandbox/renvoflash sandbox/m5atoms3lite-button-rgb.elf
```
