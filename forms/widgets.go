package forms

import "renvo.dev/std/graphics"

// Label is the Forms text-display control. Generated form code assigns its
// bounds, font, colors, and text through the retained property setters.
type Label struct {
	Control
	font *graphics.Font
}

func NewLabel() *Label {
	label := &Label{}
	label.Control = *NewControl()
	label.SetTabStop(false)
	label.SetAccessibilityRole(AccessibilityRoleLabel)
	label.Control.applyTheme = label.applyTheme
	label.applyTheme(LightTheme())
	label.Paint = label.paint
	return label
}

func (l *Label) applyTheme(theme Theme) { applyTransparentTheme(&l.Control, theme) }

func (l *Label) Font() *graphics.Font { return l.font }

func (l *Label) SetFont(font *graphics.Font) {
	if l == nil || l.font == font {
		return
	}
	l.font = font
	l.Invalidate()
}

func (l *Label) paint(surface *graphics.Surface) {
	if l.font == nil {
		return
	}
	bounds := l.Bounds()
	baseline := bounds.MinY + (bounds.Height()-labelLineHeight(l.font))/2 + l.font.Metrics.Ascent
	surface.DrawText(l.font, graphics.Point{X: bounds.MinX, Y: baseline}, l.Text(), controlForeground(&l.Control))
}

// Button is the Forms push-button control. Click remains an ordinary event
// callback on the embedded Control, matching generated WinForms-style wiring.
type Button struct {
	Control
	font    *graphics.Font
	pressed bool
}

func NewButton() *Button {
	button := &Button{}
	button.Control = *NewControl()
	button.SetAccessibilityRole(AccessibilityRoleButton)
	button.SetCursor(graphics.CursorPointingHand)
	button.Control.applyTheme = button.applyTheme
	button.applyTheme(LightTheme())
	button.Paint = button.paint
	button.PointerDown = button.pointerDown
	button.PointerUp = button.pointerUp
	button.KeyDown = button.keyDown
	return button
}

func (b *Button) keyDown(event graphics.Event) {
	if b == nil || event.Repeat {
		return
	}
	if (event.Key == graphics.KeyEnter || event.Key == graphics.KeySpace) && b.Click != nil {
		b.Click()
	}
}

func (b *Button) applyTheme(theme Theme) {
	b.SetBackground(theme.Accent)
	b.SetForeground(theme.AccentText)
}

func (b *Button) Font() *graphics.Font { return b.font }

func (b *Button) SetFont(font *graphics.Font) {
	if b == nil || b.font == font {
		return
	}
	b.font = font
	b.Invalidate()
}

func (b *Button) pointerDown(x, y graphics.Scalar) {
	if b == nil || b.pressed {
		return
	}
	b.pressed = true
	b.Invalidate()
}

func (b *Button) pointerUp(x, y graphics.Scalar) {
	if b == nil || !b.pressed {
		return
	}
	b.pressed = false
	b.Invalidate()
}

func (b *Button) paint(surface *graphics.Surface) {
	bounds := b.Bounds()
	background := b.Background()
	foreground := controlForeground(&b.Control)
	theme := controlTheme(&b.Control)
	if !b.Enabled() {
		background = theme.SurfaceRaised
	} else if b.Hovered() && !b.pressed {
		background = theme.Selection
		foreground = theme.Accent
	} else if b.pressed {
		background = shadeButtonColor(background)
	}
	surface.FillRect(bounds, background)
	surface.StrokeRect(bounds, 1, controlAccent(&b.Control))
	if b.font == nil || b.Text() == "" {
		return
	}
	metrics := graphics.MeasureText(b.font, b.Text())
	x := bounds.MinX + (bounds.Width()-metrics.Width)/2
	baseline := bounds.MinY + (bounds.Height()-metrics.Height)/2 + b.font.Metrics.Ascent
	surface.DrawText(b.font, graphics.Point{X: x, Y: baseline}, b.Text(), foreground)
}

func labelLineHeight(font *graphics.Font) graphics.Scalar {
	if font == nil {
		return 0
	}
	return font.Metrics.Ascent + font.Metrics.Descent + font.Metrics.LineGap
}

func shadeButtonColor(color graphics.Color) graphics.Color {
	red := int(color.R) * 4 / 5
	green := int(color.G) * 4 / 5
	blue := int(color.B) * 4 / 5
	return graphics.RGBA(byte(red), byte(green), byte(blue), color.A)
}

// TextBox is a single-line editable text control. Text editing is expressed
// through the same property setter used by generated and user code.
type TextBox struct {
	Control
	font *graphics.Font
}

func NewTextBox() *TextBox {
	box := &TextBox{}
	box.Control = *NewControl()
	box.SetAccessibilityRole(AccessibilityRoleTextBox)
	box.SetCursor(graphics.CursorIBeam)
	box.Control.applyTheme = box.applyTheme
	box.applyTheme(LightTheme())
	box.Paint = box.paint
	box.TextInput = box.textInput
	box.KeyDown = box.keyDown
	return box
}

func (b *TextBox) applyTheme(theme Theme) { applyFieldTheme(&b.Control, theme) }

func (b *TextBox) Font() *graphics.Font { return b.font }

func (b *TextBox) SetFont(font *graphics.Font) {
	if b == nil || b.font == font {
		return
	}
	b.font = font
	b.Invalidate()
}

func (b *TextBox) paint(surface *graphics.Surface) {
	paintTextEntry(surface, &b.Control, b.font, false)
}

func (b *TextBox) textInput(text string) {
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' || text[i] == '\r' {
			return
		}
	}
	b.SetText(b.Text() + text)
}

func (b *TextBox) keyDown(event graphics.Event) {
	if event.Key == graphics.KeyBackspace && len(b.Text()) > 0 {
		b.SetText(removeLastUTF8(b.Text()))
	}
}

// TextArea is a multiline editable text control.
type TextArea struct {
	Control
	font *graphics.Font
}

func NewTextArea() *TextArea {
	area := &TextArea{}
	area.Control = *NewControl()
	area.SetAccessibilityRole(AccessibilityRoleTextBox)
	area.SetAccessibilityMultiline(true)
	area.SetCursor(graphics.CursorIBeam)
	area.Control.applyTheme = area.applyTheme
	area.applyTheme(LightTheme())
	area.Paint = area.paint
	area.TextInput = area.textInput
	area.KeyDown = area.keyDown
	return area
}

func (a *TextArea) applyTheme(theme Theme) { applyFieldTheme(&a.Control, theme) }

func (a *TextArea) Font() *graphics.Font { return a.font }

func (a *TextArea) SetFont(font *graphics.Font) {
	if a == nil || a.font == font {
		return
	}
	a.font = font
	a.Invalidate()
}

func (a *TextArea) paint(surface *graphics.Surface) {
	paintTextEntry(surface, &a.Control, a.font, true)
}

func (a *TextArea) textInput(text string) { a.SetText(a.Text() + text) }

func (a *TextArea) keyDown(event graphics.Event) {
	if event.Key == graphics.KeyBackspace && len(a.Text()) > 0 {
		a.SetText(removeLastUTF8(a.Text()))
	} else if event.Key == graphics.KeyEnter {
		a.SetText(a.Text() + "\n")
	}
}

func paintTextEntry(surface *graphics.Surface, control *Control, font *graphics.Font, multiline bool) {
	bounds := control.Bounds()
	surface.FillRect(bounds, control.Background())
	theme := controlTheme(control)
	border := theme.Border
	if control.Hovered() {
		border = theme.MutedText
	}
	if control.Focused() {
		border = theme.Accent
	}
	surface.StrokeRect(bounds, 1, border)
	if font == nil {
		return
	}
	clip := graphics.R(bounds.MinX+5, bounds.MinY+2, bounds.Width()-10, bounds.Height()-4)
	surface.PushClipRect(clip)
	lineHeight := labelLineHeight(font)
	lineStart := 0
	line := 0
	text := control.Text()
	for i := 0; i <= len(text); i++ {
		if i < len(text) && text[i] != '\n' {
			continue
		}
		baseline := bounds.MinY + 4 + graphics.Scalar(line)*lineHeight + font.Metrics.Ascent
		if !multiline {
			baseline = bounds.MinY + (bounds.Height()-lineHeight)/2 + font.Metrics.Ascent
		}
		surface.DrawText(font, graphics.Point{X: bounds.MinX + 6, Y: baseline}, text[lineStart:i], controlForeground(control))
		line++
		lineStart = i + 1
		if !multiline {
			break
		}
	}
	surface.PopClip()
}

func removeLastUTF8(text string) string {
	i := len(text) - 1
	for i > 0 && text[i]&0xc0 == 0x80 {
		i--
	}
	return text[:i]
}

// CheckBox provides an independently toggled boolean property.
type CheckBox struct {
	Control
	font    *graphics.Font
	checked bool
}

func NewCheckBox() *CheckBox {
	box := &CheckBox{}
	box.Control = *NewControl()
	box.SetAccessibilityRole(AccessibilityRoleCheckBox)
	box.SetCursor(graphics.CursorPointingHand)
	box.AccessibilityCheckable = true
	box.AccessibilityChecked = box.accessibilityChecked
	box.AccessibilityInvoke = box.accessibilityInvoke
	box.Control.applyTheme = box.applyTheme
	box.applyTheme(LightTheme())
	box.Paint = box.paint
	box.PointerUp = box.pointerUp
	box.KeyDown = box.keyDown
	return box
}

func (b *CheckBox) keyDown(event graphics.Event) {
	if b != nil && event.Key == graphics.KeySpace && !event.Repeat {
		b.accessibilityInvoke()
	}
}

func (b *CheckBox) applyTheme(theme Theme) { applyTransparentTheme(&b.Control, theme) }

func (b *CheckBox) Font() *graphics.Font { return b.font }
func (b *CheckBox) Checked() bool        { return b.checked }

func (b *CheckBox) SetFont(font *graphics.Font) {
	if b != nil && b.font != font {
		b.font = font
		b.Invalidate()
	}
}

func (b *CheckBox) SetChecked(checked bool) {
	if b != nil && b.checked != checked {
		b.checked = checked
		b.AccessibilityChanged()
		b.Invalidate()
	}
}

func (b *CheckBox) accessibilityChecked() bool { return b.checked }

func (b *CheckBox) accessibilityInvoke() {
	b.SetChecked(!b.checked)
	if b.Click != nil {
		b.Click()
	}
}

func (b *CheckBox) pointerUp(x, y graphics.Scalar) {
	bounds := b.Bounds()
	if x >= 0 && y >= 0 && x < bounds.Width() && y < bounds.Height() {
		b.SetChecked(!b.checked)
	}
}

func (b *CheckBox) paint(surface *graphics.Surface) {
	bounds := b.Bounds()
	theme := controlTheme(&b.Control)
	if b.Hovered() {
		surface.FillRect(bounds, theme.Hover)
	}
	box := graphics.R(bounds.MinX+1, bounds.MinY+(bounds.Height()-18)/2, 18, 18)
	fill := theme.Field
	border := theme.MutedText
	if b.checked {
		fill = controlAccent(&b.Control)
		border = controlAccent(&b.Control)
	}
	surface.FillRect(box, fill)
	surface.StrokeRect(box, 1, border)
	if b.checked {
		drawIcon(surface, IconCheck, box.MinX+1, box.MinY+1, theme.AccentText)
	}
	paintChoiceText(surface, bounds, b.font, b.Text(), controlForeground(&b.Control))
}

// RadioButton is an individually checkable radio choice. Group exclusivity
// can be implemented by the containing form's event callback.
type RadioButton struct {
	Control
	font    *graphics.Font
	checked bool
}

func NewRadioButton() *RadioButton {
	button := &RadioButton{}
	button.Control = *NewControl()
	button.SetAccessibilityRole(AccessibilityRoleRadioButton)
	button.SetCursor(graphics.CursorPointingHand)
	button.AccessibilityCheckable = true
	button.AccessibilityChecked = button.accessibilityChecked
	button.AccessibilityInvoke = button.accessibilityInvoke
	button.Control.applyTheme = button.applyTheme
	button.applyTheme(LightTheme())
	button.Paint = button.paint
	button.PointerUp = button.pointerUp
	button.KeyDown = button.keyDown
	return button
}

func (b *RadioButton) keyDown(event graphics.Event) {
	if b != nil && event.Key == graphics.KeySpace && !event.Repeat {
		b.accessibilityInvoke()
	}
}

func (b *RadioButton) applyTheme(theme Theme) { applyTransparentTheme(&b.Control, theme) }

func (b *RadioButton) Font() *graphics.Font { return b.font }
func (b *RadioButton) Checked() bool        { return b.checked }

func (b *RadioButton) SetFont(font *graphics.Font) {
	if b != nil && b.font != font {
		b.font = font
		b.Invalidate()
	}
}

func (b *RadioButton) SetChecked(checked bool) {
	if b != nil && b.checked != checked {
		b.checked = checked
		b.AccessibilityChanged()
		b.Invalidate()
	}
}

func (b *RadioButton) accessibilityChecked() bool { return b.checked }

func (b *RadioButton) accessibilityInvoke() {
	b.SetChecked(true)
	if b.Click != nil {
		b.Click()
	}
}

func (b *RadioButton) pointerUp(x, y graphics.Scalar) {
	bounds := b.Bounds()
	if x >= 0 && y >= 0 && x < bounds.Width() && y < bounds.Height() {
		b.SetChecked(true)
	}
}

func (b *RadioButton) paint(surface *graphics.Surface) {
	bounds := b.Bounds()
	theme := controlTheme(&b.Control)
	if b.Hovered() {
		surface.FillRect(bounds, theme.Hover)
	}
	circle := graphics.R(bounds.MinX, bounds.MinY+(bounds.Height()-16)/2, 16, 16)
	surface.FillEllipse(circle, theme.Field)
	surface.StrokeEllipse(circle, 1, theme.MutedText)
	if b.checked {
		surface.FillEllipse(graphics.R(circle.MinX+4, circle.MinY+4, 8, 8), controlAccent(&b.Control))
	}
	paintChoiceText(surface, bounds, b.font, b.Text(), controlForeground(&b.Control))
}

func paintChoiceText(surface *graphics.Surface, bounds graphics.Rect, font *graphics.Font, text string, color graphics.Color) {
	if font == nil || text == "" {
		return
	}
	baseline := bounds.MinY + (bounds.Height()-labelLineHeight(font))/2 + font.Metrics.Ascent
	surface.DrawText(font, graphics.Point{X: bounds.MinX + 23, Y: baseline}, text, color)
}

// PictureBox reserves an image surface in generated layouts. Image decoding
// and assignment can be layered on without changing designer source shape.
type PictureBox struct{ Control }

func NewPictureBox() *PictureBox {
	box := &PictureBox{}
	box.Control = *NewControl()
	box.SetTabStop(false)
	box.SetAccessibilityRole(AccessibilityRoleImage)
	box.Control.applyTheme = box.applyTheme
	box.applyTheme(LightTheme())
	box.Paint = box.paint
	return box
}

func (b *PictureBox) applyTheme(theme Theme) { applyRaisedTheme(&b.Control, theme) }

func (b *PictureBox) paint(surface *graphics.Surface) {
	bounds := b.Bounds()
	surface.FillRect(bounds, b.Background())
	theme := controlTheme(&b.Control)
	surface.StrokeRect(bounds, 1, theme.Border)
	color := theme.MutedText
	if !b.Enabled() {
		color = theme.Disabled
	}
	surface.DrawLine(graphics.Point{X: bounds.MinX + 5, Y: bounds.MaxY - 6}, graphics.Point{X: bounds.MinX + bounds.Width()/2, Y: bounds.MinY + bounds.Height()/2}, 1, color)
	surface.DrawLine(graphics.Point{X: bounds.MinX + bounds.Width()/2, Y: bounds.MinY + bounds.Height()/2}, graphics.Point{X: bounds.MaxX - 5, Y: bounds.MaxY - 6}, 1, color)
}

// Panel is a visual container boundary. Child control ownership remains with
// the form for now, so generated z-order stays explicit and deterministic.
type Panel struct{ Control }

func NewPanel() *Panel {
	panel := &Panel{}
	panel.Control = *NewControl()
	panel.SetTabStop(false)
	panel.Control.applyTheme = panel.applyTheme
	panel.applyTheme(LightTheme())
	panel.Paint = panel.paint
	return panel
}

func (p *Panel) applyTheme(theme Theme) { applyRaisedTheme(&p.Control, theme) }

func (p *Panel) paint(surface *graphics.Surface) {
	surface.FillRect(p.Bounds(), p.Background())
	surface.StrokeRect(p.Bounds(), 1, controlTheme(&p.Control).Border)
}
