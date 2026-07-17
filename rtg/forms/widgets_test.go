package forms

import (
	"testing"

	"j5.nz/rtg/rtg/std/graphics"
)

func TestButtonPropertiesPaintAndDispatchClick(t *testing.T) {
	var form Form
	form.Initialize(180, 80)
	button := NewButton()
	button.SetBounds(graphics.R(10, 10, 120, 36))
	button.SetFont(graphics.NewBuiltinFont(1))
	button.SetText("Say Hello")
	clicked := 0
	button.Click = func() { clicked++ }
	form.Add(&button.Control)

	surface := graphics.NewSurface(180, 80)
	if !form.Paint(surface) {
		t.Fatal("initial form did not paint")
	}
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 20, Y: 20, Button: 1})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: 20, Y: 20, Button: 1})
	if clicked != 1 {
		t.Fatalf("button click count = %d, want 1", clicked)
	}
	if len(form.InvalidRects()) == 0 {
		t.Fatal("button press did not invalidate its retained bounds")
	}
}

func TestLabelTextChangeInvalidatesOwningForm(t *testing.T) {
	var form Form
	form.Initialize(180, 80)
	label := NewLabel()
	label.SetBounds(graphics.R(10, 10, 140, 24))
	label.SetFont(graphics.NewBuiltinFont(1))
	form.Add(&label.Control)
	form.Paint(graphics.NewSurface(180, 80))

	label.SetText("Hello, World!")
	invalid := form.InvalidRects()
	if len(invalid) != 1 || invalid[0] != label.Bounds() {
		t.Fatalf("label invalidation = %#v, want %#v", invalid, label.Bounds())
	}
}
