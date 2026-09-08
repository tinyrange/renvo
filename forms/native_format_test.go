package forms

import (
	"renvo.dev/std/graphics"
	"testing"
)

func TestFormPaintsAndUpdatesNativeRGB565(t *testing.T) {
	var form Form
	form.Initialize(160, 120)
	form.ApplyTheme(DarkTheme())
	button := NewButton()
	button.SetBounds(graphics.R(6, 22, 70, 24))
	button.SetFont(graphics.NewBuiltinFont(1))
	button.SetText("Tap me")
	clicks := 0
	button.Click = func() { clicks++; button.SetText("Clicked") }
	form.Add(&button.Control)
	surface := graphics.NewSurfaceFormat(160, 120, graphics.PixelRGB565)
	first := &surface.Pixels[0]
	if !form.Paint(surface) {
		t.Fatal("initial native paint failed")
	}
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 20, Y: 32, Button: 1})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: 20, Y: 32, Button: 1})
	surface.ResetDirty()
	if clicks != 1 || !form.Paint(surface) {
		t.Fatal("button did not update native surface")
	}
	if surface.Format != graphics.PixelRGB565 || len(surface.Pixels) != 38400 || &surface.Pixels[0] != first {
		t.Fatal("Forms replaced native framebuffer storage or format")
	}
	dirty, ok := surface.DirtyRect()
	if !ok || dirty != button.Bounds() {
		t.Fatalf("button damage = %v, %v", dirty, ok)
	}
}
