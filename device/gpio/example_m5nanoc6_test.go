//go:build m5nanoc6

package gpio_test

import (
	"renvo.dev/device/board"
	"renvo.dev/device/esp32c6"
	"renvo.dev/device/gpio"
)

func ExampleLED_Set() {
	var led *gpio.LED = &board.BlueLED
	led.Set(true)
	board.Clock.DelayMilliseconds(100)
	led.Set(false)
}

func ExampleNewButton() {
	button := gpio.NewButton(esp32c6.GPIO(9), gpio.PullUp, true)
	if button.Pressed() {
		// Handle the active-low button press.
	}
}

func ExampleButton_Pressed() {
	if board.Button.Pressed() {
		board.BlueLED.Toggle()
	}
}

func ExampleNewLED() {
	status := gpio.NewLED(esp32c6.GPIO(7), false) // Active-high.
	status.Set(true)
}

func ExampleLED_Toggle() {
	board.BlueLED.Toggle()
}

// Config is useful when a driver needs a raw pin rather than a Button or LED.
func ExampleConfig() {
	interrupt := esp32c6.GPIO(2)
	_ = interrupt.Configure(gpio.Config{
		Direction: gpio.Input,
		Pull:      gpio.PullUp,
	})
}

// Pin lets drivers accept digital IO without depending on a particular chip.
func ExamplePin() {
	var enable gpio.Pin = esp32c6.GPIO(2)
	_ = enable.Configure(gpio.Config{Direction: gpio.Output})
	enable.Set(true)
}
