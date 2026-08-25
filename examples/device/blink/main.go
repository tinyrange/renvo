package main

import (
	"renvo.dev/device/board"
)

func main() {
	for {
		board.Clock.DelayMilliseconds(500)
		board.BlueLED.Set(true)
		board.Clock.DelayMilliseconds(500)
		board.BlueLED.Set(false)
	}
}
