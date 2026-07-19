package forms

import (
	"testing"

	"renvo.dev/std/graphics"
)

func TestPropertySettersInvalidateOnlyControlBounds(t *testing.T) {
	var form Form
	form.Initialize(100, 80)
	surface := graphics.NewSurface(100, 80)
	form.Paint(surface)
	surface.ResetDirty()

	control := NewControl()
	control.SetBounds(graphics.R(10, 12, 20, 16))
	form.Add(control)
	form.Paint(surface)
	surface.ResetDirty()
	control.SetText("changed")

	assertRects(t, form.InvalidRects(), []graphics.Rect{graphics.R(10, 12, 20, 16)})
	form.Paint(surface)
	dirty, ok := surface.DirtyRect()
	if !ok || dirty != graphics.R(10, 12, 20, 16) {
		t.Fatalf("surface dirty = %#v, %v", dirty, ok)
	}
}

func TestMovingControlKeepsOldAndNewDamageSeparate(t *testing.T) {
	var form Form
	form.Initialize(200, 100)
	control := NewControl()
	control.SetBounds(graphics.R(10, 10, 20, 20))
	form.Add(control)
	form.invalid = nil

	control.SetBounds(graphics.R(150, 60, 20, 20))
	assertRects(t, form.InvalidRects(), []graphics.Rect{
		graphics.R(10, 10, 20, 20),
		graphics.R(150, 60, 20, 20),
	})
	surface := graphics.NewSurface(200, 100)
	surface.ResetDirty()
	form.Paint(surface)
	assertRects(t, surface.DirtyRects(), []graphics.Rect{
		graphics.R(10, 10, 20, 20),
		graphics.R(150, 60, 20, 20),
	})
}

func TestPaintClipsControlsToInvalidRegions(t *testing.T) {
	var form Form
	form.Initialize(20, 10)
	form.SetBackground(graphics.Black)
	control := NewControl()
	control.SetBounds(graphics.R(0, 0, 20, 10))
	control.Paint = func(surface *graphics.Surface) {
		surface.FillRect(control.Bounds(), graphics.White)
	}
	form.Add(control)
	surface := graphics.NewSurface(20, 10)
	form.Paint(surface)
	surface.ResetDirty()

	// Paint a known marker outside the next invalid region.
	surface.FillRect(graphics.R(15, 5, 1, 1), graphics.RGBA(255, 0, 0, 255))
	surface.ResetDirty()
	form.Invalidate(graphics.R(2, 2, 3, 2))
	form.Paint(surface)
	if pixel := surfacePixel(surface, 15, 5); pixel != (graphics.Color{R: 255, A: 255}) {
		t.Fatalf("pixel outside invalid region was redrawn: %#v", pixel)
	}
	dirty, ok := surface.DirtyRect()
	if !ok || dirty != graphics.R(2, 2, 3, 2) {
		t.Fatalf("clipped dirty = %#v, %v", dirty, ok)
	}
}

func TestFormPaintBackgroundOverridesDefaultDrawing(t *testing.T) {
	var form Form
	form.Initialize(80, 50)
	painted := 0
	form.PaintBackground = func(surface *graphics.Surface) {
		painted++
		surface.FillRect(graphics.R(0, 0, 80, 50), graphics.RGBA(12, 34, 56, 255))
	}
	if !form.Paint(graphics.NewSurface(80, 50)) || painted != 1 {
		t.Fatalf("custom background paint count = %d", painted)
	}
}

func TestGeneratedStyleEventWiringDispatchesToFocusedControl(t *testing.T) {
	var form Form
	form.Initialize(80, 40)
	control := NewControl()
	control.SetBounds(graphics.R(5, 5, 30, 20))
	clicks := 0
	text := ""
	control.Click = func() { clicks++ }
	control.TextInput = func(value string) { text += value }
	form.Add(control)

	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 10, Y: 10})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: 10, Y: 10})
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "λ"})
	if clicks != 1 || text != "λ" || !control.Focused() {
		t.Fatalf("event state = clicks %d, text %q, focused %v", clicks, text, control.Focused())
	}
}

func TestPressedControlKeepsPointerCaptureUntilRelease(t *testing.T) {
	var form Form
	form.Initialize(100, 50)
	control := NewControl()
	control.SetBounds(graphics.R(5, 5, 20, 20))
	moves := 0
	releases := 0
	clicks := 0
	control.PointerMove = func(x, y graphics.Scalar) { moves++ }
	control.PointerUp = func(x, y graphics.Scalar) { releases++ }
	control.Click = func() { clicks++ }
	form.Add(control)

	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 10, Y: 10})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerMove, X: 90, Y: 40})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: 90, Y: 40})
	if moves != 1 || releases != 1 || clicks != 0 {
		t.Fatalf("captured events = moves %d releases %d clicks %d", moves, releases, clicks)
	}
}

func TestShortcutTextDoesNotLeakIntoFocusedControl(t *testing.T) {
	var form Form
	form.Initialize(80, 40)
	control := NewControl()
	control.SetBounds(graphics.R(0, 0, 40, 20))
	text := ""
	control.TextInput = func(value string) { text += value }
	form.Add(control)
	control.Focus()
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "s", Modifiers: graphics.ModifierCommand})
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "x", Modifiers: graphics.ModifierControl})
	form.Dispatch(graphics.Event{Type: graphics.EventTextInput, Text: "λ", Modifiers: graphics.ModifierControl | graphics.ModifierAlt})
	if text != "λ" {
		t.Fatalf("text after shortcuts and AltGr input = %q", text)
	}
}

func assertRects(t *testing.T, got, want []graphics.Rect) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("rects = %#v, want %#v", got, want)
	}
	for i := 0; i < len(got); i++ {
		if got[i] != want[i] {
			t.Fatalf("rects = %#v, want %#v", got, want)
		}
	}
}

func surfacePixel(surface *graphics.Surface, x, y int) graphics.Color {
	offset := y*surface.Stride + x*4
	return graphics.Color{
		R: surface.Pixels[offset],
		G: surface.Pixels[offset+1],
		B: surface.Pixels[offset+2],
		A: surface.Pixels[offset+3],
	}
}
