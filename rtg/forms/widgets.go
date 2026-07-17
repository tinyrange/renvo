package forms

import "j5.nz/rtg/rtg/std/graphics"

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
	label.SetBackground(graphics.RGBA(255, 255, 255, 0))
	label.Paint = label.paint
	return label
}

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
	surface.DrawText(l.font, graphics.Point{X: bounds.MinX, Y: baseline}, l.Text(), l.Foreground())
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
	button.SetBackground(graphics.RGBA(25, 118, 210, 255))
	button.SetForeground(graphics.White)
	button.Paint = button.paint
	button.PointerDown = button.pointerDown
	button.PointerUp = button.pointerUp
	return button
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
	if b.pressed {
		background = shadeButtonColor(background)
	}
	surface.FillRect(bounds, background)
	surface.StrokeRect(bounds, 1, shadeButtonColor(background))
	if b.font == nil || b.Text() == "" {
		return
	}
	metrics := graphics.MeasureText(b.font, b.Text())
	x := bounds.MinX + (bounds.Width()-metrics.Width)/2
	baseline := bounds.MinY + (bounds.Height()-metrics.Height)/2 + b.font.Metrics.Ascent
	surface.DrawText(b.font, graphics.Point{X: x, Y: baseline}, b.Text(), b.Foreground())
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
