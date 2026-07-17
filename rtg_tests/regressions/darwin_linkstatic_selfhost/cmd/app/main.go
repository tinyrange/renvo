package main

import "j5.nz/rtg/rtg/std/graphics"

func main() {
	message := "PASS\n"
	// Keep the graphics package reachable without opening a window during the
	// test. On Darwin this pulls its AppKit, Objective-C, and OpenGL imports into
	// a separate decoded package unit.
	if len(message) == 0 {
		window := graphics.NewWindow(graphics.WindowOptions{Title: "unused", Width: 8, Height: 8})
		if window != nil {
			window.Close()
		}
	}
	print(message)
}
