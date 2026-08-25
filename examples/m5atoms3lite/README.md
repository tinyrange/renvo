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
  -t esp32s3/xtensa_lx7 -tags m5atoms3lite \
  -o sandbox/m5atoms3lite-button-rgb.elf \
  ./examples/device/button_rgb
```

Flash the factory application region while preserving the bootloader,
partition table, NVS, and eFuses:

```sh
./sandbox/renvoflash sandbox/m5atoms3lite-button-rgb.elf
```

## SK6812 strip

The `sk6812_strip` example drives the 10 cm, 15-pixel M5Stack A035 RGB LED
Strip from the AtomS3 Lite Grove port on GPIO2. Each debounced button press
chooses a low-brightness color from the hardware RNG and shoots it from the
connector end to the far end of the strip. Input remains live during the
animation, so rapid presses launch independently moving shots while holding
the button still triggers only once.

```sh
sandbox/renvo \
  -backend backends/esp32s3.rtg \
  -t esp32s3/xtensa_lx7 -tags m5atoms3lite \
  -o sandbox/m5atoms3lite-sk6812-strip.elf \
  ./examples/device/ws2812_shots
./sandbox/renvoflash sandbox/m5atoms3lite-sk6812-strip.elf
```

## ADXL345 accelerometer

The `adxl345` example reads signed raw X, Y, and Z acceleration once per
second from an ADXL345 connected to the Grove port. The status LED is green
while samples are arriving, red after a read failure, and blue when the device
does not initialize. It uses the sensor's `0x53` address, selected when the
SDO/ALT ADDRESS pin is grounded.

```sh
sandbox/renvo \
  -backend backends/esp32s3.rtg \
  -t esp32s3/xtensa_lx7 -tags m5atoms3lite \
  -o sandbox/m5atoms3lite-adxl345.elf \
  ./examples/device/adxl345
./sandbox/renvoflash sandbox/m5atoms3lite-adxl345.elf
```

## ENV Pro environmental sensor

The `env_pro` example reads calibrated temperature, pressure, humidity, and
gas resistance once per second from the BME688 in an M5Stack ENV Pro Unit on
the Grove port. The status LED is green when gas data and the heater are both
valid, amber while the heater settles, red after a read failure, and blue when
the device does not initialize. The ENV Pro selects I2C address `0x77`.

The BME688 itself does not directly report IAQ or equivalent CO2. Those values
require Bosch's separately distributed BSEC algorithm; this open driver reports
the calibrated physical sensor outputs without inventing an IAQ conversion.
Its serial stream uses Arduino Serial Plotter-compatible labelled values, which
Renvo's browser IDE graphs automatically after Flash & Run.

```sh
sandbox/renvo \
  -backend backends/esp32s3.rtg \
  -t esp32s3/xtensa_lx7 -tags m5atoms3lite \
  -o sandbox/m5atoms3lite-env-pro.elf \
  ./examples/device/env_pro
./sandbox/renvoflash sandbox/m5atoms3lite-env-pro.elf
```
