package main

import (
	"fmt"

	"renvo.dev/device/uefi"
)

func main() {
	gop, status := uefi.GraphicsOutput()
	if status.Failed() {
		fmt.Println("Graphics Output Protocol: " + status.Error())
		return
	}
	frame, status := gop.Framebuffer()
	if status.Failed() {
		fmt.Println("Framebuffer: " + status.Error())
		return
	}

	for y := 0; y < int(frame.Height); y++ {
		for x := 0; x < int(frame.Width); x++ {
			red := byte((x * 255) / int(frame.Width))
			green := byte((y * 255) / int(frame.Height))
			blue := byte(((x + y) * 255) / int(frame.Width+frame.Height))
			frame.Set(x, y, red, green, blue)
		}
	}

	// Keep the image visible briefly, then return to the firmware menu.
	uefi.Stall(5000000)
}
