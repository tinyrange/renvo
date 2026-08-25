package main

import "renvo.dev/std/graphics"

const (
	keyboardTop    = 746
	keyboardBottom = 1208
)

func (overlay *touchOverlay) showKeyboard() {
	overlay.keyboardVisible = true
	overlay.keyboardFullPasses = 2
	overlay.keyboardPressed = -1
	overlay.keyboardPrevious = -1
	if overlay.Form() != nil {
		overlay.Form().Invalidate(graphics.R(0, keyboardTop, width, keyboardBottom-keyboardTop))
	}
}

func (overlay *touchOverlay) hideKeyboard() {
	if !overlay.keyboardVisible {
		return
	}
	overlay.keyboardVisible = false
	overlay.keyboardPressed = -1
	if overlay.Form() != nil {
		overlay.Form().Invalidate(graphics.R(0, keyboardTop, width, keyboardBottom-keyboardTop))
	}
}

func keyboardIndexAt(x, y graphics.Scalar) int {
	yy := int(y)
	if yy < keyboardTop || yy >= keyboardBottom {
		return -1
	}
	row := (yy - keyboardTop) / 88
	localY := (yy - keyboardTop) % 88
	if row > 4 || localY < 6 || localY >= 86 {
		return -1
	}
	if row == 0 || row == 1 {
		position := int(x) - 13
		if position < 0 || position%(64+6) >= 64 || position/(64+6) >= 10 {
			return -1
		}
		return row*10 + position/(64+6)
	}
	if row == 2 {
		position := int(x) - 48
		if position < 0 || position%(64+6) >= 64 || position/(64+6) >= 9 {
			return -1
		}
		return 20 + position/(64+6)
	}
	if row == 3 {
		if x >= 9 && x < 103 {
			return 29
		}
		if x >= 109 && x < 593 {
			position := int(x) - 109
			if position%(64+6) < 64 {
				return 30 + position/(64+6)
			}
		}
		if x >= 599 && x < 711 {
			return 37
		}
		return -1
	}
	if x >= 19 && x < 539 {
		return 38
	}
	if x >= 547 && x < 701 {
		return 39
	}
	return -1
}

func keyboardRect(index int) graphics.Rect {
	if index < 20 {
		row := index / 10
		column := index % 10
		return graphics.R(graphics.Scalar(13+column*(64+6)), graphics.Scalar(keyboardTop+row*88+6), 64, 80)
	}
	if index < 29 {
		return graphics.R(graphics.Scalar(48+(index-20)*(64+6)), keyboardTop+2*88+6, 64, 80)
	}
	if index == 29 {
		return graphics.R(9, keyboardTop+3*88+6, 94, 80)
	}
	if index < 37 {
		return graphics.R(graphics.Scalar(109+(index-30)*(64+6)), keyboardTop+3*88+6, 64, 80)
	}
	if index == 37 {
		return graphics.R(599, keyboardTop+3*88+6, 112, 80)
	}
	if index == 38 {
		return graphics.R(19, keyboardTop+4*88+6, 520, 80)
	}
	return graphics.R(547, keyboardTop+4*88+6, 154, 80)
}

func keyboardLabel(index int) string {
	if index < 10 {
		return "1234567890"[index : index+1]
	}
	if index < 20 {
		return "qwertyuiop"[index-10 : index-9]
	}
	if index < 29 {
		return "asdfghjkl"[index-20 : index-19]
	}
	if index == 29 {
		return "Shift"
	}
	if index < 37 {
		return "zxcvbnm"[index-30 : index-29]
	}
	if index == 37 {
		return "Back"
	}
	if index == 38 {
		return "space"
	}
	return "Done"
}

func (overlay *touchOverlay) markKeyboardKey(index int) {
	if overlay.keyboardPressed == index {
		return
	}
	overlay.keyboardPrevious = overlay.keyboardPressed
	overlay.keyboardPressed = index
	if overlay.Form() != nil {
		if overlay.keyboardPrevious >= 0 {
			overlay.Form().Invalidate(keyboardRect(overlay.keyboardPrevious))
		}
		if index >= 0 {
			overlay.Form().Invalidate(keyboardRect(index))
		}
	}
}

func (overlay *touchOverlay) keyboardDown(x, y graphics.Scalar) {
	overlay.markKeyboardKey(keyboardIndexAt(x, y))
}

func (overlay *touchOverlay) keyboardMove(x, y graphics.Scalar) {
	overlay.markKeyboardKey(keyboardIndexAt(x, y))
}

func (overlay *touchOverlay) keyboardUp(x, y graphics.Scalar) {
	pressed := overlay.keyboardPressed
	hit := keyboardIndexAt(x, y)
	overlay.markKeyboardKey(-1)
	if pressed < 0 || pressed != hit {
		return
	}
	if pressed == 29 {
		overlay.keyboardShift = !overlay.keyboardShift
		overlay.keyboardFullPasses = 2
		overlay.Form().Invalidate(graphics.R(0, keyboardTop, width, keyboardBottom-keyboardTop))
		return
	}
	if pressed == 37 {
		text := overlay.target.Text()
		if len(text) > 0 {
			overlay.target.SetText(text[:len(text)-1])
		}
		return
	}
	if pressed == 38 {
		overlay.target.SetText(overlay.target.Text() + " ")
		return
	}
	if pressed == 39 {
		overlay.hideKeyboard()
		return
	}
	value := keyboardLabel(pressed)
	if overlay.keyboardShift && pressed >= 10 {
		if pressed < 20 {
			value = "QWERTYUIOP"[pressed-10 : pressed-9]
		} else if pressed < 29 {
			value = "ASDFGHJKL"[pressed-20 : pressed-19]
		} else {
			value = "ZXCVBNM"[pressed-30 : pressed-29]
		}
		overlay.keyboardShift = false
		overlay.keyboardFullPasses = 2
		overlay.Form().Invalidate(graphics.R(0, keyboardTop, width, keyboardBottom-keyboardTop))
	}
	overlay.target.SetText(overlay.target.Text() + value)
}

func (overlay *touchOverlay) paintKeyboardKey(canvas graphics.Canvas, index int, theme graphics.Color, border graphics.Color, text graphics.Color) {
	rect := keyboardRect(index)
	color := theme
	if overlay.keyboardPressed == index || index == 29 && overlay.keyboardShift {
		color = overlay.Theme().Selection
	}
	canvas.FillRect(rect, color)
	canvas.StrokeRect(rect, 2, border)
	label := keyboardLabel(index)
	x := rect.MinX + rect.Width()/2 - graphics.Scalar(len(label)*5)
	baseline := rect.MinY + 49
	canvas.DrawText(overlay.font, graphics.Point{X: x, Y: baseline}, label, text)
}

func (overlay *touchOverlay) paintKeyboard(canvas graphics.Canvas) {
	theme := overlay.Theme()
	if overlay.keyboardFullPasses > 0 {
		canvas.FillRect(graphics.R(0, keyboardTop, width, keyboardBottom-keyboardTop), theme.SurfaceRaised)
		for index := 0; index < 40; index++ {
			overlay.paintKeyboardKey(canvas, index, theme.Field, theme.Border, theme.Text)
		}
		return
	}
	if overlay.keyboardPrevious >= 0 {
		overlay.paintKeyboardKey(canvas, overlay.keyboardPrevious, theme.Field, theme.Border, theme.Text)
	}
	if overlay.keyboardPressed >= 0 && overlay.keyboardPressed != overlay.keyboardPrevious {
		overlay.paintKeyboardKey(canvas, overlay.keyboardPressed, theme.Field, theme.Border, theme.Text)
	}
}

// afterPaint counts application presentations, not individual clipped paint
// callbacks. Two complete passes keep both alternating framebuffers valid.
func (overlay *touchOverlay) afterPaint() {
	if overlay.keyboardFullPasses > 0 {
		overlay.keyboardFullPasses--
	}
}
