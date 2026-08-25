package main

import (
	"renvo.dev/device/board"
	"renvo.dev/device/i2c"
	"renvo.dev/device/sensor/bme688"
)

func printTwoDigits(value uint16) {
	if value < 10 {
		print("0")
	}
	print(value)
}

func printTemperature(value int16) {
	if value < 0 {
		print("-")
		value = -value
	}
	print(value/100, ".")
	printTwoDigits(uint16(value % 100))
}

func printHumidity(value uint32) {
	print(value/1000, ".")
	fraction := value % 1000
	if fraction < 100 {
		print("0")
	}
	printTwoDigits(uint16(fraction))
}

func printPressure(value uint32) {
	print(value/100, ".")
	printTwoDigits(uint16(value % 100))
}

func printGas(value uint32) {
	print(value/1000, ".")
	fraction := value % 1000
	if fraction < 100 {
		print("0")
	}
	printTwoDigits(uint16(fraction))
}

func main() {
	board.RGB.Set(0, 0, 0)
	sensor := bme688.New(i2c.New(board.Grove), bme688.AddressHigh)

	for {
		if err := sensor.Initialize(); err != nil {
			board.RGB.Set(0, 0, 32)
			print("BME688 initialization failed: ", err.Error(), "\n")
			board.Clock.DelayMilliseconds(1000)
			continue
		}

		for {
			var reading bme688.Reading
			if err := sensor.ReadInto(&reading); err != nil {
				board.RGB.Set(32, 0, 0)
				print("BME688 read failed: ", err.Error(), "\n")
				break
			}
			if reading.GasValid && reading.HeaterStable {
				board.RGB.Set(0, 32, 0)
			} else {
				board.RGB.Set(24, 12, 0)
			}
			// Arduino Serial Plotter-compatible labelled values. Renvo's browser
			// IDE recognizes the same format and opens its Plotter panel.
			print("Temperature_C:")
			printTemperature(reading.Temperature)
			print("\tPressure_hPa:")
			printPressure(reading.Pressure)
			print("\tHumidity_pct:")
			printHumidity(reading.Humidity)
			print("\tGas_kOhm:")
			printGas(reading.GasResistance)
			print("\n")
			board.Clock.DelayMilliseconds(857)
		}
	}
}
