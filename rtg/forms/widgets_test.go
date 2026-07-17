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

func TestTextControlsEditThroughPortableInputEvents(t *testing.T) {
	var form Form
	form.Initialize(240, 120)
	box := NewTextBox()
	box.SetBounds(graphics.R(10, 10, 180, 30))
	box.SetFont(graphics.NewBuiltinFont(1))
	area := NewTextArea()
	area.SetBounds(graphics.R(10, 50, 180, 60))
	area.SetFont(graphics.NewBuiltinFont(1))
	form.Add(&box.Control)
	form.Add(&area.Control)

	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 20, Y: 20})
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "hé"})
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyBackspace})
	if box.Text() != "h" {
		t.Fatalf("text box value = %q", box.Text())
	}
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 20, Y: 60})
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "one"})
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyEnter})
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "two"})
	if area.Text() != "one\ntwo" {
		t.Fatalf("text area value = %q", area.Text())
	}
}

func TestChoiceAndLayoutControlsPaintAndToggle(t *testing.T) {
	var form Form
	form.Initialize(280, 180)
	font := graphics.NewBuiltinFont(1)
	check := NewCheckBox()
	check.SetBounds(graphics.R(10, 10, 140, 28))
	check.SetFont(font)
	check.SetText("Enabled")
	radio := NewRadioButton()
	radio.SetBounds(graphics.R(10, 45, 140, 28))
	radio.SetFont(font)
	radio.SetText("Choice")
	picture := NewPictureBox()
	picture.SetBounds(graphics.R(160, 10, 100, 70))
	panel := NewPanel()
	panel.SetBounds(graphics.R(10, 90, 250, 70))
	form.Add(&panel.Control)
	form.Add(&picture.Control)
	form.Add(&check.Control)
	form.Add(&radio.Control)
	if !form.Paint(graphics.NewSurface(280, 180)) {
		t.Fatal("new controls produced no paint")
	}

	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 20, Y: 20})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: 20, Y: 20})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 20, Y: 55})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: 20, Y: 55})
	if !check.Checked() || !radio.Checked() {
		t.Fatalf("choice state = checkbox %v radio %v", check.Checked(), radio.Checked())
	}
}
