//go:build m5nanoc6

// Package board describes the hardware attached to an M5Stack NanoC6.
package board

import (
	"renvo.dev/device/clock"
	"renvo.dev/device/esp32c6"
	"renvo.dev/device/gpio"
	"renvo.dev/device/i2c"
	"renvo.dev/device/uart"
	"renvo.dev/device/ws2812"
)

// BlueLED is the active-high blue indicator connected to GPIO7.
var BlueLED = gpio.NewLED(esp32c6.GPIO(7), false)

// LED is the board status indicator.
var LED = BlueLED

// Button is the active-low front button on GPIO9. Its pull-up also preserves
// the normal boot strapping level when released.
var Button = gpio.NewButton(esp32c6.GPIO(9), gpio.PullUp, true)

var clockSource = esp32c6.SystemTimer{}

// Clock is the board monotonic clock and busy-wait timer.
var Clock = clock.New(&clockSource)

// Random is the hardware entropy source.
var Random = esp32c6.Random{}

// RGB is the addressable pixel on GPIO20, powered through GPIO19.
var RGB = ws2812.New(esp32c6.GPIO(20), esp32c6.GPIO(19))

var groveData = esp32c6.GPIO(2)
var groveClock = esp32c6.GPIO(1)
var groveController = i2c.NewBitBang(groveData, groveClock, &Clock, 100000)
var groveUARTController = esp32c6.NewUART1(groveData)

// GroveData is the data/SDA signal on the Grove connector. Passing it to
// ws2812.New selects the ESP32-C6 RMT transmitter without exposing chip pins
// to application code. It cannot be used concurrently with Grove as I2C.
var GroveData ws2812.Output = groveData

// Grove is the board's four-pin Grove I2C connector: GPIO2 SDA and GPIO1 SCL.
var Grove = i2c.DefinePort(&groveController, &Clock)

// GroveUART is the transmit-capable serial signal on the Grove connector's
// yellow wire (GPIO2). It cannot be used concurrently with Grove or GroveData.
var GroveUART = uart.DefinePort(&groveUARTController)
