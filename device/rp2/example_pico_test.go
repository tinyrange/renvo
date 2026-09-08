//go:build pico

package rp2_test

import "renvo.dev/device/rp2"

// NewLED binds the Raspberry Pi Pico's onboard LED to GPIO25.
func ExampleNewLED() {
	led := rp2.NewLED(rp2.GPIO(25), false)
	led.Set(true)
}
