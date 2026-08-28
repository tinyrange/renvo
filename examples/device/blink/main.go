package main

import (
	"renvo.dev/device/board"
)

func main() {
	for {
		board.Clock.DelayMilliseconds(500)
		board.LED.Set(true)
		board.Clock.DelayMilliseconds(500)
		board.LED.Set(false)
	}
}
