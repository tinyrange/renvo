//go:build m5nanoc6

package board

import "renvo.dev/device/ws2812"

func Example() {
	BlueLED.Set(true)
	Clock.DelayMilliseconds(100)
	BlueLED.Set(false)

	RGB.SetPixels([]ws2812.RGB{
		{Red: 32},
		{Green: 32},
	})
}
