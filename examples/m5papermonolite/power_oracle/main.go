package main

import "renvo.dev/device/board"

func main() {
	if board.Initialize() != nil {
		for {
		}
	}
	board.Clock.DelayMilliseconds(500)
	if _, err := board.Power.Probe(); err != nil {
		print("RENVO PAPERMONO-LITE PHASE2 IDENTIFY FAIL\n")
		for {
		}
	}
	if err := board.Power.DisableDisplayAndTouch(); err != nil {
		print("RENVO PAPERMONO-LITE PHASE2 INITIAL SHUTDOWN FAIL\n")
		for {
		}
	}
	if err := board.Power.EnableDisplayAndTouch(); err != nil {
		print("RENVO PAPERMONO-LITE PHASE2 POWER FAIL\n")
		for {
		}
	}
	if err := board.Power.DisableDisplayAndTouch(); err != nil {
		print("RENVO PAPERMONO-LITE PHASE2 SHUTDOWN FAIL\n")
		for {
		}
	}
	print("RENVO PAPERMONO-LITE PHASE2 IDENTIFY PASS\n")
	print("RENVO PAPERMONO-LITE PHASE2 POWER PASS\n")
	print("RENVO PAPERMONO-LITE BUTTONS READY\n")

	buttonA := board.ButtonA.Pressed()
	buttonB := board.ButtonB.Pressed()
	for {
		nextA := board.ButtonA.Pressed()
		nextB := board.ButtonB.Pressed()
		if nextA != buttonA {
			buttonA = nextA
			if buttonA {
				print("BUTTON A DOWN\n")
			} else {
				print("BUTTON A UP\n")
			}
		}
		if nextB != buttonB {
			buttonB = nextB
			if buttonB {
				print("BUTTON B DOWN\n")
			} else {
				print("BUTTON B UP\n")
			}
		}
		board.Clock.DelayMilliseconds(10)
	}
}
