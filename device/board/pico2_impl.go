//go:build pico2 && rp2350

// Package board describes the hardware attached to a Raspberry Pi Pico 2.
package board

import (
	"renvo.dev/device/clock"
	"renvo.dev/device/gpio"
	"renvo.dev/device/rp2350"
)

// BlueLED is the active-high indicator connected to GPIO25.
var BlueLED = gpio.NewLED(rp2350.GPIO(25), false)

var clockSource = rp2350.SystemTimer{}

// Clock is the board monotonic clock and busy-wait timer.
var Clock = clock.New(&clockSource)
