//go:build (pico || pico2) && rp2

// Package board describes Raspberry Pi Pico compatibility hardware.
package board

import (
	"renvo.dev/device/rp2"
)

// LED is the active-high status indicator connected to GPIO25.
var LED = rp2.GPIO(25)

// Clock is the board monotonic clock and busy-wait timer.
var Clock = rp2.Clock{}
