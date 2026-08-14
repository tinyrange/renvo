package main

import (
	board "renvo.dev/device/board/m5atoms3lite"
	"renvo.dev/device/esp32s3"
	"renvo.dev/device/ws2812"
)

const (
	pixelCount       = 15
	shotCount        = 32
	pollMilliseconds = 2
	stepMilliseconds = 35
	debounceSamples  = 3
)

type shot struct {
	color    ws2812.RGB
	position int
	active   bool
}

var (
	pixels [pixelCount]ws2812.RGB
	shots  [shotCount]shot
)

func mix(value uint32) uint32 {
	value = value ^ value<<13
	value = value ^ value>>17
	return value ^ value<<5
}

func randomColor(timing uint32) ws2812.RGB {
	value := mix(board.Random.Uint32() ^ timing)
	color := ws2812.RGB{
		Red:   uint8(value>>16)&31 + 2,
		Green: uint8(value>>8)&31 + 2,
		Blue:  uint8(value)&31 + 2,
	}
	// Keep every shot visibly colored, including RNG values near black.
	switch value % 3 {
	case 0:
		color.Red = color.Red | 16
	case 1:
		color.Green = color.Green | 16
	default:
		color.Blue = color.Blue | 16
	}
	return color
}

func launch(color ws2812.RGB) {
	for i := 0; i < len(shots); i++ {
		if !shots[i].active {
			shots[i] = shot{color: color, position: 0, active: true}
			return
		}
	}
}

func advance() {
	for i := 0; i < len(shots); i++ {
		if shots[i].active {
			shots[i].position++
			if shots[i].position == len(pixels) {
				shots[i].active = false
			}
		}
	}
}

func render(strip *ws2812.Strip) {
	for i := 0; i < len(pixels); i++ {
		pixels[i] = ws2812.RGB{}
	}
	for i := 0; i < len(shots); i++ {
		if shots[i].active {
			pixels[shots[i].position] = shots[i].color
		}
	}
	strip.SetPixels(pixels[:])
}

func main() {
	strip := esp32s3.NewWS2812(esp32s3.GPIO(2), nil)
	render(&strip)

	stablePressed := false
	candidatePressed := false
	candidateSamples := 0
	stepElapsed := uint32(0)
	timing := uint32(0)
	for {
		pressed := board.Button.Pressed()
		if pressed == candidatePressed {
			candidateSamples++
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
