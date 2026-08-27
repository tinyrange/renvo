package main

import "renvo.dev/device/board"

func main() {
	for {
		board.RGB.Set(32, 0, 0)
		board.Clock.DelayMilliseconds(500)
		board.RGB.Set(0, 32, 0)
		board.Clock.DelayMilliseconds(500)
		board.RGB.Set(0, 0, 32)
		board.Clock.DelayMilliseconds(500)
	}
}
