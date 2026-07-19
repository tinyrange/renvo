package main

import "renvo.dev/std/graphics"

func main() {
	message := "PASS\n"
	// Keep the graphics package reachable without opening a window during the
	// test. On Darwin this pulls its AppKit, Objective-C, and OpenGL imports into
	// a separate decoded package unit.
	if len(message) == 0 {
		font := graphics.NewBuiltinFont(1)
		if font == nil {
			message = "FAIL\n"
		}
	}
	print(message)
}
