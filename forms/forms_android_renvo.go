//go:build renvo && android && compact_forms

package forms

import "renvo.dev/std/graphics"

// Form is the retained root of the compact Android profile. The initial port
// owns background invalidation and a mobile title/body/action presentation;
// controls and native input can be layered onto the same surface contract.
type Form struct {
	width         int
	height        int
	background    graphics.Color
	font          *graphics.Font
	title         string
	body          string
	action        string
	ActionClick   func()
	actionPressed bool
	invalid       bool
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

func (f *Form) SetFont(font *graphics.Font) {
	if f != nil {
		f.font = font
		f.invalid = true
	}
}

func (f *Form) SetContent(title, body, action string) {
	if f != nil {
		f.title = title
		f.body = body
		f.action = action
		f.invalid = true
	}
}

func (f *Form) actionBounds() graphics.Rect {
	return graphics.R(20, 112, graphics.Scalar(f.width-40), 36)
}

func (f *Form) pointInAction(x, y graphics.Scalar) bool {
	bounds := f.actionBounds()
	return x >= bounds.MinX && x < bounds.MaxX &&
		y >= bounds.MinY && y < bounds.MaxY
}

func (f *Form) Dispatch(event *graphics.Event) {
	if f == nil || event == nil {
		return
	}
	if event.Type == graphics.EventWindowResize {
		f.width = int(event.Dirty.Width())
		f.height = int(event.Dirty.Height())
		f.invalid = true
		return
	}
	if event.Type == graphics.EventWindowExpose {
		f.invalid = true
		return
	}
	if event.Type == graphics.EventPointerDown {
		f.actionPressed = f.pointInAction(event.X, event.Y)
		f.invalid = true
		return
	}
	if event.Type == graphics.EventPointerUp {
		clicked := f.actionPressed && f.pointInAction(event.X, event.Y)
		f.actionPressed = false
		f.invalid = true
		if clicked && f.ActionClick != nil {
			f.ActionClick()
		}
		return
	}
	if event.Type == graphics.EventPointerCancel {
		f.actionPressed = false
		f.invalid = true
	}
}

func (f *Form) SetBackground(color graphics.Color) {
	if f != nil {
		f.background = color
		f.invalid = true
	}
}

func (f *Form) Paint(surface graphics.Canvas) bool {
	if f == nil || surface == nil || !f.invalid {
		return false
	}
	f.invalid = false
	dirty := graphics.R(0, 0, graphics.Scalar(f.width), graphics.Scalar(f.height))
	surface.BeginDamage(dirty)
	surface.FillRect(dirty, f.background)
	surface.FillRect(graphics.R(12, 22, graphics.Scalar(f.width-24), 152), graphics.RGBA(35, 40, 49, 255))
	actionColor := graphics.RGBA(65, 145, 230, 255)
	if f.actionPressed {
		actionColor = graphics.RGBA(38, 111, 190, 255)
	}
	surface.FillRect(f.actionBounds(), actionColor)
	if f.font != nil {
		surface.DrawText(f.font, graphics.Point{X: 20, Y: 51}, f.title, graphics.RGBA(232, 235, 240, 255))
		surface.DrawText(f.font, graphics.Point{X: 20, Y: 82}, f.body, graphics.RGBA(164, 172, 184, 255))
		metrics := graphics.MeasureText(f.font, f.action)
		x := graphics.Scalar(20) + (graphics.Scalar(f.width-40)-metrics.Width)/2
		surface.DrawText(f.font, graphics.Point{X: x, Y: 136}, f.action, graphics.White)
	}
	surface.EndDamage()
	return true
}
