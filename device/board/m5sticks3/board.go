// Package board describes the hardware attached to an M5Stack StickS3.
package board

import (
	"renvo.dev/device/clock"
	"renvo.dev/device/esp32s3"
	"renvo.dev/device/gpio"
	"renvo.dev/std/graphics"
)

// FrontButton is the active-low front button on GPIO11.
var FrontButton = gpio.NewButton(esp32s3.GPIO(11), gpio.PullNone, true)

// SideButton is the active-low side button on GPIO12.
var SideButton = gpio.NewButton(esp32s3.GPIO(12), gpio.PullNone, true)

var clockSource = esp32s3.SystemTimer{}

// Clock is the board monotonic clock and busy-wait timer.
var Clock = clock.New(&clockSource)

// Screen exposes the board-attached ST7789 without leaking its SPI and GDMA
// wiring.
type Screen struct {
	ready bool
}

// Display is the attached 135x240 LCD.
var Display = Screen{}

func (*Screen) Width() int  { return displayWidth }
func (*Screen) Height() int { return displayHeight }

// Initialize powers and configures the display once. Ordinary display
// operations call it automatically, so applications have no hidden setup
// prerequisite.
func (s *Screen) Initialize() bool {
	if !s.ready {
		s.ready = initializeDisplay()
	}
	return s.ready
}

func (s *Screen) Fill(red, green, blue byte) bool {
	if !s.Initialize() {
		return false
	}
	fillDisplay(red, green, blue)
	return true
}
func (s *Screen) PresentRGBA(pixels []byte, stride, x0, y0, x1, y1 int) bool {
	if !s.Initialize() {
		return false
	}
	return presentRGBA(pixels, stride, x0, y0, x1, y1)
}
func (s *Screen) PresentRGBA2x(pixels []byte, stride, x0, y0, x1, y1 int) bool {
	if !s.Initialize() {
		return false
	}
	return presentRGBA2x(pixels, stride, x0, y0, x1, y1)
}
func (s *Screen) PresentRGBA3x(pixels []byte, stride, x0, y0, x1, y1 int) bool {
	if !s.Initialize() {
		return false
	}
	return presentRGBA3x(pixels, stride, x0, y0, x1, y1)
}
func (s *Screen) PresentSurface2x(surface *graphics.Surface) bool {
	if !s.Initialize() {
		return false
	}
	return presentSurface2x(surface)
}
func (s *Screen) PresentSurface3x(surface *graphics.Surface) bool {
	if !s.Initialize() {
		return false
	}
	return presentSurface3x(surface)
}
func (s *Screen) DrawLineDiagnostic() bool {
	if !s.Initialize() {
		return false
	}
	lcdDrawLines()
	return true
}
func (s *Screen) DrawButtonRectangle(index int, visible bool) bool {
	if !s.Initialize() {
		return false
	}
	drawButtonRectangle(index, visible)
	return true
}
