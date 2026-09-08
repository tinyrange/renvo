package main

import (
	"renvo.dev/device/board"
	"renvo.dev/forms"
	"renvo.dev/std/graphics"
	"renvo.dev/std/strconv"
)

type demo struct {
	form     forms.Form
	status   *forms.Label
	slider   *forms.Slider
	progress *forms.ProgressBar
	count    int
	dark     bool
}

func (d *demo) clicked() {
	d.count++
	d.status.SetText("Taps: " + strconv.Itoa(d.count))
}
func (d *demo) theme() {
	d.dark = !d.dark
	if d.dark {
		d.form.ApplyTheme(forms.DarkTheme())
	} else {
		d.form.ApplyTheme(forms.LightTheme())
	}
}
func (d *demo) changed() { d.progress.SetValue(d.slider.Value()) }
func main() {
	println("CoreS3-SE: initializing display and touch")
	if !board.Display.Initialize() {
		println("CoreS3-SE: initialization failed")
		return
	}
	var d demo
	d.form.Initialize(160, 120)
	d.form.ApplyTheme(forms.DarkTheme())
	d.dark = true
	font := graphics.NewBuiltinFont(1)
	title := forms.NewLabel()
	title.SetBounds(graphics.R(6, 4, 148, 12))
	title.SetFont(font)
	title.SetText("RENVO / CoreS3 SE")
	d.form.Add(&title.Control)
	button := forms.NewButton()
	button.SetBounds(graphics.R(6, 22, 70, 24))
	button.SetFont(font)
	button.SetText("Tap me")
	button.Click = d.clicked
	d.form.Add(&button.Control)
	theme := forms.NewButton()
	theme.SetBounds(graphics.R(84, 22, 70, 24))
	theme.SetFont(font)
	theme.SetText("Theme")
	theme.Click = d.theme
	d.form.Add(&theme.Control)
	d.slider = forms.NewSlider()
	d.slider.SetBounds(graphics.R(6, 52, 148, 24))
	d.slider.SetRange(0, 100)
	d.slider.SetValue(40)
	d.form.Add(&d.slider.Control)
	d.progress = forms.NewProgressBar()
	d.progress.SetBounds(graphics.R(6, 82, 148, 12))
	d.progress.SetValue(40)
	d.form.Add(&d.progress.Control)
	d.slider.Changed = d.changed
	d.status = forms.NewLabel()
	d.status.SetBounds(graphics.R(6, 102, 148, 12))
	d.status.SetFont(font)
	d.status.SetText("Touch + drag to play")
	d.form.Add(&d.status.Control)
	surface := graphics.NewSurfaceFormat(160, 120, board.Display.PixelFormat())
	var points [2]board.TouchPoint
	pressed := false
	x, y := 0, 0
	ready := false
	for {
		count, ok := board.Touch.Read(points[:])
		if ok {
			if count > 0 {
				nx, ny := points[0].X/2, points[0].Y/2
				if !pressed {
					d.form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: graphics.Scalar(nx), Y: graphics.Scalar(ny), Button: 1})
				} else if nx != x || ny != y {
					d.form.Dispatch(graphics.Event{Type: graphics.EventPointerMove, X: graphics.Scalar(nx), Y: graphics.Scalar(ny), Button: 1})
				}
				pressed = true
				x = nx
				y = ny
			} else if pressed {
				d.form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: graphics.Scalar(x), Y: graphics.Scalar(y), Button: 1})
				pressed = false
			}
		}
		paintStart := board.Clock.Ticks()
		d.form.Paint(surface)
		paintTicks := board.Clock.Ticks() - paintStart
		transferStart := board.Clock.Ticks()
		if !board.Display.PresentSurface2x(surface) {
			println("CoreS3-SE: display transfer failed")
			return
		}
		transferTicks := board.Clock.Ticks() - transferStart
		if !ready {
			println("first frame us paint/transfer", paintTicks/16, transferTicks/16)
		}
		surface.ResetDirty()
		if !ready {
			println("CoreS3-SE: forms ready")
			ready = true
		}
		board.Clock.DelayMilliseconds(10)
	}
}
