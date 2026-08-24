package main

import (
	board "renvo.dev/device/board/m5nanoc6"
	"renvo.dev/device/ws2812"
)

const (
	pixelCount        = 15
	frameMilliseconds = 75
)

var pixels [pixelCount]ws2812.RGB

func main() {
	strip := ws2812.New(board.GroveData, nil)
	position := 0
	for {
		for i := 0; i < len(pixels); i++ {
			pixels[i] = ws2812.RGB{}
		}
		pixels[position] = ws2812.RGB{Red: 32, Green: 16, Blue: 4}
		strip.SetPixels(pixels[:])

		position++
		if position == len(pixels) {
			position = 0
		}
		board.Clock.DelayMilliseconds(frameMilliseconds)
	}
}
