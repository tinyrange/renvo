// Package board describes the hardware attached to an M5Stack NanoC6.
package board

import (
	"renvo.dev/device/clock"
	"renvo.dev/device/esp32c6"
	"renvo.dev/device/gpio"
	"renvo.dev/device/i2c"
)

// BlueLED is the active-high blue indicator connected to GPIO7.
var BlueLED = gpio.NewLED(esp32c6.GPIO(7), false)

// Button is the active-low front button on GPIO9. Its pull-up also preserves
// the normal boot strapping level when released.
var Button = gpio.NewButton(esp32c6.GPIO(9), gpio.PullUp, true)

var clockSource = esp32c6.SystemTimer{}

// Clock is the board monotonic clock and busy-wait timer.
var Clock = clock.New(&clockSource)

// Random is the hardware entropy source.
var Random = esp32c6.Random{}

// RGB is the addressable pixel on GPIO20, powered through GPIO19.
var RGB = esp32c6.NewWS2812(esp32c6.GPIO(20), esp32c6.GPIO(19))

var groveController = i2c.NewBitBang(esp32c6.GPIO(2), esp32c6.GPIO(1), &Clock, 100000)

// Grove is the board's four-pin Grove I2C connector: GPIO2 SDA and GPIO1 SCL.
var Grove = i2c.DefinePort(&groveController, &Clock)
