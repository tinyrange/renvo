package main

import (
	"testing"

	"renvo.dev/forms"
	"renvo.dev/std/graphics"
)

func TestTouchKeyboardLayersEditingAndBound(t *testing.T) {
	target := forms.NewControl()
	keyboard := newTouchKeyboard(graphics.NewBuiltinFont(1))
	keyboard.show(target, false)

	keyboard.commitKey(int('q'))
	keyboard.commitKey(int('w'))
	if target.Text() != "Qw" {
		t.Fatalf("one-shot shift text = %q, want Qw", target.Text())
	}
	keyboard.commitKey(keyShift)
	keyboard.commitKey(keyShift)
	keyboard.commitKey(int('e'))
	keyboard.commitKey(int('r'))
	if target.Text() != "QwER" {
		t.Fatalf("caps text = %q, want QwER", target.Text())
	}

	target.SetText("é")
	keyboard.commitKey(keyBackspace)
	if target.Text() != "" {
		t.Fatalf("UTF-8 backspace left %q", target.Text())
	}

	text := ""
	for len(text) < keyboardMaximumTextBytes {
		text += "x"
	}
	target.SetText(text)
	keyboard.commitKey(int('a'))
	if len(target.Text()) != keyboardMaximumTextBytes {
		t.Fatalf("bounded text length = %d", len(target.Text()))
	}
}

func TestTouchKeyboardHitGeometry(t *testing.T) {
	keyboard := newTouchKeyboard(graphics.NewBuiltinFont(1))
	for _, test := range []struct {
		x, y graphics.Scalar
		key  int
	}{
		{21, 32, int('q')},
		{127, 32, int('r')},
		{91, 92, int('s')},
		{180, 152, int('v')},
		{28, 152, keyShift},
		{332, 152, keyBackspace},
		{170, 212, keySpace},
		{322, 212, keyDone},
	} {
		if got := keyboard.keyAt(test.x, test.y); got != test.key {
			t.Errorf("keyAt(%v, %v) = %d, want %d", test.x, test.y, got, test.key)
		}
	}
}

func TestRenderTimingDisplayUsesRollingWindow(t *testing.T) {
	d := controlsDemo{renderTimes: make([]int, renderTimingSampleCount)}
	if got := d.renderTimingText(); got != "Render timing: waiting for first frame" {
		t.Fatalf("empty timing text = %q", got)
	}
	for i := 0; i < renderTimingSampleCount+1; i++ {
		d.renderTimes[d.renderNext] = (i + 1) * 1000
		d.renderNext = (d.renderNext + 1) % renderTimingSampleCount
		if d.renderCount < renderTimingSampleCount {
			d.renderCount++
		}
	}
	if got := d.renderTimingText(); got != "Render 21.0 ms  avg 11.5  peak 21.0" {
		t.Fatalf("rolling timing text = %q", got)
	}
}
