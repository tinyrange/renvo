package main

import (
	"renvo.dev/examples/m5nanoc6/board"
	"renvo.dev/examples/m5nanoc6/sgp30"
)

const maximumBrightness = uint32(64)

func scale(value uint16, good uint16, poor uint16) uint32 {
	if value <= good {
		return 0
	}
	if value >= poor {
		return 1024
	}
	return uint32(value-good) * 1024 / uint32(poor-good)
}

// airQualityColor maps 0..600 ppb TVOC continuously from green through orange
// to red. This is a deliberately simple display scale, not a health or
// regulatory threshold. The sensor's eCO2 output is derived from its VOC
// signal, so it is not treated as a second independent measurement here.
func airQualityColor(tvoc uint16) (uint8, uint8, uint8) {
	quality := scale(tvoc, 0, 600)
	red := uint32(0)
	green := maximumBrightness
	if quality <= 512 {
		red = quality * maximumBrightness / 512
		green = maximumBrightness - quality*(maximumBrightness/2)/512
	} else {
		red = maximumBrightness
		green = (1024 - quality) * (maximumBrightness / 2) / 512
	}
	return uint8(red), uint8(green), 0
}

func main() {
	board.ConfigureRGB()
	board.SetRGB(0, 0, 0)

	for {
		if !sgp30.Initialize() {
			board.SetRGB(32, 0, 32)
			board.DelayMilliseconds(1000)
			continue
		}

		for {
			started := board.TimerTicks()
			tvoc := uint16(0)
			if !sgp30.Measure(&tvoc) {
				board.SetRGB(32, 0, 32)
				board.DelayUntil(started, 1000000)
				break
			}
			red, green, blue := airQualityColor(tvoc)
			board.SetRGB(red, green, blue)

			board.DelayUntil(started, 1000000)
		}
	}
}
