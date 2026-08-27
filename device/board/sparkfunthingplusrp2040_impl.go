//go:build sparkfunthingplusrp2040 && rp2

// Package board describes the SparkFun Thing Plus RP2040.
package board

import (
	"renvo.dev/device/rp2"
	"renvo.dev/device/ws2812"
)

// BlueLED is the active-high status indicator on GPIO25.
var BlueLED = rp2.NewLED(rp2.GPIO(25), false)

// Clock is the board monotonic clock and busy-wait timer.
var Clock = rp2.Clock{}

// RGB is the onboard WS2812 addressable pixel on GPIO8.
var RGB = ws2812.New(rp2.GPIO(8), nil)
