package main

import "renvo.dev/examples/m5nanoc6/board"

func main() {
	board.ConfigureBlueLED()
	for {
		board.Delay(2000000)
		board.SetBlueLED(true)
		board.Delay(2000000)
		board.SetBlueLED(false)
	}
}
