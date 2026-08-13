package main

import (
	board "renvo.dev/device/board/m5nanoc6"
)

func main() {
	for {
		board.Clock.DelayMilliseconds(500)
		board.BlueLED.Set(true)
		board.Clock.DelayMilliseconds(500)
		board.BlueLED.Set(false)
	}
}
