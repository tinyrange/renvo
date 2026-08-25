//go:build m5cardputeradv

// Package board describes the hardware attached to an M5Stack Cardputer Adv.
package board

import (
	"renvo.dev/device/clock"
	"renvo.dev/device/esp32s3"
	"renvo.dev/device/i2c"
	"renvo.dev/device/input/tca8418"
	"renvo.dev/std/graphics"
)

var clockSource = esp32s3.SystemTimer{}

// Clock is the board monotonic clock and busy-wait timer.
var Clock = clock.New(&clockSource)

var keyboardController = i2c.NewBitBang(esp32s3.GPIO(8), esp32s3.GPIO(9), &clockSource, 100000)
var keyboardPort = i2c.DefinePort(&keyboardController, &clockSource)
var keyboardBus = i2c.New(keyboardPort)

// Keyboard is the board's 56-key TCA8418-backed keyboard.
var Keyboard = newKeyboard(tca8418.New(keyboardBus))

// Screen exposes the board-attached ST7789V2 without leaking its SPI and GDMA
// wiring.
type Screen struct {
	ready bool
}

// Display is the attached 240x135 landscape LCD.
var Display = Screen{}

// Width returns the display width in pixels.
func (*Screen) Width() int { return displayWidth }

// Height returns the display height in pixels.
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

// Fill paints the entire display with one RGB color.
func (s *Screen) Fill(red, green, blue byte) bool {
	if !s.Initialize() {
		return false
	}
	fillDisplay(red, green, blue)
	return true
}

// PresentRGBA copies an RGBA rectangle to the display at native scale.
func (s *Screen) PresentRGBA(pixels []byte, stride, x0, y0, x1, y1 int) bool {
	if !s.Initialize() {
		return false
	}
	return presentRGBA(pixels, stride, x0, y0, x1, y1)
}

// PresentRGBA2x copies an RGBA rectangle while doubling each source pixel.
func (s *Screen) PresentRGBA2x(pixels []byte, stride, x0, y0, x1, y1 int) bool {
	if !s.Initialize() {
		return false
	}
	return presentRGBA2x(pixels, stride, x0, y0, x1, y1)
}

// PresentRGBA3x copies an RGBA rectangle while tripling each source pixel.
func (s *Screen) PresentRGBA3x(pixels []byte, stride, x0, y0, x1, y1 int) bool {
	if !s.Initialize() {
		return false
	}
	return presentRGBA3x(pixels, stride, x0, y0, x1, y1)
}

// PresentSurface2x displays a graphics surface at double scale.
func (s *Screen) PresentSurface2x(surface *graphics.Surface) bool {
	if !s.Initialize() {
		return false
	}
	return presentSurface2x(surface)
}

// PresentSurface3x displays a graphics surface at triple scale.
func (s *Screen) PresentSurface3x(surface *graphics.Surface) bool {
	if !s.Initialize() {
		return false
	}
	return presentSurface3x(surface)
}

// InitializeTerminal returns a memory-efficient 120 by 67 surface for the
// embedded terminal package.
func (s *Screen) InitializeTerminal() (*graphics.Surface, bool) {
	if !s.Initialize() {
		return nil, false
	}
	return graphics.NewSurface(displayWidth/2, displayHeight/2), true
}

// PresentTerminal presents a terminal surface at double physical scale.
func (s *Screen) PresentTerminal(surface *graphics.Surface) bool {
	return s.PresentSurface2x(surface)
}

// DrawLineDiagnostic draws a border and color/alignment grid over the panel.
func (s *Screen) DrawLineDiagnostic() bool {
	if !s.Initialize() {
		return false
	}
	lcdDrawLines()
	return true
}

// DrawKey displays one cell in the keyboard's four-row by fourteen-column
// layout. The oracle uses it to verify the controller-to-keyboard remap without
// requiring a font engine.
func (s *Screen) DrawKey(row, column int, pressed bool) bool {
	if !s.Initialize() {
		return false
	}
	return drawKey(row, column, pressed)
}
