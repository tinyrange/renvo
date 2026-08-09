//go:build renvo && android

package forms

import "renvo.dev/std/graphics"

// Form is the retained root of the compact Android profile. The initial port
// owns background invalidation and a mobile title/body/action presentation;
// controls and native input can be layered onto the same surface contract.
type Form struct {
	width      int
	height     int
	background graphics.Color
	invalid    bool
}

func (f *Form) Initialize(width, height int) {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	f.width = width
	f.height = height
	f.background = graphics.RGBA(27, 31, 38, 255)
	f.invalid = true
}

func (f *Form) SetBackground(color graphics.Color) {
	if f != nil {
		f.background = color
		f.invalid = true
	}
}

func (f *Form) Paint(surface *graphics.Surface) bool {
	if f == nil || surface == nil || !f.invalid {
		return false
	}
	f.invalid = false
	dirty := graphics.R(0, 0, graphics.Scalar(f.width), graphics.Scalar(f.height))
	surface.BeginDamage(dirty)
	surface.FillRect(dirty, f.background)
	surface.FillRect(graphics.R(12, 22, 156, 152), graphics.RGBA(35, 40, 49, 255))
	surface.FillRect(graphics.R(20, 38, 88, 8), graphics.RGBA(232, 235, 240, 255))
	surface.FillRect(graphics.R(20, 66, 116, 5), graphics.RGBA(164, 172, 184, 255))
	surface.FillRect(graphics.R(20, 78, 92, 5), graphics.RGBA(164, 172, 184, 255))
	surface.FillRect(graphics.R(20, 112, 140, 36), graphics.RGBA(65, 145, 230, 255))
	surface.FillRect(graphics.R(54, 126, 72, 6), graphics.RGBA(255, 255, 255, 255))
	surface.EndDamage()
	return true
}
