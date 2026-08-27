package main

import (
	"renvo.dev/device/app"
	"renvo.dev/device/board"
)

type blinker struct {
	on bool
}

func (led *blinker) Setup() {
	led.on = false
	board.LED.Set(false)
}

func (led *blinker) Loop() {
	board.Clock.DelayMilliseconds(250)
	led.on = !led.on
	board.LED.Set(led.on)
}

func main() {
	app.Run([]app.Component{&blinker{}})
}
