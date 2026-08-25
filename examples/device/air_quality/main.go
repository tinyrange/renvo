package main

import (
	"renvo.dev/device/board"
	"renvo.dev/device/i2c"
	"renvo.dev/device/sensor/sgp30"
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
	board.RGB.Set(0, 0, 0)
	air := sgp30.New(i2c.New(board.Grove))

	for {
		if air.Initialize() != nil {
			board.RGB.Set(32, 0, 32)
			board.Clock.DelayMilliseconds(1000)
			continue
		}

		for {
			started := board.Clock.Ticks()
			tvoc, err := air.Measure()
			if err != nil {
				board.RGB.Set(32, 0, 32)
				board.Clock.DelayUntil(started, 1000000)
				break
			}
			red, green, blue := airQualityColor(tvoc)
			board.RGB.Set(red, green, blue)

			board.Clock.DelayUntil(started, 1000000)
		}
	}
}
