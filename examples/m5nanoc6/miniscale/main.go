package main

import (
	board "renvo.dev/device/board/m5nanoc6"
	"renvo.dev/device/i2c"
	"renvo.dev/device/sensor/miniscale"
)

func printHundredths(value int32) {
	if value < 0 {
		print("-")
		value = -value
	}
	print(value/100, ".")
	fraction := value % 100
	if fraction < 10 {
		print("0")
	}
	print(fraction)
}

func main() {
	board.RGB.Set(0, 0, 0)
	scales := miniscale.New(i2c.New(board.Grove))
	version, err := scales.FirmwareVersion()
	if err != nil {
		board.RGB.Set(32, 0, 0)
		print("Mini Scales not found: ", err.Error(), "\n")
		for {
			board.Clock.DelayMilliseconds(1000)
		}
	}

	board.RGB.Set(0, 24, 0)
	scales.SetLED(0, 24, 0)
	print("Mini Scales firmware v", version, " ready; press NanoC6 button to tare\n")

	buttonWasPressed := false
	for {
		buttonPressed := board.Button.Pressed()
		if buttonPressed && !buttonWasPressed {
			if err := scales.Tare(); err != nil {
				board.RGB.Set(32, 0, 0)
				print("Tare failed: ", err.Error(), "\n")
			} else {
				board.RGB.Set(0, 0, 24)
				print("Tared\n")
			}
		}
		buttonWasPressed = buttonPressed

		weight, weightErr := scales.ReadWeightHundredths()
		raw, rawErr := scales.ReadRaw()
		if weightErr != nil || rawErr != nil {
			board.RGB.Set(32, 0, 0)
			print("Scale read failed")
			if weightErr != nil {
				print(": ", weightErr.Error())
			} else {
				print(": ", rawErr.Error())
			}
			print("\n")
			board.Clock.DelayMilliseconds(500)
			continue
		}

		board.RGB.Set(0, 24, 0)
		print("Weight_g:")
		printHundredths(weight)
		print("\tRawADC:", raw, "\n")
		board.Clock.DelayMilliseconds(100)
	}
}
