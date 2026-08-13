package main

import board "renvo.dev/device/board/m5cardputeradv"

func main() {
	if !board.Display.DrawLineDiagnostic() {
		for {
			print("FAIL: display initialization\n")
			board.Clock.DelayMilliseconds(1000)
		}
	}
	if err := board.Keyboard.Initialize(); err != nil {
		for {
			print("FAIL: keyboard initialization\n")
			board.Clock.DelayMilliseconds(1000)
		}
	}

	print("PASS: Cardputer Adv display and keyboard ready\n")
	for {
		event, ok, err := board.Keyboard.NextEvent()
		if err != nil {
			print("FAIL: keyboard transaction\n")
			board.Clock.DelayMilliseconds(100)
			continue
		}
		if !ok {
			board.Clock.DelayMilliseconds(2)
			continue
		}
		board.Display.DrawKey(event.Row, event.Column, event.Pressed)
		if event.Pressed {
			if event.Character != 0 {
				character := [1]byte{event.Character}
				print("KEY ")
				print(string(character[:]))
				print(" DOWN\n")
			} else {
				print("KEY SPECIAL DOWN\n")
			}
		} else {
			print("KEY UP\n")
		}
	}
}
