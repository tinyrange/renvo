package main

import (
	"renvo.dev/device/app"
	board "renvo.dev/device/board/m5nanoc6"
)

type blinker struct {
	on bool
}

func (led *blinker) Setup() {
	led.on = false
	board.BlueLED.Set(false)
}

func (led *blinker) Loop() {
	board.Clock.DelayMilliseconds(250)
	led.on = !led.on
	board.BlueLED.Set(led.on)
}

func main() {
	app.Run([]app.Component{&blinker{}})
}
