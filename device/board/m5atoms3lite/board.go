// Package board describes the hardware attached to an M5Stack AtomS3 Lite.
package board

import (
	"renvo.dev/device/clock"
	"renvo.dev/device/esp32s3"
	"renvo.dev/device/gpio"
	"renvo.dev/device/i2c"
)

// Button is the active-low front button on GPIO41.
var Button = gpio.NewButton(esp32s3.GPIO(41), gpio.PullNone, true)

var clockSource = esp32s3.SystemTimer{}

// Clock is the board monotonic clock and busy-wait timer.
var Clock = clock.New(&clockSource)

// Random is the hardware entropy source.
var Random = esp32s3.Random{}

// RGB is the addressable status pixel on GPIO35.
var RGB = esp32s3.NewWS2812(esp32s3.GPIO(35), nil)

var groveController = i2c.NewBitBang(esp32s3.GPIO(2), esp32s3.GPIO(1), &Clock, 100000)

// Grove is the board's four-pin Grove connector configured for I2C: GPIO2 SDA
// and GPIO1 SCL.
var Grove = i2c.DefinePort(&groveController, &Clock)
