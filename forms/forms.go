// Package forms provides retained controls and Windows Forms-style event
// wiring on top of RENVO's portable graphics package.
package forms

import "renvo.dev/std/graphics"

const maxInvalidRects = 64

// DockStyle controls how a control consumes space from its form. Edge-docked
// controls are laid out in control order; fill controls receive the remaining
// client rectangle after every visible edge-docked control.
type DockStyle int

const (
	DockNone DockStyle = iota
	DockTop
	DockBottom
	DockLeft
	DockRight
	DockFill
)

// ThemeRole selects the standard surface treatment used by a base Control.
// Specialized widgets may still provide their own theme handler.
type ThemeRole int

const (
	ThemeSurface ThemeRole = iota
	ThemeField
	ThemeRaised
	ThemeTransparent
)

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
	form                     *Form
	bounds                   graphics.Rect
	dock                     DockStyle
	dockWidth                graphics.Scalar
	dockHeight               graphics.Scalar
	layoutBounds             func(bounds graphics.Rect)
	visible                  bool
	enabled                  bool
	tabStop                  bool
	acceptsTab               bool
	background               graphics.Color
	foreground               graphics.Color
	cursor                   graphics.Cursor
	theme                    Theme
	themeRole                ThemeRole
	hasTheme                 bool
	text                     string
	accessibilityID          string
	accessibilityRole        AccessibilityRole
	accessibilityName        string
	accessibilityDescription string
	accessibilityMultiline   bool
	popup                    bool
	dismiss                  EventHandler
	applyTheme               func(theme Theme)

	Paint         PaintHandler
	Click         EventHandler
	PointerDown   PointerHandler
	PointerUp     PointerHandler
	PointerMove   PointerHandler
	PointerWheel  WheelHandler
	PointerEnter  EventHandler
	PointerLeave  EventHandler
	PointerCursor func(x, y graphics.Scalar) graphics.Cursor
	TextInput     TextInputHandler
	KeyDown       KeyHandler
	KeyUp         KeyHandler

	AccessibilityValue          func() string
	AccessibilityCheckable      bool
	AccessibilityChecked        func() bool
	AccessibilitySelectable     bool
	AccessibilitySelected       func() bool
	AccessibilityInvoke         EventHandler
	AccessibilitySetValue       TextInputHandler
	AccessibilityChildren       func() []AccessibilityNode
	AccessibilityPerform        func(id string, action AccessibilityAction, value string) bool
	AccessibilitySelectionStart func() int
	AccessibilitySelectionEnd   func() int
	AccessibilitySetSelection   func(start, end int)
}

func NewControl() *Control {
	control := &Control{
		visible: true,
		enabled: true,
		tabStop: true,
	}
	control.applyBaseTheme(LightTheme())
	return control
}

func (c *Control) applyBaseTheme(theme Theme) {
	if c.themeRole == ThemeField {
		applyFieldTheme(c, theme)
	} else if c.themeRole == ThemeRaised {
		applyRaisedTheme(c, theme)
	} else if c.themeRole == ThemeTransparent {
		applyTransparentTheme(c, theme)
	} else {
		applySurfaceTheme(c, theme)
	}
}

func (c *Control) Bounds() graphics.Rect { return c.bounds }
func (c *Control) Dock() DockStyle       { return c.dock }
func (c *Control) Visible() bool         { return c.visible }
func (c *Control) Enabled() bool         { return c.enabled }
func (c *Control) TabStop() bool         { return c.tabStop }
func (c *Control) AcceptsTab() bool      { return c.acceptsTab }
func (c *Control) Background() graphics.Color {
	return c.background
}
func (c *Control) Foreground() graphics.Color { return c.foreground }
func (c *Control) Cursor() graphics.Cursor    { return c.cursor }
func (c *Control) Theme() Theme               { return controlTheme(c) }
func (c *Control) ThemeRole() ThemeRole       { return c.themeRole }
func (c *Control) Text() string               { return c.text }
func (c *Control) Form() *Form                { return c.form }
func (c *Control) Focused() bool              { return c.form != nil && c.form.focused == c }
func (c *Control) Hovered() bool              { return c.form != nil && c.form.hovered == c }

func (c *Control) SetBounds(bounds graphics.Rect) {
	if c == nil {
		return
	}
	c.dockWidth = bounds.Width()
	c.dockHeight = bounds.Height()
	if c.form != nil && c.dock != DockNone {
		c.form.performLayout()
		return
	}
	c.setBoundsCore(bounds)
}

func (c *Control) setBoundsCore(bounds graphics.Rect) {
	if c == nil || rectEqual(c.bounds, bounds) {
		return
	}
	old := c.bounds
	c.bounds = bounds
	c.AccessibilityChanged()
	c.AccessibilityChildrenChanged()
	if c.form != nil && c.visible {
		c.form.Invalidate(old)
		c.form.Invalidate(bounds)
	}
}

func (c *Control) setLayoutBounds(bounds graphics.Rect) {
	if c.layoutBounds != nil {
		c.layoutBounds(bounds)
	} else {
		c.setBoundsCore(bounds)
	}
}

func (c *Control) setPreferredBounds(bounds graphics.Rect) {
	if c == nil {
		return
	}
	c.dockWidth = bounds.Width()
	c.dockHeight = bounds.Height()
}

// SetDock changes the control's automatic form layout. SetBounds continues to
// define the thickness of an edge-docked control; its other coordinates are
// supplied by the form.
func (c *Control) SetDock(dock DockStyle) {
	if c == nil || dock < DockNone || dock > DockFill || c.dock == dock {
		return
	}
	c.dock = dock
	if c.form != nil {
		c.form.performLayout()
	}
}

func (c *Control) SetVisible(visible bool) {
	if c == nil || c.visible == visible {
		return
	}
	if !visible && c.popup && c.dismiss != nil {
		c.dismiss()
	}
	if c.form != nil && c.visible {
		c.form.Invalidate(c.bounds)
	}
	c.visible = visible
	c.AccessibilityChanged()
	c.AccessibilityChildrenChanged()
	if !visible && c.form != nil && c.form.focused == c {
		c.form.focused = nil
	}
	if !visible && c.form != nil && c.form.hovered == c {
		c.form.setHovered(nil)
	}
	if c.form != nil && c.visible {
		c.form.Invalidate(c.bounds)
	}
	if c.form != nil && c.dock != DockNone {
		c.form.performLayout()
	}
}

func (c *Control) SetEnabled(enabled bool) {
	if c == nil || c.enabled == enabled {
		return
	}
	if !enabled && c.popup && c.dismiss != nil {
		c.dismiss()
	}
	c.enabled = enabled
	if !enabled && c.form != nil {
		if c.form.focused == c {
			c.form.focused = nil
		}
		if c.form.pressed == c {
			c.form.pressed = nil
		}
		if c.form.hovered == c {
			c.form.setHovered(nil)
		}
	}
	c.AccessibilityChanged()
	c.AccessibilityChildrenChanged()
	c.Invalidate()
}

func (c *Control) SetTabStop(tabStop bool) {
	if c == nil || c.tabStop == tabStop {
		return
	}
	c.tabStop = tabStop
	c.AccessibilityChanged()
}

// SetAcceptsTab lets an editing control consume an unmodified Tab key. Other
// controls leave Tab to the form's normal focus traversal.
func (c *Control) SetAcceptsTab(accepts bool) {
	if c == nil || c.acceptsTab == accepts {
		return
	}
	c.acceptsTab = accepts
	c.AccessibilityChanged()
}

func (c *Control) SetThemeRole(role ThemeRole) {
	if c == nil || role < ThemeSurface || role > ThemeTransparent || c.themeRole == role {
		return
	}
	c.themeRole = role
	c.ApplyTheme(c.Theme())
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

func (c *Control) SetCursor(cursor graphics.Cursor) {
	if c != nil {
		c.cursor = cursor
	}
}

// SetThemeHandler lets an application-defined control map Theme fields onto
// its own properties. The current standalone or form theme is applied
// immediately, then reapplied on every form-level theme change.
func (c *Control) SetThemeHandler(handler func(theme Theme)) {
	if c == nil {
		return
	}
	if handler == nil {
		c.applyTheme = nil
	} else {
		c.applyTheme = handler
	}
	c.ApplyTheme(c.Theme())
}

func (c *Control) ApplyTheme(theme Theme) {
	if c == nil {
		return
	}
	c.theme = theme
	c.hasTheme = true
	if c.applyTheme == nil {
		c.applyBaseTheme(theme)
	} else {
		c.applyTheme(theme)
	}
}

func (c *Control) SetText(text string) {
	if c == nil || c.text == text {
		return
	}
	c.text = text
	c.AccessibilityChanged()
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
	width                           int
	height                          int
	background                      graphics.Color
	controls                        []*Control
	invalid                         []graphics.Rect
	focused                         *Control
	pressed                         *Control
	hovered                         *Control
	menuBar                         *MenuBar
	dockClient                      graphics.Rect
	layingOut                       bool
	theme                           Theme
	themeApplied                    bool
	accessibilityRevision           int
	accessibilityNextID             int
	accessibilityReset              bool
	accessibilityDirty              []*Control
	accessibilityChildrenDirty      []*Control
	accessibilityChildrenPatchDirty []*Control
	accessibilitySelectionDirty     []*Control
	accessibilityRemoved            []string

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
	f.theme = LightTheme()
	f.background = f.theme.Window
	f.controls = nil
	f.invalid = nil
	f.focused = nil
	f.pressed = nil
	f.hovered = nil
	f.menuBar = nil
	f.dockClient = f.clientRect()
	f.layingOut = false
	f.themeApplied = false
	f.accessibilityRevision = 0
	f.accessibilityNextID = 1
	f.accessibilityReset = true
	f.accessibilityDirty = nil
	f.accessibilityChildrenDirty = nil
	f.accessibilityChildrenPatchDirty = nil
	f.accessibilitySelectionDirty = nil
	f.accessibilityRemoved = nil
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
	f.performLayout()
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
	if f.themeApplied {
		control.ApplyTheme(f.theme)
	}
	control.form = f
	control.accessibilityID = f.uniqueAccessibilityID(control, control.accessibilityID)
	f.controls = append(f.controls, control)
	f.markAccessibilityChanged(control)
	f.performLayout()
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
		if control.accessibilityID != "" {
			f.accessibilityRemoved = append(f.accessibilityRemoved, control.accessibilityID)
		}
		control.form = nil
		if f.focused == control {
			f.focused = nil
		}
		if f.pressed == control {
			f.pressed = nil
		}
		if f.hovered == control {
			f.setHovered(nil)
		}
		if f.menuBar != nil && control == &f.menuBar.Control {
			f.menuBar = nil
		}
		f.performLayout()
		return
	}
}

// DockClientBounds returns the client-space rectangle left after laying out
// visible edge-docked controls. It is useful when a form combines docking with
// bespoke pane layout.
func (f *Form) DockClientBounds() graphics.Rect {
	if f == nil {
		return graphics.Rect{}
	}
	return f.dockClient
}

func (f *Form) performLayout() {
	if f == nil || f.layingOut {
		return
	}
	f.layingOut = true
	remaining := f.clientRect()
	for i := 0; i < len(f.controls); i++ {
		control := f.controls[i]
		if control == nil || !control.visible || control.dock == DockNone || control.dock == DockFill {
			continue
		}
		bounds := remaining
		if control.dock == DockTop || control.dock == DockBottom {
			height := control.dockHeight
			if height < 0 {
				height = 0
			}
			if height > remaining.Height() {
				height = remaining.Height()
			}
			if control.dock == DockTop {
				bounds.MaxY = bounds.MinY + height
				remaining.MinY += height
			} else {
				bounds.MinY = bounds.MaxY - height
				remaining.MaxY -= height
			}
		} else {
			width := control.dockWidth
			if width < 0 {
				width = 0
			}
			if width > remaining.Width() {
				width = remaining.Width()
			}
			if control.dock == DockLeft {
				bounds.MaxX = bounds.MinX + width
				remaining.MinX += width
			} else {
				bounds.MinX = bounds.MaxX - width
				remaining.MaxX -= width
			}
		}
		control.setLayoutBounds(bounds)
	}
	for i := 0; i < len(f.controls); i++ {
		control := f.controls[i]
		if control != nil && control.visible && control.dock == DockFill {
			control.setLayoutBounds(remaining)
		}
	}
	f.dockClient = remaining
	f.layingOut = false
}

func (f *Form) Controls() []*Control {
	if f == nil {
		return nil
	}
	controls := make([]*Control, len(f.controls))
	copy(controls, f.controls)
	return controls
}

func (f *Form) CursorAt(x, y graphics.Scalar) graphics.Cursor {
	if f == nil {
		return graphics.CursorArrow
	}
	control := f.hitTest(x, y)
	if control == nil {
		return graphics.CursorArrow
	}
	if control.PointerCursor != nil {
		return control.PointerCursor(x-control.bounds.MinX, y-control.bounds.MinY)
	}
	return control.cursor
}

// SetMenuBar registers the form's application menu for global shortcut and
// outside-click routing. Generated code still adds its Control explicitly so
// z-order remains visible in the generated component list.
func (f *Form) SetMenuBar(menuBar *MenuBar) {
	if f != nil {
		f.menuBar = menuBar
	}
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
			if !control.popup && control.visible && control.Paint != nil && rectIntersects(control.bounds, dirty) {
				control.Paint(surface)
			}
		}
		for j := 0; j < len(f.controls); j++ {
			control := f.controls[j]
			if control.popup && control.visible && control.Paint != nil && rectIntersects(control.bounds, dirty) {
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
	if event.Type == graphics.EventAccessibilityAction {
		id, value := accessibilityActionPayload(event.Text)
		f.performAccessibilityAction(id, AccessibilityAction(event.Key), value)
		return
	}
	if event.Type == graphics.EventWindowFocusLost {
		f.DismissTransientUI()
		f.pressed = nil
		f.setHovered(nil)
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
		f.setHovered(control)
		f.dismissPopupsExcept(control)
		if f.menuBar != nil && control != &f.menuBar.Control {
			f.menuBar.dismiss()
		}
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
		f.setHovered(hit)
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
		hit := f.hitTest(event.X, event.Y)
		f.setHovered(hit)
		control := f.pressed
		if control == nil {
			control = hit
		}
		if control != nil && control.PointerMove != nil {
			control.PointerMove(event.X-control.bounds.MinX, event.Y-control.bounds.MinY)
		}
		return
	}
	if event.Type == graphics.EventPointerLeave {
		f.setHovered(nil)
		f.DismissTransientUI()
		return
	}
	if event.Type == graphics.EventPointerWheel {
		control := f.hitTest(event.X, event.Y)
		if control != nil && control.PointerWheel != nil {
			control.PointerWheel(event.WheelX, event.WheelY)
		}
		return
	}
	if event.Type == graphics.EventKeyDown {
		if f.menuBar != nil && f.menuBar.Visible() && f.menuBar.Enabled() && f.menuBar.commandKey(event) {
			return
		}
		if event.Key == graphics.KeyTab && event.Modifiers&(graphics.ModifierControl|graphics.ModifierAlt|graphics.ModifierCommand) == 0 && (f.focused == nil || !f.focused.acceptsTab) {
			f.DismissTransientUI()
			f.moveFocus(event.Modifiers&graphics.ModifierShift == 0)
			return
		}
	}
	if event.Type == graphics.EventTextInput && f.menuBar != nil && f.menuBar.Visible() && f.menuBar.Enabled() && f.menuBar.commandText(event.Text) {
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

// DismissTransientUI closes menus and drop-downs without disturbing the
// control that owns keyboard focus. Hosts call it when the application loses
// activation, and applications may use it before presenting modal UI.
func (f *Form) DismissTransientUI() {
	if f == nil {
		return
	}
	f.dismissPopupsExcept(nil)
	if f.menuBar != nil {
		f.menuBar.dismiss()
	}
}

func (f *Form) moveFocus(forward bool) {
	if f == nil || len(f.controls) == 0 {
		return
	}
	start := -1
	for i := 0; i < len(f.controls); i++ {
		if f.controls[i] == f.focused {
			start = i
			break
		}
	}
	direction := -1
	if forward {
		direction = 1
	}
	index := start
	if start < 0 && !forward {
		index = 0
	}
	for count := 0; count < len(f.controls); count++ {
		index = (index + direction + len(f.controls)) % len(f.controls)
		control := f.controls[index]
		if control.visible && control.enabled && control.tabStop {
			f.setFocused(control)
			return
		}
	}
}

func (f *Form) setFocused(control *Control) {
	if f.focused == control {
		return
	}
	old := f.focused
	f.focused = control
	if old != nil {
		f.markAccessibilityChanged(old)
	}
	if control != nil {
		f.markAccessibilityChanged(control)
	}
	if old != nil {
		old.Invalidate()
	}
	if control != nil {
		control.Invalidate()
	}
}

func (f *Form) setHovered(control *Control) {
	if f == nil || f.hovered == control {
		return
	}
	old := f.hovered
	f.hovered = control
	if old != nil {
		if old.PointerLeave != nil {
			old.PointerLeave()
		}
		old.Invalidate()
	}
	if control != nil {
		if control.PointerEnter != nil {
			control.PointerEnter()
		}
		control.Invalidate()
	}
}

func (f *Form) hitTest(x, y graphics.Scalar) *Control {
	for i := len(f.controls) - 1; i >= 0; i-- {
		control := f.controls[i]
		if control.popup && control.visible && control.enabled && pointInRect(x, y, control.bounds) {
			return control
		}
	}
	for i := len(f.controls) - 1; i >= 0; i-- {
		control := f.controls[i]
		if !control.popup && control.visible && control.enabled && pointInRect(x, y, control.bounds) {
			return control
		}
	}
	return nil
}

func (f *Form) dismissPopupsExcept(keep *Control) {
	for i := 0; i < len(f.controls); i++ {
		control := f.controls[i]
		if control != keep && control.popup && control.dismiss != nil {
			control.dismiss()
		}
	}
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
