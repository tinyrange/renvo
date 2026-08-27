//go:build m5nanoc6

package ws2812_test

import (
	"renvo.dev/device/board"
	"renvo.dev/device/esp32c6"
	"renvo.dev/device/ws2812"
)

func ExampleStrip_SetPixels() {
	board.RGB.SetPixels([]ws2812.RGB{
		{Red: 32},
		{Green: 32},
		{Blue: 32},
	})
}

// New selects the chip-specific transmitter from an output-capable data pin.
func ExampleNew() {
	strip := ws2812.New(esp32c6.GPIO(20), esp32c6.GPIO(19))
	strip.Set(32, 0, 0)
}

// NewTransmitter is useful when a board package has already selected an RMT channel.
func ExampleNewTransmitter() {
	transmitter := esp32c6.GPIO(20).WS2812Transmitter(esp32c6.GPIO(19))
	strip := ws2812.NewTransmitter(transmitter)
	strip.Set(0, 32, 0)
}

func ExampleStrip_Set() {
	board.RGB.Set(8, 16, 32)
}
