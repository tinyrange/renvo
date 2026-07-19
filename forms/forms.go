// Package forms provides retained controls and Windows Forms-style event
// wiring on top of RENVO's portable graphics package.
package forms

import "renvo.dev/std/graphics"

const maxInvalidRects = 64

type PaintHandler func(surface *graphics.Surface)
type EventHandler func()
type PointerHandler func(x, y graphics.Scalar)
type WheelHandler func(x, y graphics.Scalar)
type TextInputHandler func(text string)
type KeyHandler func(event graphics.Event)

// Control is the retained base of every widget. Visual properties are changed
// through setters so the owning form can invalidate the precise old and new
// areas affected by a change.
type Control struct {
	form       *Form
	bounds     graphics.Rect
	visible    bool
	enabled    bool
	tabStop    bool
	background graphics.Color
	foreground graphics.Color
	text       string

	Paint        PaintHandler
	Click        EventHandler
	PointerDown  PointerHandler
	PointerUp    PointerHandler
	PointerMove  PointerHandler
	PointerWheel WheelHandler
	TextInput    TextInputHandler
	KeyDown      KeyHandler
	KeyUp        KeyHandler
}

func NewControl() *Control {
	return &Control{
		visible:    true,
		enabled:    true,
		tabStop:    true,
		background: graphics.White,
		foreground: graphics.Black,
	}
}

func (c *Control) Bounds() graphics.Rect { return c.bounds }
func (c *Control) Visible() bool         { return c.visible }
func (c *Control) Enabled() bool         { return c.enabled }
func (c *Control) TabStop() bool         { return c.tabStop }
func (c *Control) Background() graphics.Color {
	return c.background
}
func (c *Control) Foreground() graphics.Color { return c.foreground }
func (c *Control) Text() string               { return c.text }
func (c *Control) Form() *Form                { return c.form }
func (c *Control) Focused() bool              { return c.form != nil && c.form.focused == c }

func (c *Control) SetBounds(bounds graphics.Rect) {
	if c == nil || rectEqual(c.bounds, bounds) {
		return
	}
	old := c.bounds
	c.bounds = bounds
	if c.form != nil && c.visible {
		c.form.Invalidate(old)
		c.form.Invalidate(bounds)
	}
}

func (c *Control) SetVisible(visible bool) {
	if c == nil || c.visible == visible {
		return
	}
	if c.form != nil && c.visible {
		c.form.Invalidate(c.bounds)
	}
	c.visible = visible
	if !visible && c.form != nil && c.form.focused == c {
		c.form.focused = nil
	}
	if c.form != nil && c.visible {
		c.form.Invalidate(c.bounds)
	}
}

func (c *Control) SetEnabled(enabled bool) {
	if c == nil || c.enabled == enabled {
		return
	}
	c.enabled = enabled
	c.Invalidate()
}

func (c *Control) SetTabStop(tabStop bool) {
	if c != nil {
		c.tabStop = tabStop
	}
}

func (c *Control) SetBackground(color graphics.Color) {
	if c == nil || c.background == color {
		return
	}
	c.background = color
	c.Invalidate()
}

func (c *Control) SetForeground(color graphics.Color) {
	if c == nil || c.foreground == color {
		return
	}
	c.foreground = color
	c.Invalidate()
}

func (c *Control) SetText(text string) {
	if c == nil || c.text == text {
		return
	}
	c.text = text
	c.Invalidate()
}

func (c *Control) Focus() {
	if c != nil && c.form != nil && c.visible && c.enabled && c.tabStop {
		c.form.setFocused(c)
	}
}

func (c *Control) Invalidate() {
	if c != nil && c.form != nil && c.visible {
		c.form.Invalidate(c.bounds)
	}
}

// Form is embedded in each application-defined form struct. Generated code
// creates controls, assigns properties, wires callbacks, and adds controls;
// user code implements those callbacks on the containing form struct.
type Form struct {
	width      int
	height     int
	background graphics.Color
	controls   []*Control
	invalid    []graphics.Rect
	focused    *Control
	pressed    *Control

	Resize          EventHandler
	PaintBackground PaintHandler
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
	f.background = graphics.RGBA(248, 249, 251, 255)
	f.controls = nil
	f.invalid = nil
	f.focused = nil
	f.pressed = nil
	f.Invalidate(f.clientRect())
}

func (f *Form) Size() (int, int) { return f.width, f.height }
func (f *Form) Background() graphics.Color {
	return f.background
}

func (f *Form) SetBackground(color graphics.Color) {
	if f == nil || f.background == color {
		return
	}
	f.background = color
	f.Invalidate(f.clientRect())
}

func (f *Form) SetClientSize(width, height int) {
	if f == nil || width < 0 || height < 0 || f.width == width && f.height == height {
		return
	}
	old := f.clientRect()
	f.width = width
	f.height = height
	f.Invalidate(old)
	f.Invalidate(f.clientRect())
}

func (f *Form) Add(control *Control) {
	if f == nil || control == nil || control.form == f {
		return
	}
	if control.form != nil {
		control.form.Remove(control)
	}
	control.form = f
	f.controls = append(f.controls, control)
	if control.visible {
		f.Invalidate(control.bounds)
	}
}

func (f *Form) Remove(control *Control) {
	if f == nil || control == nil || control.form != f {
		return
	}
	for i := 0; i < len(f.controls); i++ {
		if f.controls[i] != control {
			continue
		}
		if control.visible {
			f.Invalidate(control.bounds)
		}
		copy(f.controls[i:], f.controls[i+1:])
		f.controls = f.controls[:len(f.controls)-1]
		control.form = nil
		if f.focused == control {
			f.focused = nil
		}
		if f.pressed == control {
			f.pressed = nil
		}
		return
	}
}

func (f *Form) Controls() []*Control {
	if f == nil {
		return nil
	}
	controls := make([]*Control, len(f.controls))
	copy(controls, f.controls)
	return controls
}

// Invalidate queues an exact client-space rectangle. Contained rectangles are
// discarded, but disjoint or partially overlapping rectangles stay separate
// to avoid repainting the empty bounding-box area between controls.
func (f *Form) Invalidate(rect graphics.Rect) {
	if f == nil {
		return
	}
	rect = intersectRect(rect, f.clientRect())
	if rect.Empty() {
		return
	}
	for i := 0; i < len(f.invalid); i++ {
		if rectContains(f.invalid[i], rect) {
			return
		}
		if rectContains(rect, f.invalid[i]) {
			copy(f.invalid[i:], f.invalid[i+1:])
			f.invalid = f.invalid[:len(f.invalid)-1]
			i--
		}
	}
	f.invalid = append(f.invalid, rect)
	if len(f.invalid) > maxInvalidRects {
		all := f.invalid[0]
		for i := 1; i < len(f.invalid); i++ {
			all = unionRect(all, f.invalid[i])
		}
		f.invalid = f.invalid[:1]
		f.invalid[0] = all
	}
}

func (f *Form) InvalidRects() []graphics.Rect {
	if f == nil {
		return nil
	}
	out := make([]graphics.Rect, len(f.invalid))
	copy(out, f.invalid)
	return out
}

// Paint redraws only invalidated regions. Each region is clipped independently
// and all intersecting controls are painted in z-order so moved or overlapping
// controls leave correct pixels behind.
func (f *Form) Paint(surface *graphics.Surface) bool {
	if f == nil || surface == nil || len(f.invalid) == 0 {
		return false
	}
	invalid := f.invalid
	f.invalid = nil
	for i := 0; i < len(invalid); i++ {
		dirty := invalid[i]
		surface.BeginDamage(dirty)
		surface.PushClipRect(dirty)
		if f.PaintBackground != nil {
			f.PaintBackground(surface)
		} else {
			surface.FillRect(dirty, f.background)
		}
		for j := 0; j < len(f.controls); j++ {
			control := f.controls[j]
			if control.visible && control.Paint != nil && rectIntersects(control.bounds, dirty) {
				control.Paint(surface)
			}
		}
		surface.PopClip()
		surface.EndDamage()
	}
	return true
}

// Dispatch routes portable window events to retained controls. Generated form
// code wires the handlers to methods on the application-defined form struct.
func (f *Form) Dispatch(event graphics.Event) {
	if f == nil {
		return
	}
	if event.Type == graphics.EventWindowResize {
		f.SetClientSize(int(event.Dirty.Width()), int(event.Dirty.Height()))
		if f.Resize != nil {
			f.Resize()
		}
		return
	}
	if event.Type == graphics.EventWindowExpose {
		f.Invalidate(event.Dirty)
		return
	}
	if event.Type == graphics.EventPointerDown {
		control := f.hitTest(event.X, event.Y)
		f.pressed = control
		if control != nil {
			control.Focus()
			if control.PointerDown != nil {
				control.PointerDown(event.X-control.bounds.MinX, event.Y-control.bounds.MinY)
			}
		}
		return
	}
	if event.Type == graphics.EventPointerUp {
		hit := f.hitTest(event.X, event.Y)
		control := f.pressed
		if control == nil {
			control = hit
		}
		if control != nil && control.PointerUp != nil {
			control.PointerUp(event.X-control.bounds.MinX, event.Y-control.bounds.MinY)
		}
		if control != nil && control == hit && control == f.pressed && control.Click != nil {
			control.Click()
		}
		f.pressed = nil
		return
	}
	if event.Type == graphics.EventPointerMove {
		control := f.pressed
		if control == nil {
			control = f.hitTest(event.X, event.Y)
		}
		if control != nil && control.PointerMove != nil {
			control.PointerMove(event.X-control.bounds.MinX, event.Y-control.bounds.MinY)
		}
		return
	}
	if event.Type == graphics.EventPointerWheel {
		control := f.hitTest(event.X, event.Y)
		if control != nil && control.PointerWheel != nil {
			control.PointerWheel(event.WheelX, event.WheelY)
		}
		return
	}
	if f.focused == nil {
		return
	}
	if event.Type == graphics.EventTextInput && f.focused.TextInput != nil {
		shortcut := event.Modifiers&graphics.ModifierCommand != 0 || event.Modifiers&graphics.ModifierControl != 0 && event.Modifiers&graphics.ModifierAlt == 0
		if !shortcut {
			f.focused.TextInput(event.Text)
		}
	}
	if event.Type == graphics.EventKeyDown && f.focused.KeyDown != nil {
		f.focused.KeyDown(event)
	}
	if event.Type == graphics.EventKeyUp && f.focused.KeyUp != nil {
		f.focused.KeyUp(event)
	}
}

func (f *Form) setFocused(control *Control) {
	if f.focused == control {
		return
	}
	old := f.focused
	f.focused = control
	if old != nil {
		old.Invalidate()
	}
	if control != nil {
		control.Invalidate()
	}
}

func (f *Form) hitTest(x, y graphics.Scalar) *Control {
	for i := len(f.controls) - 1; i >= 0; i-- {
		control := f.controls[i]
		if control.visible && control.enabled && pointInRect(x, y, control.bounds) {
			return control
		}
	}
	return nil
}

func (f *Form) clientRect() graphics.Rect {
	return graphics.R(0, 0, graphics.Scalar(f.width), graphics.Scalar(f.height))
}

func rectEqual(a, b graphics.Rect) bool {
	return a.MinX == b.MinX && a.MinY == b.MinY && a.MaxX == b.MaxX && a.MaxY == b.MaxY
}

func rectContains(outer, inner graphics.Rect) bool {
	return outer.MinX <= inner.MinX && outer.MinY <= inner.MinY && outer.MaxX >= inner.MaxX && outer.MaxY >= inner.MaxY
}

func rectIntersects(a, b graphics.Rect) bool {
	return a.MinX < b.MaxX && b.MinX < a.MaxX && a.MinY < b.MaxY && b.MinY < a.MaxY
}

func pointInRect(x, y graphics.Scalar, rect graphics.Rect) bool {
	return x >= rect.MinX && x < rect.MaxX && y >= rect.MinY && y < rect.MaxY
}

func intersectRect(a, b graphics.Rect) graphics.Rect {
	if b.MinX > a.MinX {
		a.MinX = b.MinX
	}
	if b.MinY > a.MinY {
		a.MinY = b.MinY
	}
	if b.MaxX < a.MaxX {
		a.MaxX = b.MaxX
	}
	if b.MaxY < a.MaxY {
		a.MaxY = b.MaxY
	}
	if a.MaxX < a.MinX {
		a.MaxX = a.MinX
	}
	if a.MaxY < a.MinY {
		a.MaxY = a.MinY
	}
	return a
}

func unionRect(a, b graphics.Rect) graphics.Rect {
	if b.MinX < a.MinX {
		a.MinX = b.MinX
	}
	if b.MinY < a.MinY {
		a.MinY = b.MinY
	}
	if b.MaxX > a.MaxX {
		a.MaxX = b.MaxX
	}
	if b.MaxY > a.MaxY {
		a.MaxY = b.MaxY
	}
	return a
}
