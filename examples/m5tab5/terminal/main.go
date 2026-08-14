package main

import (
	board "renvo.dev/device/board/m5tab5"
	"renvo.dev/device/i2c"
	"renvo.dev/device/sensor/adxl345"
	"renvo.dev/device/terminal"
	"renvo.dev/std/fmt"
	"renvo.dev/std/graphics"
)

func discardInput(console *terminal.Terminal) {
	var input [16]byte
	for {
		count, _ := console.Read(input[:])
		if count < len(input) {
			return
		}
	}
}

func main() {
	console, err := terminal.Start(&board.Display, terminal.Options{
		Scrollback:     48,
		Font:           graphics.NewBuiltinFont(3),
		CellWidth:      18,
		CellHeight:     30,
		Baseline:       21,
		Pointer:        &board.Touch,
		TouchKeyboard:  true,
		KeyboardHeight: 450,
		FlushPolicy:    terminal.FlushManual,
		Clock:          &board.Display,
	})
	if err != nil {
		for {
			print("Tab5 terminal initialization failed: ", err.Error(), "\n")
		}
	}

	bus := i2c.New(board.PortA())
	sensor := adxl345.New(bus, adxl345.AddressLow)
	if err := sensor.Initialize(); err != nil {
		sensor = adxl345.New(bus, adxl345.AddressHigh)
		if err = sensor.Initialize(); err != nil {
			fmt.Fprintf(console, "\x1b[31mADXL345 initialization failed: %s\x1b[0m", err)
			console.Flush()
			return
		}
	}

	console.Reset()
	var reading adxl345.Reading
	// The sensor produces data at 100 Hz by default. A 60 Hz cooperative loop
	// matches the display cadence without requesting duplicate display frames.
	for terminal.Tick(terminal.Second / 60) {
		discardInput(console)
		if err := sensor.ReadInto(&reading); err != nil {
			fmt.Fprintf(console, "\x1b[31mADXL345 read failed: %s\x1b[0m\r\n", err)
		} else {
			fmt.Fprintf(console,
				"\x1b[36mX=%d\x1b[0m, \x1b[35mY=%d\x1b[0m, \x1b[33mZ=%d\x1b[0m\r\n",
				reading.X, reading.Y, reading.Z,
			)
		}
		console.Flush()
	}
}
