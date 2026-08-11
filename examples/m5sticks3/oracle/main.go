package main

import board "renvo.dev/device/board/m5sticks3"

func main() {
	if !board.Display.DrawLineDiagnostic() {
		for {
			print("FAIL: M5PM1 identity or acknowledgement\n")
			board.Clock.DelayMilliseconds(1000)
		}
	}
	buttonAWasPressed := false
	buttonBWasPressed := false
	buttonARectangleVisible := false
	buttonBRectangleVisible := false
	print("PASS: LCD and buttons ready\n")
	for {
		buttonAIsPressed := board.FrontButton.Pressed()
		buttonBIsPressed := board.SideButton.Pressed()
		if buttonAIsPressed != buttonAWasPressed {
			board.Clock.DelayMilliseconds(20)
			buttonAIsPressed = board.FrontButton.Pressed()
			if buttonAIsPressed != buttonAWasPressed {
				buttonAWasPressed = buttonAIsPressed
				if buttonAIsPressed {
					buttonARectangleVisible = !buttonARectangleVisible
					board.Display.DrawButtonRectangle(0, buttonARectangleVisible)
				}
			}
		}
		if buttonBIsPressed != buttonBWasPressed {
			board.Clock.DelayMilliseconds(20)
			buttonBIsPressed = board.SideButton.Pressed()
			if buttonBIsPressed != buttonBWasPressed {
				buttonBWasPressed = buttonBIsPressed
				if buttonBIsPressed {
					buttonBRectangleVisible = !buttonBRectangleVisible
					board.Display.DrawButtonRectangle(1, buttonBRectangleVisible)
				}
			}
		}
		board.Clock.DelayMilliseconds(10)
	}
}
