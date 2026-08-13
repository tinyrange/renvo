package board

import (
	"testing"

	"renvo.dev/device/input/tca8418"
)

type fakeKeypad struct {
	events []tca8418.Event
	index  int
}

func (*fakeKeypad) Initialize() error { return nil }
func (k *fakeKeypad) NextEvent() (tca8418.Event, bool, error) {
	if k.index == len(k.events) {
		return tca8418.Event{}, false, nil
	}
	event := k.events[k.index]
	k.index++
	return event, true, nil
}

func rawEvent(row, column uint8, pressed bool) tca8418.Event {
	return tca8418.Event{Row: row, Column: column, Pressed: pressed}
}

func TestDisplayDimensions(t *testing.T) {
	if Display.Width() != 240 || Display.Height() != 135 {
		t.Fatalf("display dimensions = %dx%d", Display.Width(), Display.Height())
	}
}

func TestKeyboardRemapCorners(t *testing.T) {
	tests := []struct {
		rawRow, rawColumn uint8
		row, column       int
	}{
		{0, 0, 0, 0},
		{0, 7, 3, 1},
		{6, 0, 0, 12},
		{6, 7, 3, 13},
	}
	for _, test := range tests {
		row, column := remapKey(rawEvent(test.rawRow, test.rawColumn, true))
		if row != test.row || column != test.column {
			t.Fatalf("remap (%d,%d) = (%d,%d), want (%d,%d)",
				test.rawRow, test.rawColumn, row, column, test.row, test.column)
		}
	}
}

func TestKeyboardShiftAndFunctionLayers(t *testing.T) {
	// Raw (0,6) is logical Shift (row 2, column 1), raw (1,2) is
	// logical A, and raw (0,2) is logical Fn.
	device := &fakeKeypad{events: []tca8418.Event{
		rawEvent(0, 6, true),
		rawEvent(1, 2, true),
		rawEvent(0, 6, false),
		rawEvent(0, 2, true),
		rawEvent(5, 6, true), // logical semicolon/up key
	}}
	keyboard := newKeyboard(device)
	wantKeys := []Key{KeyShift, 'a', KeyShift, KeyFunction, KeyUp}
	wantCharacters := []byte{0, 'A', 0, 0, 0}
	for index := range wantKeys {
		event, ok, err := keyboard.NextEvent()
		if err != nil || !ok {
			t.Fatalf("event %d = %#v, %t, %v", index, event, ok, err)
		}
		if event.Key != wantKeys[index] || event.Character != wantCharacters[index] {
			t.Fatalf("event %d = %#v, want key %d character %q", index, event, wantKeys[index], wantCharacters[index])
		}
	}
}

func TestRGB565(t *testing.T) {
	if got := rgb565(255, 0, 0); got != 0xf800 {
		t.Fatalf("rgb565(red) = %#04x", got)
	}
}
