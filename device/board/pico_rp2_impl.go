//go:build (pico || pico2) && rp2

// Package board describes Raspberry Pi Pico compatibility hardware.
package board

import (
	"renvo.dev/device/rp2"
)

// BlueLED is the active-high indicator connected to GPIO25.
var BlueLED = rp2.GPIO(25)

// LED is the board status indicator.
var LED = BlueLED

// Clock is the board monotonic clock and busy-wait timer.
var Clock = rp2.Clock{}
