// Package board describes the hardware attached to an M5Stack StickS3.
package board

import (
	"renvo.dev/device/esp32s3"
	"renvo.dev/device/gpio"
	"renvo.dev/device/usb"
	legacy "renvo.dev/examples/m5sticks3/board"
	"renvo.dev/std/graphics"
)

// FrontButton is the active-low front button on GPIO11.
var FrontButton = gpio.NewButton(esp32s3.GPIO(11), gpio.PullNone, true)

// SideButton is the active-low side button on GPIO12.
var SideButton = gpio.NewButton(esp32s3.GPIO(12), gpio.PullNone, true)

// Clock is the board monotonic clock and busy-wait timer.
var Clock = esp32s3.SystemTimer{}

var usbController = esp32s3.DWC2{}

// USB is the native USB-C device connection on GPIO19/GPIO20.
var USB = usb.DefinePort(&usbController)

// Screen exposes the board-attached ST7789 without leaking its SPI and GDMA
// wiring. Its temporary legacy calls are removed when that driver moves below
// the board package in the next migration slice.
type Screen struct{}

// Display is the attached 135x240 LCD.
var Display = Screen{}

func (*Screen) Width() int  { return legacy.DisplayWidth }
func (*Screen) Height() int { return legacy.DisplayHeight }
func (*Screen) Initialize() bool {
	return legacy.InitializeDisplay()
}
func (*Screen) PresentSurface2x(surface *graphics.Surface) bool {
	return legacy.PresentSurface2x(surface)
}
func (*Screen) DrawLineDiagnostic() bool { return legacy.DrawLineDiagnostic() }
func (*Screen) DrawButtonRectangle(index int, visible bool) {
	legacy.DrawButtonRectangle(index, visible)
}
