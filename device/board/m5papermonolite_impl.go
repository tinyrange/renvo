//go:build m5papermonolite

// Package board describes the hardware attached to an M5Stack PaperMono-Lite.
package board

import (
	"renvo.dev/device/clock"
	"renvo.dev/device/esp32s3"
	"renvo.dev/device/gpio"
)

var buttonAPin = esp32s3.GPIO(2)
var buttonBPin = esp32s3.GPIO(3)

// ButtonA is the active-low user button on GPIO2.
var ButtonA = gpio.NewButton(buttonAPin, gpio.PullNone, true)

// ButtonB is the active-low user button on GPIO3.
var ButtonB = gpio.NewButton(buttonBPin, gpio.PullNone, true)

var clockSource = esp32s3.SystemTimer{}

// Clock is the board monotonic clock and busy-wait timer.
var Clock = clock.New(&clockSource)

// Random is the hardware entropy source.
var Random = esp32s3.Random{}

// Initialize establishes the safe Phase 1 GPIO baseline. Display power, reset,
// SPI, and the shared internal I2C bus remain untouched until their board
// drivers can enforce the required sequencing and timeouts.
func Initialize() error {
	input := gpio.Config{Direction: gpio.Input, Pull: gpio.PullNone}
	if err := buttonAPin.Configure(input); err != nil {
		return err
	}
	return buttonBPin.Configure(input)
}
