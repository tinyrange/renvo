package main

import "renvo.dev/examples/m5nanoc6/board"

// The oracle uses a shorter delay than the human-visible example. Reaching a
// GPIO7 rising edge proves that startup, calls, loops, volatile loads, and MMIO
// stores all executed successfully without making the test artificially slow.
func main() {
	board.ConfigureBlueLED()
	board.Delay(5000)
	board.SetBlueLED(true)
	for {
	}
}
