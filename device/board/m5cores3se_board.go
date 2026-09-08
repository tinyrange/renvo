//go:build m5cores3se

package board

import (
	"renvo.dev/device/clock"
	"renvo.dev/device/esp32s3"
	"renvo.dev/device/i2c"
	"renvo.dev/std/graphics"
)

var clockSource = esp32s3.SystemTimer{}
var Clock = clock.New(&clockSource)
var coreController = i2c.NewBitBang(esp32s3.GPIO(12), esp32s3.GPIO(11), &Clock, 100000)
var coreBus = i2c.New(i2c.DefinePort(&coreController, &Clock))

// Screen owns the 320x240 ILI9342C and SPI3/GDMA channel 0.
type Screen struct{ ready bool }

var Display = Screen{}

func (*Screen) Width() int  { return displayWidth }
func (*Screen) Height() int { return displayHeight }
func (s *Screen) Initialize() bool {
	if !s.ready {
		if esp32s3.Use160MHzClock() != nil {
			return false
		}
		s.ready = initializeDisplay()
	}
	return s.ready
}

// PixelFormat is the native opaque framebuffer format accepted by the LCD.
func (*Screen) PixelFormat() graphics.PixelFormat { return graphics.PixelRGB565 }

// PresentSurface2x sends the dirty part of a native RGB565 surface at 2x scale.
func (s *Screen) PresentSurface2x(surface *graphics.Surface) bool {
	return s.Initialize() && presentSurface2x(surface)
}

func coreWrite(address uint16, register, value byte) bool {
	return coreBus.Write(address, []byte{register, value}) == nil
}
func coreUpdate(address uint16, register, mask, bits byte) bool {
	var data [1]byte
	if coreBus.Tx(address, []byte{register}, data[:]) != nil {
		return false
	}
	return coreWrite(address, register, data[0]&^mask|bits)
}

// Only configure display/touch reset pins and the LCD supply. Preserve the
// firmware's unrelated audio, charger, and USB/bus power-routing settings.
func initializePower() bool {
	var id [1]byte
	if coreBus.Tx(0x58, []byte{0x10}, id[:]) != nil || id[0] != 0x23 {
		return false
	}
	if !coreUpdate(0x34, 0x90, 0x80, 0) {
		return false
	}
	if !coreUpdate(0x34, 0x80, 1, 1) {
		return false
	} // DCDC1 LCD supply
	if !coreUpdate(0x58, 0x11, 0x10, 0x10) {
		return false
	} // P0 push-pull
	if !coreUpdate(0x58, 0x12, 1, 1) || !coreUpdate(0x58, 0x13, 2, 2) {
		return false
	}
	if !coreUpdate(0x58, 2, 1, 1) || !coreUpdate(0x58, 3, 2, 2) {
		return false
	}
	if !coreUpdate(0x58, 4, 1, 0) || !coreUpdate(0x58, 5, 6, 4) {
		return false
	}
	if !coreUpdate(0x58, 2, 1, 0) {
		return false
	}
	Clock.DelayMilliseconds(10)
	if !coreUpdate(0x58, 2, 1, 1) {
		return false
	}
	Clock.DelayMilliseconds(200)
	return coreWrite(0x38, 0xa4, 0) // FT6336U polling mode
}

// TouchPoint uses native landscape display coordinates.
type TouchPoint struct{ X, Y, ID int }
type TouchController struct{}

var Touch = TouchController{}

// Read returns up to two contacts. A bus error is distinct from finger release.
func (*TouchController) Read(points []TouchPoint) (int, bool) {
	var report [13]byte
	if coreBus.Tx(0x38, []byte{2}, report[:]) != nil {
		return 0, false
	}
	return decodeCoreTouch(report[:], points)
}
func decodeCoreTouch(report []byte, points []TouchPoint) (int, bool) {
	if len(report) < 13 || report[0]&15 > 2 {
		return 0, false
	}
	count := int(report[0] & 15)
	n := 0
	for i := 0; i < count; i++ {
		at := 1 + i*6
		if report[at]>>6 == 1 {
			continue
		}
		x := int(report[at]&15)<<8 | int(report[at+1])
		y := int(report[at+2]&15)<<8 | int(report[at+3])
		if x >= displayWidth || y >= displayHeight {
			continue
		}
		if n < len(points) {
			points[n] = TouchPoint{X: x, Y: y, ID: int(report[at+2] >> 4)}
			n++
		}
	}
	return n, true
}
