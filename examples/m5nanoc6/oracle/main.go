package main

import (
	"renvo.dev/device/board"
)

// The oracle uses a shorter delay than the human-visible example. Reaching a
// GPIO7 rising edge proves that startup, calls, loops, volatile loads, and MMIO
// stores all executed successfully without making the test artificially slow.
func main() {
	board.Clock.DelayMicroseconds(100)
	board.LED.Set(true)
	for {
	}
}
