package main

import "example.com/rtgtests/regressions/issue154/widget"

func main() {
	window := widget.NewWindow(widget.Options{Title: "unused", Width: 6, Height: 7})
	if window != nil && window.Area() == 42 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
