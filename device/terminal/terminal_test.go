package terminal

import (
	"bytes"
	"testing"

	"renvo.dev/std/graphics"
)

func terminalLine(t *Terminal, row int) string {
	line := make([]byte, t.Columns())
	for column := 0; column < t.Columns(); column++ {
		cell, _ := t.CellAt(row, column)
		line[column] = cell.Character
	}
	return string(line)
}

func TestTerminalWrapsAndPreservesExactWidthLine(t *testing.T) {
	state := New(5, 2, 4)
	state.WriteString("abcde")
	if row, column := state.Cursor(); row != 0 || column != 4 {
		t.Fatalf("cursor after exact width = %d,%d", row, column)
	}
	if got := terminalLine(state, 0); got != "abcde" {
		t.Fatalf("first line = %q", got)
	}
	state.WriteString("f")
	if got := terminalLine(state, 1); got != "f    " {
		t.Fatalf("wrapped line = %q", got)
	}
}

func TestTerminalScrollbackAndViewport(t *testing.T) {
	state := New(6, 3, 4)
	for _, text := range []string{"zero\r\n", "one\r\n", "two\r\n", "three\r\n", "four"} {
		state.WriteString(text)
	}
	if got := terminalLine(state, 0); got != "two   " {
		t.Fatalf("live first line = %q", got)
	}
	if got := terminalLine(state, 2); got != "four  " {
		t.Fatalf("live last line = %q", got)
	}
	state.Scroll(2)
	if got := terminalLine(state, 0); got != "zero  " {
		t.Fatalf("history first line = %q", got)
	}
	state.WriteString("!\r\n")
	if state.ViewOffset() != 3 || terminalLine(state, 0) != "zero  " {
		t.Fatalf("new output did not retain viewport: offset=%d line=%q", state.ViewOffset(), terminalLine(state, 0))
	}
	state.ScrollToBottom()
	if got := terminalLine(state, 1); got != "four! " {
		t.Fatalf("returned live line = %q", got)
	}
}

func TestTerminalANSIColorAndAttributes(t *testing.T) {
	state := New(12, 2, 0)
	state.WriteString("\x1b[31;44;1;4mR\x1b[38;2;1;2;3;48;5;46mX\x1b[0mN")
	red, _ := state.CellAt(0, 0)
	if red.Foreground != ansiColors[1] || red.Background != ansiColors[4] || !red.Bold || !red.Underline {
		t.Fatalf("basic SGR cell = %#v", red)
	}
	rgb, _ := state.CellAt(0, 1)
	if rgb.Foreground != graphics.RGBA(1, 2, 3, 255) || rgb.Background != ansi256Color(46) {
		t.Fatalf("extended SGR cell = %#v", rgb)
	}
	normal, _ := state.CellAt(0, 2)
	if normal.Foreground != defaultForeground || normal.Background != defaultBackground || normal.Bold || normal.Underline {
		t.Fatalf("reset SGR cell = %#v", normal)
	}
}

func TestTerminalCursorEraseAndVisibility(t *testing.T) {
	state := New(8, 3, 0)
	state.WriteString("first\r\nsecond\x1b[1;3H!\x1b[K")
	if got := terminalLine(state, 0); got != "fi!     " {
		t.Fatalf("cursor/erase line = %q", got)
	}
	state.WriteString("\x1b[?25l")
	if state.cursorVisible {
		t.Fatal("cursor remains visible")
	}
	state.WriteString("\x1b[?25h")
	if !state.cursorVisible {
		t.Fatal("cursor remains hidden")
	}
}

func TestTerminalWriteDoesNotAllocateAfterConstruction(t *testing.T) {
	state := New(40, 12, 100)
	allocations := testing.AllocsPerRun(100, func() {
		state.Reset()
		state.WriteString("\x1b[32mthe quick brown fox jumps over the lazy dog\x1b[0m\r\n")
	})
	if allocations != 0 {
		t.Fatalf("terminal write allocations = %v", allocations)
	}
}

type testDisplay struct {
	surface       *graphics.Surface
	frames        int
	damageRegions int
	damageArea    int
}

type testScrollDisplay struct {
	*testDisplay
	scrolls int
}

func (*testScrollDisplay) PrepareTerminal(*graphics.Surface, bool) bool { return true }

func (d *testScrollDisplay) ScrollTerminal(surface *graphics.Surface, top, bottom, pixels int) bool {
	d.scrolls++
	start := (top + pixels) * surface.Stride
	length := (bottom - top - pixels) * surface.Stride
	copy(surface.Pixels[top*surface.Stride:top*surface.Stride+length], surface.Pixels[start:start+length])
	return true
}

func (d *testDisplay) InitializeTerminal() (*graphics.Surface, bool) {
	return d.surface, d.surface != nil
}

func (d *testDisplay) PresentTerminal(surface *graphics.Surface) bool {
	d.frames++
	d.damageRegions = surface.DirtyRectCount()
	d.damageArea = 0
	for index := 0; index < d.damageRegions; index++ {
		if rect, ok := surface.DirtyRectAt(index); ok {
			d.damageArea += int(rect.Width() * rect.Height())
		}
	}
	return true
}

type testClock struct {
	now     uint32
	delayed uint32
}

type testPointer struct {
	x, y    int
	pressed bool
}

func (*testPointer) InitializePointer() bool { return true }

func (p *testPointer) ReadPointer() (x, y int, pressed, ok bool) {
	return p.x, p.y, p.pressed, true
}

func (c *testClock) Milliseconds() uint32 { return c.now }

func (c *testClock) DelayMilliseconds(milliseconds uint32) {
	c.delayed += milliseconds
	c.now += milliseconds
}

func TestStartMirrorsLineBufferedOutput(t *testing.T) {
	display := &testDisplay{surface: graphics.NewSurface(60, 40)}
	state, err := Start(display, Options{Columns: 10, Rows: 4, Font: graphics.NewBuiltinFont(1)})
	if err != nil {
		t.Fatal(err)
	}
	defer state.Stop()
	initialFrames := display.frames
	renvo_runtime_PrintMirror("hello")
	if display.frames != initialFrames {
		t.Fatalf("partial line presented: frames=%d", display.frames)
	}
	renvo_runtime_PrintMirror(" world\n")
	if display.frames != initialFrames+1 || terminalLine(state, 0) != "hello worl" {
		t.Fatalf("line mirror: frames=%d line=%q", display.frames, terminalLine(state, 0))
	}
}

func TestRenderCoalescesCellDamageIntoRowBands(t *testing.T) {
	display := &testDisplay{surface: graphics.NewSurface(60, 40)}
	state, err := Start(display, Options{Columns: 10, Rows: 4, Font: graphics.NewBuiltinFont(1)})
	if err != nil {
		t.Fatal(err)
	}
	defer state.Stop()
	if display.damageRegions != 1 {
		t.Fatalf("initial damage regions = %d, want one row band", display.damageRegions)
	}
	state.WriteString("\x1b[2;1Hcolored \x1b[31mtext\x1b[0m")
	state.Flush()
	if display.damageRegions != 1 {
		t.Fatalf("changed-row damage regions = %d, want 1", display.damageRegions)
	}
}

func TestTickPacesActiveTerminalWithoutDrift(t *testing.T) {
	display := &testDisplay{surface: graphics.NewSurface(60, 40)}
	clock := &testClock{}
	state, err := Start(display, Options{
		Columns: 10, Rows: 4, Font: graphics.NewBuiltinFont(1), Clock: clock,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer state.Stop()
	if !Tick(Second) || clock.delayed != 1000 {
		t.Fatalf("first tick delayed %d ms", clock.delayed)
	}
	clock.now += 250 // application work consumes part of the next interval
	if !state.Tick(Second) || clock.delayed != 1750 {
		t.Fatalf("paced ticker delayed %d ms total, want 1750", clock.delayed)
	}
}

func TestTouchKeyboardRepaintsOnlyChangedKey(t *testing.T) {
	display := &testDisplay{surface: graphics.NewSurface(200, 160)}
	pointer := &testPointer{}
	state, err := Start(display, Options{
		Columns: 20, Rows: 8, Font: graphics.NewBuiltinFont(1),
		CellWidth: 6, CellHeight: 10, Baseline: 7,
		Pointer: pointer, TouchKeyboard: true, KeyboardHeight: 80,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer state.Stop()

	key := state.keyboardRect(0)
	pointer.x = int((key.MinX + key.MaxX) / 2)
	pointer.y = int((key.MinY + key.MaxY) / 2)
	pointer.pressed = true
	if !state.Poll() {
		t.Fatal("press poll failed")
	}
	if display.damageRegions != 1 {
		t.Fatalf("key press damage regions = %d, want 1", display.damageRegions)
	}
	keyArea := int(key.Width() * key.Height())
	if display.damageArea != keyArea {
		t.Fatalf("key press damage area = %d, want %d", display.damageArea, keyArea)
	}

	pointer.pressed = false
	if !state.Poll() {
		t.Fatal("release poll failed")
	}
	if display.damageRegions != 1 {
		t.Fatalf("key release damage regions = %d, want 1", display.damageRegions)
	}
	if display.damageArea != keyArea {
		t.Fatalf("key release damage area = %d, want %d", display.damageArea, keyArea)
	}
	var input [1]byte
	count, _ := state.Read(input[:])
	if count != 1 || input[0] != '1' {
		t.Fatalf("keyboard input = %q, count %d", input[:count], count)
	}
}

func TestRenderedScrollMovesPixelsAndPaintsExposedRows(t *testing.T) {
	display := &testDisplay{surface: graphics.NewSurface(60, 30)}
	state, err := Start(display, Options{
		Columns: 10, Rows: 3, Font: graphics.NewBuiltinFont(1),
		CellWidth: 6, CellHeight: 10, Baseline: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer state.Stop()
	state.WriteString("zero\r\none\r\ntwo")
	state.Flush()
	state.WriteString("\r\nthree")
	if state.pendingScroll != 1 {
		t.Fatalf("pending pixel scroll = %d, want 1", state.pendingScroll)
	}
	dirtyRows := 0
	for row := 0; row < state.rows; row++ {
		if state.dirtyStart[row] < state.dirtyEnd[row] {
			dirtyRows++
		}
	}
	if dirtyRows > 2 {
		t.Fatalf("scroll dirtied %d text rows, want at most 2", dirtyRows)
	}
	state.Flush()
	optimized := append([]byte(nil), display.surface.Pixels...)
	state.markAllDirty()
	state.Flush()
	if !bytes.Equal(optimized, display.surface.Pixels) {
		t.Fatal("pixel-moved scroll differs from a complete repaint")
	}
}

func TestRenderedScrollUsesDisplayAccelerator(t *testing.T) {
	display := &testScrollDisplay{testDisplay: &testDisplay{surface: graphics.NewSurface(60, 30)}}
	state, err := Start(display, Options{
		Columns: 10, Rows: 3, Font: graphics.NewBuiltinFont(1),
		CellWidth: 6, CellHeight: 10, Baseline: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer state.Stop()
	state.WriteString("zero\r\none\r\ntwo")
	state.Flush()
	state.WriteString("\r\nthree")
	state.Flush()
	if display.scrolls != 1 {
		t.Fatalf("accelerated scroll calls = %d, want 1", display.scrolls)
	}
	optimized := append([]byte(nil), display.surface.Pixels...)
	state.markAllDirty()
	state.Flush()
	if !bytes.Equal(optimized, display.surface.Pixels) {
		t.Fatal("accelerated scroll differs from a complete repaint")
	}
}

func BenchmarkTerminalWriteAndScroll(b *testing.B) {
	state := New(80, 30, 200)
	line := "\x1b[38;5;45m0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ\x1b[0m\r\n"
	b.ReportAllocs()
	b.SetBytes(int64(len(line)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.WriteString(line)
	}
}

func BenchmarkTerminalRenderChangedLine(b *testing.B) {
	display := &testDisplay{surface: graphics.NewSurface(480, 300)}
	state, err := Start(display, Options{
		Columns: 80, Rows: 30, Font: graphics.NewBuiltinFont(1),
		FlushPolicy: FlushManual,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer state.Stop()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.WriteString("\x1b[1;1Hframe")
		state.Flush()
	}
}

func BenchmarkTerminalRenderScrolledLine(b *testing.B) {
	display := &testDisplay{surface: graphics.NewSurface(480, 300)}
	state, err := Start(display, Options{
		Columns: 80, Rows: 30, Font: graphics.NewBuiltinFont(1),
		CellWidth: 6, CellHeight: 10, Baseline: 7,
		FlushPolicy: FlushManual,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer state.Stop()
	for line := 0; line < 30; line++ {
		state.WriteString("prefill the visible terminal viewport\r\n")
	}
	state.Flush()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.WriteString("X=123, Y=-456, Z=789\r\n")
		state.Flush()
	}
}
