package frontend_tests

import (
	. "renvo.dev/forms"
	"renvo.dev/std/graphics"
	"testing"
)

func TestRotateSurfaceRelayoutAndRetainedDamage(t *testing.T) {
	var form Form
	form.Initialize(40, 60)
	control := NewControl()
	control.SetDock(DockFill)
	control.Paint = func(c graphics.Canvas) { c.FillRect(control.Bounds(), graphics.White) }
	form.Add(control)
	s := graphics.NewRotatedSurface(40, 60, graphics.PixelRGB565, graphics.Rotation0)
	pixels := &s.Pixels[0]
	for _, rotation := range []graphics.Rotation{graphics.Rotation90, graphics.Rotation0, graphics.Rotation270, graphics.Rotation180} {
		if !form.RotateSurface(s, rotation) || !form.Paint(s) {
			t.Fatal("rotation did not repaint")
		}
		w, h := form.Size()
		if w != s.Width || h != s.Height || control.Bounds() != graphics.R(0, 0, graphics.Scalar(w), graphics.Scalar(h)) {
			t.Fatal("layout not resized")
		}
		if &s.Pixels[0] != pixels {
			t.Fatal("buffer replaced")
		}
		s.ResetDirty()
		r := graphics.R(2, 3, 5, 7)
		form.Invalidate(r)
		if !form.Paint(s) {
			t.Fatal("retained paint missing")
		}
		dirty, ok := s.DirtyRectAt(0)
		if !ok || dirty != r || s.DirtyRectCount() != 1 {
			t.Fatal("logical damage lost", dirty)
		}
	}
}
