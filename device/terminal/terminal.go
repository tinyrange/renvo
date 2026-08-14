// Package terminal provides a small VT-style terminal emulator for embedded
// displays. A started terminal mirrors Renvo's standard output while remaining
// usable directly as an io.Writer.
package terminal

import "renvo.dev/std/graphics"

// FlushPolicy controls when mirrored standard output is presented.
type FlushPolicy byte

const (
	// FlushLine presents after a carriage return, line feed, or bell. It avoids
	// repainting once for every argument in a built-in print call.
	FlushLine FlushPolicy = iota
	// FlushEveryWrite presents after every mirrored write.
	FlushEveryWrite
	// FlushManual presents only when Flush or Poll is called.
	FlushManual
)

// Display is the board-specific surface and presentation capability required
// by Start. InitializeTerminal must return the surface that PresentTerminal
// accepts.
type Display interface {
	InitializeTerminal() (*graphics.Surface, bool)
	PresentTerminal(*graphics.Surface) bool
}

// ScrollDisplay optionally accelerates vertical movement of already-rendered
// terminal pixels. PrepareTerminal is called before rendering and lets a
// retained double-buffered display repair a stale back buffer when scrolling
// stops. ScrollTerminal copies [top+pixels, bottom) upward by pixels into
// [top, bottom-pixels). Returning false selects the portable CPU fallback.
type ScrollDisplay interface {
	PrepareTerminal(surface *graphics.Surface, scrolling bool) bool
	ScrollTerminal(surface *graphics.Surface, top, bottom, pixels int) bool
}

// Pointer is an optional touch or mouse-like input source. Coordinates use the
// same orientation and pixels as the terminal surface.
type Pointer interface {
	InitializePointer() bool
	ReadPointer() (x, y int, pressed, ok bool)
}

// Clock provides short monotonic timing measurements for Stats.
type Clock interface {
	Milliseconds() uint32
	DelayMilliseconds(uint32)
}

// Duration is a cooperative terminal ticker interval measured in
// milliseconds. It deliberately stays small and freestanding-friendly.
type Duration uint32

const (
	// Millisecond is one terminal ticker millisecond.
	Millisecond Duration = 1
	// Second is one thousand terminal ticker milliseconds.
	Second = 1000 * Millisecond
)

// Options configures a terminal. Zero dimensions are derived from the surface
// and font. The default font is the readable 2x built-in font.
type Options struct {
	Columns, Rows  int
	Scrollback     int
	Font           *graphics.Font
	CellWidth      int
	CellHeight     int
	Baseline       int
	Foreground     graphics.Color
	Background     graphics.Color
	Cursor         graphics.Color
	FlushPolicy    FlushPolicy
	Pointer        Pointer
	TouchKeyboard  bool
	KeyboardHeight int
	LocalEcho      bool
	Clock          Clock
}

// Cell is one terminal grid position and its resolved rendition.
type Cell struct {
	Character  byte
	Foreground graphics.Color
	Background graphics.Color
	Bold       bool
	Underline  bool
	Inverse    bool
}

// Stats reports cumulative terminal and rendering work.
type Stats struct {
	Bytes           uint64
	Lines           uint64
	Scrolls         uint64
	Frames          uint64
	RenderMillis    uint64
	LastFrameMillis uint32
}

type terminalError string

func (e terminalError) Error() string { return string(e) }

const (
	errDisplay       terminalError = "terminal display initialization failed"
	errSurface       terminalError = "terminal display returned an invalid surface"
	errPointer       terminalError = "terminal touch keyboard input initialization failed"
	errConfiguration terminalError = "terminal dimensions do not fit the display"
)

var defaultForeground = graphics.RGBA(220, 232, 240, 255)
var defaultBackground = graphics.RGBA(7, 12, 18, 255)
var defaultCursor = graphics.RGBA(80, 200, 130, 255)

// Terminal is a fixed-capacity terminal model, renderer, scrollback buffer,
// and optional touch keyboard. It performs no allocations while processing
// output after construction.
type Terminal struct {
	columns, rows int
	capacity      int
	// Each cell occupies two native 32-bit words. Keeping the hot parser and
	// renderer off 64-bit integer helpers matters on 32-bit microcontrollers.
	cells       []uint32
	first       int
	lineCount   int
	screenStart int
	cursorLine  int
	column      int
	viewOffset  int
	pendingWrap bool

	foreground               graphics.Color
	background               graphics.Color
	defaultForeground        graphics.Color
	defaultBackground        graphics.Color
	bold, underline, inverse bool
	cursorVisible            bool
	savedLine, savedColumn   int

	escapeState      byte
	parameters       [16]int
	parameterPresent [16]bool
	parameterIndex   int
	privateCSI       bool
	utf8Remaining    byte

	display                         Display
	scrollDisplay                   ScrollDisplay
	surface                         *graphics.Surface
	font                            *graphics.Font
	cellWidth, cellHeight, baseline int
	contentHeight                   int
	cursorColor                     graphics.Color
	dirtyStart, dirtyEnd            []int
	pendingScroll                   int
	flushPolicy                     FlushPolicy
	clock                           Clock
	tickDeadline                    uint32
	tickReady                       bool
	stats                           Stats

	pointer                           Pointer
	touchKeyboard                     bool
	keyboardHeight                    int
	keyboardPressed                   int
	keyboardShift                     bool
	keyboardDirty                     bool
	keyboardFullDirty                 bool
	keyboardDirtyKeys                 [42]bool
	pointerDown                       bool
	pointerStartY                     int
	pointerLastY                      int
	localEcho                         bool
	input                             [128]byte
	inputRead, inputWrite, inputCount int
}

// New constructs a model-only terminal with fixed dimensions and scrollback.
// Use Start when output should also be rendered and mirrored from stdout.
func New(columns, rows, scrollback int) *Terminal {
	if columns < 1 {
		columns = 1
	}
	if rows < 1 {
		rows = 1
	}
	if scrollback < 0 {
		scrollback = 0
	}
	t := &Terminal{
		columns: columns, rows: rows, capacity: rows + scrollback,
		defaultForeground: defaultForeground, defaultBackground: defaultBackground,
		foreground: defaultForeground, background: defaultBackground,
		cursorVisible: true, keyboardPressed: -1,
	}
	t.cells = make([]uint32, t.capacity*t.columns*2)
	t.dirtyStart = make([]int, t.rows)
	t.dirtyEnd = make([]int, t.rows)
	t.lineCount = t.rows
	t.clearAllStorage()
	t.markAllDirty()
	return t
}

// Start initializes display, constructs a terminal sized to its surface, and
// makes it the destination for mirrored Renvo standard output.
func Start(display Display, options Options) (*Terminal, error) {
	if display == nil {
		return nil, errDisplay
	}
	surface, ok := display.InitializeTerminal()
	if !ok {
		return nil, errDisplay
	}
	if surface == nil || surface.Width <= 0 || surface.Height <= 0 {
		return nil, errSurface
	}
	font := options.Font
	if font == nil {
		font = graphics.NewBuiltinFont(2)
	}
	cellWidth := options.CellWidth
	if cellWidth <= 0 {
		cellWidth = int(graphics.MeasureText(font, "M").Width + 0.5)
	}
	cellHeight := options.CellHeight
	if cellHeight <= 0 {
		cellHeight = int(font.Metrics.Ascent + font.Metrics.Descent + font.Metrics.LineGap + 0.5)
	}
	baseline := options.Baseline
	if baseline <= 0 {
		baseline = int(font.Metrics.Ascent + 0.5)
	}
	keyboardHeight := 0
	if options.TouchKeyboard {
		keyboardHeight = options.KeyboardHeight
		if keyboardHeight <= 0 {
			keyboardHeight = surface.Height * 36 / 100
		}
		if options.Pointer == nil {
			return nil, errPointer
		}
	}
	contentHeight := surface.Height - keyboardHeight
	columns, rows := options.Columns, options.Rows
	if columns <= 0 {
		columns = surface.Width / cellWidth
	}
	if rows <= 0 {
		rows = contentHeight / cellHeight
	}
	if columns < 1 || rows < 1 || columns*cellWidth > surface.Width || rows*cellHeight > contentHeight {
		return nil, errConfiguration
	}
	t := New(columns, rows, options.Scrollback)
	t.display, t.surface, t.font = display, surface, font
	if scrollDisplay, ok := display.(ScrollDisplay); ok {
		t.scrollDisplay = scrollDisplay
	}
	t.cellWidth, t.cellHeight, t.baseline = cellWidth, cellHeight, baseline
	t.contentHeight = contentHeight
	t.flushPolicy, t.clock = options.FlushPolicy, options.Clock
	t.pointer, t.touchKeyboard = options.Pointer, options.TouchKeyboard
	t.keyboardHeight, t.localEcho = keyboardHeight, options.LocalEcho
	if t.touchKeyboard {
		t.keyboardDirty = true
		t.keyboardFullDirty = true
	}
	if options.Foreground.A != 0 {
		t.defaultForeground = options.Foreground
		t.foreground = options.Foreground
	}
	if options.Background.A != 0 {
		t.defaultBackground = options.Background
		t.background = options.Background
	}
	t.cursorColor = options.Cursor
	if t.cursorColor.A == 0 {
		t.cursorColor = defaultCursor
	}
	active = t
	if !t.Flush() {
		active = nil
		return nil, errDisplay
	}
	if t.pointer != nil && !t.pointer.InitializePointer() {
		active = nil
		return nil, errPointer
	}
	return t, nil
}

var active *Terminal

// Active returns the terminal currently mirroring Renvo standard output.
func Active() *Terminal { return active }

// Tick services the active terminal and paces a cooperative application loop.
// It returns false if no terminal is active or display/input servicing fails,
// allowing the natural form:
//
//	for terminal.Tick(16 * terminal.Millisecond) {
//		// application work
//	}
//
// Deadlines advance from the prior deadline, so application work does not add
// drift. A late frame begins a fresh interval rather than trying to catch up.
func Tick(interval Duration) bool {
	if active == nil {
		return false
	}
	return active.Tick(interval)
}

// Tick services t and waits until its next cooperative interval.
func (t *Terminal) Tick(interval Duration) bool {
	if !t.Poll() {
		return false
	}
	if t.clock == nil || interval == 0 {
		return true
	}
	now := t.clock.Milliseconds()
	if !t.tickReady {
		t.tickDeadline = now + uint32(interval)
		t.tickReady = true
	}
	remaining := int32(t.tickDeadline - now)
	if remaining > 0 {
		t.clock.DelayMilliseconds(uint32(remaining))
		now = t.clock.Milliseconds()
	}
	next := t.tickDeadline + uint32(interval)
	if int32(next-now) <= 0 {
		next = now + uint32(interval)
	}
	t.tickDeadline = next
	return true
}

// Stop disables standard-output mirroring when t is the active terminal.
func (t *Terminal) Stop() {
	if active == t {
		active = nil
	}
}

// renvo_runtime_PrintMirror is called by the Renvo backend before it emits a
// standard-output fragment. Its reserved name is intentionally kept by the
// frontend linker.
func renvo_runtime_PrintMirror(text string) {
	if active == nil {
		return
	}
	active.WriteString(text)
	flush := active.flushPolicy == FlushEveryWrite
	if active.flushPolicy == FlushLine {
		for i := 0; i < len(text); i++ {
			if text[i] == '\n' || text[i] == '\r' || text[i] == 7 {
				flush = true
				break
			}
		}
	}
	if flush {
		active.Flush()
	}
}

// Columns returns the visible terminal width in cells.
func (t *Terminal) Columns() int { return t.columns }

// Rows returns the visible terminal height in cells.
func (t *Terminal) Rows() int { return t.rows }

// Cursor returns the cursor row and column in the live screen.
func (t *Terminal) Cursor() (row, column int) {
	return t.cursorLine - t.screenStart, t.column
}

// ViewOffset returns the number of history lines above the live screen.
func (t *Terminal) ViewOffset() int { return t.viewOffset }

// Statistics returns a snapshot of cumulative terminal work.
func (t *Terminal) Statistics() Stats { return t.stats }

func (t *Terminal) physicalLine(logical int) int {
	return (t.first + logical) % t.capacity
}

func packColor(color graphics.Color) uint32 {
	return uint32(color.R) | uint32(color.G)<<8 | uint32(color.B)<<16
}

func unpackColor(value uint32) graphics.Color {
	return graphics.RGBA(byte(value), byte(value>>8), byte(value>>16), 255)
}

func packCell(cell Cell) (uint32, uint32) {
	first := uint32(cell.Character) | packColor(cell.Foreground)<<8
	second := packColor(cell.Background)
	if cell.Bold {
		second |= uint32(1) << 24
	}
	if cell.Underline {
		second |= uint32(1) << 25
	}
	if cell.Inverse {
		second |= uint32(1) << 26
	}
	return first, second
}

func unpackCell(first, second uint32) Cell {
	return Cell{
		Character: byte(first), Foreground: unpackColor(first >> 8),
		Background: unpackColor(second), Bold: second&(uint32(1)<<24) != 0,
		Underline: second&(uint32(1)<<25) != 0, Inverse: second&(uint32(1)<<26) != 0,
	}
}

func (t *Terminal) cellIndex(logical, column int) int {
	return (t.physicalLine(logical)*t.columns + column) * 2
}

func (t *Terminal) cell(logical, column int) Cell {
	index := t.cellIndex(logical, column)
	return unpackCell(t.cells[index], t.cells[index+1])
}

func (t *Terminal) setCell(logical, column int, cell Cell) {
	index := t.cellIndex(logical, column)
	first, second := packCell(cell)
	if t.cells[index] == first && t.cells[index+1] == second {
		return
	}
	t.cells[index], t.cells[index+1] = first, second
	t.markLogicalRange(logical, column, column+1)
}

func (t *Terminal) blankCell() (uint32, uint32) {
	return packCell(Cell{Character: ' ', Foreground: t.defaultForeground, Background: t.defaultBackground})
}

func (t *Terminal) setBlank(index int) {
	t.cells[index], t.cells[index+1] = t.blankCell()
}

func (t *Terminal) setBlankCell(logical, column int) {
	index := t.cellIndex(logical, column)
	first, second := t.blankCell()
	if t.cells[index] == first && t.cells[index+1] == second {
		return
	}
	t.cells[index], t.cells[index+1] = first, second
	t.markLogicalRange(logical, column, column+1)
}

func (t *Terminal) clearStorageLine(logical int) {
	first, second := t.blankCell()
	physical := t.physicalLine(logical) * t.columns * 2
	for column := 0; column < t.columns; column++ {
		index := physical + column*2
		t.cells[index], t.cells[index+1] = first, second
	}
}

func (t *Terminal) clearAllStorage() {
	first, second := t.blankCell()
	for i := 0; i < len(t.cells); i += 2 {
		t.cells[i], t.cells[i+1] = first, second
	}
}

func (t *Terminal) displayStart() int {
	start := t.screenStart - t.viewOffset
	if start < 0 {
		start = 0
	}
	return start
}

func (t *Terminal) markAllDirty() {
	for row := 0; row < len(t.dirtyStart); row++ {
		t.dirtyStart[row] = 0
		t.dirtyEnd[row] = t.columns
	}
}

func (t *Terminal) markLogicalRange(logical, start, end int) {
	row := logical - t.displayStart()
	if row < 0 || row >= len(t.dirtyStart) || start >= end {
		return
	}
	if start < 0 {
		start = 0
	}
	if end > t.columns {
		end = t.columns
	}
	if t.dirtyStart[row] >= t.dirtyEnd[row] || start < t.dirtyStart[row] {
		t.dirtyStart[row] = start
	}
	if end > t.dirtyEnd[row] {
		t.dirtyEnd[row] = end
	}
}

func (t *Terminal) markLogicalDirty(logical int) {
	t.markLogicalRange(logical, 0, t.columns)
}

func (t *Terminal) markCursorDirty() {
	if t.cursorVisible {
		t.markLogicalRange(t.cursorLine, t.column, t.column+1)
	}
}

func (t *Terminal) appendLine() {
	wasLive := t.viewOffset == 0
	if t.lineCount < t.capacity {
		t.lineCount++
	} else {
		t.first = (t.first + 1) % t.capacity
		t.cursorLine--
		t.screenStart--
		if t.viewOffset > 0 {
			t.viewOffset--
		}
	}
	if t.viewOffset > 0 {
		t.viewOffset++
	}
	t.screenStart++
	t.cursorLine++
	t.clearStorageLine(t.lineCount - 1)
	t.stats.Scrolls++
	if wasLive {
		// Preserve dirty work on rows that move upward and repaint only the new
		// bottom row. Flush moves the already-rasterized pixels by the same count.
		for row := 0; row < t.rows-1; row++ {
			t.dirtyStart[row] = t.dirtyStart[row+1]
			t.dirtyEnd[row] = t.dirtyEnd[row+1]
		}
		t.dirtyStart[t.rows-1] = 0
		t.dirtyEnd[t.rows-1] = t.columns
		t.pendingScroll++
		if t.pendingScroll >= t.rows {
			t.pendingScroll = 0
			t.markAllDirty()
		}
	} else {
		// New output normally leaves a history viewport stationary. A complete
		// repaint also covers the capacity-eviction case at the oldest line.
		t.markAllDirty()
	}
}

func (t *Terminal) lineFeed() {
	t.markCursorDirty()
	if t.cursorLine < t.screenStart+t.rows-1 {
		t.cursorLine++
	} else {
		t.appendLine()
	}
	t.pendingWrap = false
	t.stats.Lines++
	t.markCursorDirty()
}

func (t *Terminal) carriageReturn() {
	t.markCursorDirty()
	t.column = 0
	t.pendingWrap = false
	t.markCursorDirty()
}

func (t *Terminal) put(value byte) {
	if value < 0x20 || value == 0x7f {
		return
	}
	if t.pendingWrap {
		t.column = 0
		t.lineFeed()
	}
	t.markCursorDirty()
	t.setCell(t.cursorLine, t.column, Cell{
		Character: value, Foreground: t.foreground, Background: t.background,
		Bold: t.bold, Underline: t.underline, Inverse: t.inverse,
	})
	if t.column == t.columns-1 {
		t.pendingWrap = true
	} else {
		t.column++
	}
	t.markCursorDirty()
}

func (t *Terminal) backspace() {
	t.markCursorDirty()
	if t.pendingWrap {
		t.pendingWrap = false
	} else if t.column > 0 {
		t.column--
	}
	t.setBlankCell(t.cursorLine, t.column)
	t.markCursorDirty()
}

func (t *Terminal) tab() {
	stop := (t.column + 8) &^ 7
	if stop >= t.columns {
		stop = t.columns - 1
	}
	for t.column < stop {
		t.put(' ')
	}
}

func (t *Terminal) move(row, column int) {
	t.markCursorDirty()
	if row < 0 {
		row = 0
	} else if row >= t.rows {
		row = t.rows - 1
	}
	if column < 0 {
		column = 0
	} else if column >= t.columns {
		column = t.columns - 1
	}
	t.cursorLine = t.screenStart + row
	t.column = column
	t.pendingWrap = false
	t.markCursorDirty()
}

// CellAt returns a copy of one cell in the currently displayed viewport.
func (t *Terminal) CellAt(row, column int) (Cell, bool) {
	if row < 0 || row >= t.rows || column < 0 || column >= t.columns {
		return Cell{}, false
	}
	return t.cell(t.displayStart()+row, column), true
}

// Scroll moves the displayed viewport into history. Positive lines move up;
// negative lines move back toward live output.
func (t *Terminal) Scroll(lines int) {
	t.pendingScroll = 0
	t.viewOffset += lines
	if t.viewOffset < 0 {
		t.viewOffset = 0
	}
	if t.viewOffset > t.screenStart {
		t.viewOffset = t.screenStart
	}
	t.markAllDirty()
}

// ScrollToBottom returns the viewport to live output.
func (t *Terminal) ScrollToBottom() {
	t.pendingScroll = 0
	t.viewOffset = 0
	t.markAllDirty()
}

// Reset clears screen and scrollback and restores default terminal attributes.
func (t *Terminal) Reset() {
	t.pendingScroll = 0
	t.first, t.lineCount, t.screenStart, t.cursorLine = 0, t.rows, 0, 0
	t.column, t.viewOffset = 0, 0
	t.pendingWrap = false
	t.foreground, t.background = t.defaultForeground, t.defaultBackground
	t.bold, t.underline, t.inverse = false, false, false
	t.cursorVisible = true
	t.escapeState, t.parameterIndex = 0, 0
	t.parameters = [16]int{}
	t.parameterPresent = [16]bool{}
	t.clearAllStorage()
	t.markAllDirty()
}

// Write implements io.Writer and feeds bytes into the terminal parser.
func (t *Terminal) Write(data []byte) (int, error) {
	for i := 0; i < len(data); i++ {
		t.WriteByte(data[i])
	}
	return len(data), nil
}

// WriteString feeds a string into the terminal parser without conversion.
func (t *Terminal) WriteString(text string) (int, error) {
	for i := 0; i < len(text); i++ {
		t.WriteByte(text[i])
	}
	return len(text), nil
}

// WriteByte feeds one byte into the terminal parser.
func (t *Terminal) WriteByte(value byte) {
	t.stats.Bytes++
	if t.escapeState != 0 {
		t.writeEscape(value)
		return
	}
	if t.utf8Remaining != 0 {
		if value&0xc0 == 0x80 {
			t.utf8Remaining--
			if t.utf8Remaining == 0 {
				t.put('?')
			}
			return
		}
		t.utf8Remaining = 0
		t.put('?')
	}
	if value >= 0x80 {
		if value&0xe0 == 0xc0 {
			t.utf8Remaining = 1
		} else if value&0xf0 == 0xe0 {
			t.utf8Remaining = 2
		} else if value&0xf8 == 0xf0 {
			t.utf8Remaining = 3
		} else {
			t.put('?')
		}
		return
	}
	if value < 0x20 || value == 0x7f {
		t.writeControl(value)
		return
	}
	t.put(value)
}

func (t *Terminal) writeControl(value byte) {
	switch value {
	case 0:
	case 7:
		// A visual or audible bell can be layered on by the board adapter. The
		// terminal itself has no bell state to repaint.
	case 8, 0x7f:
		t.backspace()
	case 9:
		t.tab()
	case 10, 11:
		t.lineFeed()
	case 12:
		t.Reset()
	case 13:
		t.carriageReturn()
	case 27:
		t.escapeState = 1
	default:
		t.put('^')
		t.put(value + '@')
	}
}

// Read drains bytes produced by the optional touch keyboard. It is
// non-blocking and returns zero when no input is queued.
func (t *Terminal) Read(data []byte) (int, error) {
	count := len(data)
	if count > t.inputCount {
		count = t.inputCount
	}
	for i := 0; i < count; i++ {
		data[i] = t.input[t.inputRead]
		t.inputRead = (t.inputRead + 1) % len(t.input)
	}
	t.inputCount -= count
	return count, nil
}

// SendInput queues one byte from a physical or application-provided keyboard.
// When LocalEcho is enabled it is also written to the terminal display.
func (t *Terminal) SendInput(value byte) { t.queueInput(value) }

func (t *Terminal) queueInput(value byte) {
	if t.inputCount == len(t.input) {
		return
	}
	t.input[t.inputWrite] = value
	t.inputWrite = (t.inputWrite + 1) % len(t.input)
	t.inputCount++
	if t.localEcho {
		t.WriteByte(value)
	}
}
