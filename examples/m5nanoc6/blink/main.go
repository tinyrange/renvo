package main

import "renvo.dev/examples/m5nanoc6/board"

func main() {
	board.ConfigureBlueLED()
	for {
		board.DelayMilliseconds(500)
		board.SetBlueLED(true)
		board.DelayMilliseconds(500)
		board.SetBlueLED(false)
	}
}
