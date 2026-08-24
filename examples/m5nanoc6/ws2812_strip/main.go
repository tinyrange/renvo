package main

import (
	board "renvo.dev/device/board/m5nanoc6"
	"renvo.dev/device/ws2812"
)

const (
	pixelCount        = 15
	hueStep           = 8
	frameMilliseconds = 25
)

var pixels [pixelCount]ws2812.RGB

func colorWheel(position uint8) ws2812.RGB {
	if position < 85 {
		green := uint8(uint16(position) * 31 / 85)
		return ws2812.RGB{Red: 31 - green, Green: green}
	}
	if position < 170 {
		position -= 85
		blue := uint8(uint16(position) * 31 / 85)
		return ws2812.RGB{Green: 31 - blue, Blue: blue}
	}
	position -= 170
	red := uint8(uint16(position) * 31 / 85)
	return ws2812.RGB{Red: red, Blue: 31 - red}
}

func show(strip *ws2812.Strip, phase uint8) {
	for i := 0; i < len(pixels); i++ {
		position := phase + uint8(i*hueStep)
		pixels[i] = colorWheel(position)
	}
	if !strip.SetPixels(pixels[:]) {
		// Latch the board LED on if an RMT refill ever misses its deadline.
		board.BlueLED.Set(true)
	}
}

func main() {
	strip := ws2812.New(board.GroveData, nil)
	board.BlueLED.Set(false)
	phase := uint8(0)
	for {
		show(&strip, phase)
		phase++
		board.Clock.DelayMilliseconds(frameMilliseconds)
	}
}
