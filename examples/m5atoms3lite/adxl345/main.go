package main

import (
	board "renvo.dev/device/board/m5atoms3lite"
	"renvo.dev/device/i2c"
	"renvo.dev/device/sensor/adxl345"
)

func main() {
	board.RGB.Set(0, 0, 0)
	sensor := adxl345.New(i2c.New(board.Grove), adxl345.AddressLow)

	for {
		if err := sensor.Initialize(); err != nil {
			board.RGB.Set(0, 0, 32)
			print("ADXL345 initialization failed: ", err.Error(), "\n")
			board.Clock.DelayMilliseconds(1000)
			continue
		}

		for {
			reading, err := sensor.Read()
			if err != nil {
				board.RGB.Set(32, 0, 0)
				print("ADXL345 read failed: ", err.Error(), "\n")
				break
			}
			board.RGB.Set(0, 32, 0)
			print("x=", reading.X, ", y=", reading.Y, ", z=", reading.Z, "\n")
			board.Clock.DelayMilliseconds(1000)
		}
	}
}
