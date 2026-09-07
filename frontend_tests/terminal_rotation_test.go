package frontend_tests

import (
	. "renvo.dev/device/terminal"
	"renvo.dev/std/graphics"
	"testing"
)

func TestTerminalOrientationSwitchAndScroll(t *testing.T) {
	s := graphics.NewRotatedSurface(60, 100, graphics.PixelRGB565, graphics.Rotation0)
	display := &rotationDisplay{surface: s}
	console, err := Start(display, Options{Font: graphics.NewBuiltinFont(1), CellWidth: 6, CellHeight: 10, FlushPolicy: FlushManual, Scrollback: 8})
	if err != nil {
		t.Fatal(err)
	}
	defer console.Stop()
	pixels := &s.Pixels[0]
	console.WriteString("red\r\nblue")
	for i := 0; i < 10; i++ {
		rotation := graphics.Rotation90
		if i%2 != 0 {
			rotation = graphics.Rotation0
		}
		s.SetRotation(rotation)
		if !console.Flush() || console.Columns() != s.Width/6 || console.Rows() != s.Height/10 || &s.Pixels[0] != pixels {
			t.Fatal("refit failed")
		}
		for line := 0; line < 15; line++ {
			console.WriteString("\x1b[31mline\x1b[0m\r\n")
			if !console.Flush() {
				t.Fatal("scroll failed")
			}
		}
		// A full redraw of the model must exactly match the scrolling pixels.
		before := append([]byte(nil), s.Pixels...)
		console.ResizeToSurface()
		console.Flush()
		for p := range before {
			if before[p] != s.Pixels[p] {
				t.Fatalf("rotated scrolling differs at byte %d rotation %d", p, rotation)
			}
		}
	}
}

type rotationDisplay struct{ surface *graphics.Surface }

func TestTerminalCarriageReturnAndLineFeed(t *testing.T) {
	for _, rotation := range []graphics.Rotation{graphics.Rotation0, graphics.Rotation90} {
		for _, tc := range []struct {
			name, text  string
			row, column int
		}{
			{"CR", "abc\rX", 0, 1},
			{"LF", "abc\nX", 1, 4},
			{"CRLF", "abc\r\nX", 1, 1},
			{"startup lines", "touch\r\nready\r\nX", 2, 1},
			{"exact width CRLF", "0123456789\r\nX", 1, 1},
		} {
			s := graphics.NewRotatedSurface(60, 60, graphics.PixelRGB565, rotation)
			console, err := Start(&rotationDisplay{surface: s}, Options{Font: graphics.NewBuiltinFont(1), CellWidth: 6, CellHeight: 10, FlushPolicy: FlushManual})
			if err != nil {
				t.Fatal(err)
			}
			// Feed separately to cover CRLF split across writes, as with input.
			for i := range tc.text {
				console.WriteString(tc.text[i : i+1])
			}
			row, column := console.Cursor()
			cell, _ := console.CellAt(tc.row, tc.column-1)
			if row != tc.row || column != tc.column || cell.Character != 'X' || !console.Flush() {
				t.Errorf("%s rotation %d: cursor %d,%d, cell %q", tc.name, rotation, row, column, cell.Character)
			}
			console.Stop()
		}
	}
}

func (d *rotationDisplay) InitializeTerminal() (*graphics.Surface, bool) { return d.surface, true }
func (d *rotationDisplay) PresentTerminal(*graphics.Surface) bool        { return true }
