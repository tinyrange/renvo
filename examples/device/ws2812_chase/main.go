package main

import (
	"renvo.dev/device/board"
	"renvo.dev/device/ws2812"
)

const (
	pixelCount       = 15
	cometCount       = 8
	tailLength       = 3
	pollMilliseconds = 2
	stepMilliseconds = 45
	debounceSamples  = 3
)

type comet struct {
	color    ws2812.RGB
	position int
	active   bool
}

var pixels [pixelCount]ws2812.RGB
var comets [cometCount]comet

func mix(value uint32) uint32 {
	value = value ^ value<<13
	value = value ^ value>>17
	return value ^ value<<5
}

func randomColor(timing uint32) ws2812.RGB {
	value := mix(board.Random.Uint32() ^ timing)
	color := ws2812.RGB{
		Red:   uint8(value>>16)&31 + 8,
		Green: uint8(value>>8)&31 + 8,
		Blue:  uint8(value)&31 + 8,
	}
	switch value % 3 {
	case 0:
		color.Red = color.Red | 32
	case 1:
		color.Green = color.Green | 32
	default:
		color.Blue = color.Blue | 32
	}
	return color
}

func launch(color ws2812.RGB) {
	for index := 0; index < len(comets); index++ {
		if !comets[index].active {
			comets[index] = comet{color: color, position: 0, active: true}
			return
		}
	}
}

func advance() {
	for index := 0; index < len(comets); index++ {
		if comets[index].active {
			comets[index].position++
			if comets[index].position-tailLength >= len(pixels) {
				comets[index].active = false
			}
		}
	}
}

func add(into, value uint8) uint8 {
	total := uint16(into) + uint16(value)
	if total > 255 {
		return 255
	}
	return uint8(total)
}

func blend(position int, color ws2812.RGB, shift uint8) {
	if position < 0 || position >= len(pixels) {
		return
	}
	pixel := &pixels[position]
	pixel.Red = add(pixel.Red, color.Red>>shift)
	pixel.Green = add(pixel.Green, color.Green>>shift)
	pixel.Blue = add(pixel.Blue, color.Blue>>shift)
}

func render(strip *ws2812.Strip) {
	for index := 0; index < len(pixels); index++ {
		pixels[index] = ws2812.RGB{}
	}
	for index := 0; index < len(comets); index++ {
		if comets[index].active {
			blend(comets[index].position, comets[index].color, 0)
			blend(comets[index].position-1, comets[index].color, 1)
			blend(comets[index].position-2, comets[index].color, 2)
			blend(comets[index].position-3, comets[index].color, 3)
		}
	}
	strip.SetPixels(pixels[:])
}

func main() {
	strip := ws2812.New(board.GroveData, nil)
	render(&strip)

	stablePressed := false
	candidatePressed := false
	candidateSamples := 0
	stepElapsed := uint32(0)
	timing := uint32(0)
	for {
		pressed := board.Button.Pressed()
		if pressed == candidatePressed {
			if candidateSamples < debounceSamples {
				candidateSamples++
			}
		} else {
			candidatePressed = pressed
			candidateSamples = 1
		}

		dirty := false
		if candidateSamples == debounceSamples && stablePressed != candidatePressed {
			stablePressed = candidatePressed
			if stablePressed {
				launch(randomColor(timing))
				dirty = true
			}
		}

		stepElapsed += pollMilliseconds
		if stepElapsed >= stepMilliseconds {
			stepElapsed -= stepMilliseconds
			advance()
			dirty = true
		}
		if dirty {
			render(&strip)
		}

		timing++
		board.Clock.DelayMilliseconds(pollMilliseconds)
	}
}
