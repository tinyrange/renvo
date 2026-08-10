package main

import "renvo.dev/examples/m5sticks3/board"

func main() {
	if !board.DrawLineDiagnostic() {
		for {
			print("FAIL: M5PM1 identity or acknowledgement\n")
			board.DelayMilliseconds(1000)
		}
	}
	board.ConfigureButton(11)
	board.ConfigureButton(12)
	buttonAWasPressed := false
	buttonBWasPressed := false
	buttonARectangleVisible := false
	buttonBRectangleVisible := false
	print("PASS: LCD and buttons ready\n")
	for {
		buttonAIsPressed := board.ButtonPressed(11)
		buttonBIsPressed := board.ButtonPressed(12)
		if buttonAIsPressed != buttonAWasPressed {
			board.DelayMilliseconds(20)
			buttonAIsPressed = board.ButtonPressed(11)
			if buttonAIsPressed != buttonAWasPressed {
				buttonAWasPressed = buttonAIsPressed
				if buttonAIsPressed {
					buttonARectangleVisible = !buttonARectangleVisible
					board.DrawButtonRectangle(0, buttonARectangleVisible)
				}
			}
		}
		if buttonBIsPressed != buttonBWasPressed {
			board.DelayMilliseconds(20)
			buttonBIsPressed = board.ButtonPressed(12)
			if buttonBIsPressed != buttonBWasPressed {
				buttonBWasPressed = buttonBIsPressed
				if buttonBIsPressed {
					buttonBRectangleVisible = !buttonBRectangleVisible
					board.DrawButtonRectangle(1, buttonBRectangleVisible)
				}
			}
		}
		board.DelayMilliseconds(10)
	}
}
