//go:build m5nanoc6

package bme688_test

import (
	"renvo.dev/device/board"
	"renvo.dev/device/i2c"
	"renvo.dev/device/sensor/bme688"
)

func ExampleNew() {
	sensor := bme688.New(i2c.New(board.Grove), bme688.AddressHigh)
	if err := sensor.Initialize(); err != nil {
		return
	}
	reading, _ := sensor.Read()
	_ = reading
}
