//go:build m5tab5

package board

import (
	"renvo.dev/device/terminal"
	"renvo.dev/std/graphics"
	"unsafe"
)

// Screen adapts the Tab5 portrait framebuffer to display-oriented packages.
type Screen struct {
	ready                   bool
	surface                 *graphics.Surface
	scrollPrepared          bool
	scrollBackStale         bool
	scrollTop, scrollBottom int
}

// Display is the Tab5's 720 by 1280 portrait display.
var Display = Screen{}

// Console is the board's active terminal after StartTerminal succeeds.
var Console *terminal.Terminal

func defaultTerminalOptions() terminal.Options {
	return terminal.Options{
		Scrollback:     256,
		Font:           graphics.NewBuiltinFont(3),
		CellWidth:      18,
		CellHeight:     30,
		Baseline:       21,
		Pointer:        &Touch,
		TouchKeyboard:  true,
		KeyboardHeight: 450,
		FlushPolicy:    terminal.FlushManual,
		Clock:          &Display,
	}
}

// StartTerminal initializes a large-font color terminal with scrollback and
// the Tab5 touch keyboard, then assigns it to Console and stdout mirroring.
func StartTerminal() error {
	return StartTerminalWithOptions(defaultTerminalOptions())
}

// StartTerminalWithOptions initializes Console with caller-supplied terminal
// options. StartTerminal supplies the recommended Tab5 defaults.
func StartTerminalWithOptions(options terminal.Options) error {
	console, err := terminal.Start(&Display, options)
	if err != nil {
		return err
	}
	Console = console
	return nil
}

// TickTerminal services Console at the Tab5 display's 60 Hz cadence. It is
// intended for the condition in a cooperative application loop.
func TickTerminal() bool {
	return TickTerminalEvery(terminal.Second / 60)
}

// TickTerminalEvery services Console at a caller-selected interval.
func TickTerminalEvery(interval terminal.Duration) bool {
	return Console != nil && Console.Tick(interval)
}

// InitializeTerminal initializes the framebuffer and returns its RGB565 back
// surface. Repeated calls reuse the same surface.
func (s *Screen) InitializeTerminal() (*graphics.Surface, bool) {
	if s.ready {
		return s.surface, s.surface != nil
	}
	if !InitFramebuffer() {
		return nil, false
	}
	s.surface = NewPortraitSurface()
	s.ready = s.surface != nil
	return s.surface, s.ready
}

// PresentTerminal publishes terminal damage at a complete frame boundary.
func (s *Screen) PresentTerminal(surface *graphics.Surface) bool {
	if !s.scrollPrepared {
		ok := PresentPortrait(surface)
		if ok {
			s.scrollBackStale = false
		}
		return ok
	}
	top, bottom := s.scrollTop, s.scrollBottom
	s.scrollPrepared = false
	ok := presentPortrait(surface, true, top, bottom)
	if ok {
		// Content in the old front buffer is intentionally left one generation
		// behind. The next scroll overwrites it directly; a non-scroll frame
		// repairs it in PrepareTerminal before any incremental CPU drawing.
		s.scrollBackStale = true
	}
	return ok
}

// PrepareTerminal repairs retained scroll content only when the next frame is
// not another scroll. Continuous scrolling therefore avoids the redundant
// post-flip full-content DMA copy.
func (s *Screen) PrepareTerminal(surface *graphics.Surface, scrolling bool) bool {
	if !s.scrollBackStale || scrolling {
		return true
	}
	top, bottom := s.scrollTop, s.scrollBottom
	if !s.copyTerminalRegion(surface, top, top, bottom-top) {
		return false
	}
	s.scrollBackStale = false
	return true
}

func (s *Screen) copyTerminalRegion(surface *graphics.Surface, sourceY, destinationY, height int) bool {
	if surface == nil || surface.Format != graphics.PixelRGB565 ||
		surface.Width != DisplayWidth || surface.Height != DisplayHeight ||
		len(surface.Pixels) < framebufferSize || !scanoutStarted ||
		sourceY < 0 || destinationY < 0 || height <= 0 ||
		sourceY+height > DisplayHeight || destinationY+height > DisplayHeight {
		return false
	}
	destination := uintptr(unsafe.Pointer(&surface.Pixels[0]))
	if destination != backFramebuffer {
		return false
	}
	destinationStart := destinationY * surface.Stride
	destinationSize := height * surface.Stride
	writeBackInvalidate(destination+uintptr(destinationStart), destinationSize)
	if !copyRectDMA2DAt(
		frontFramebuffer, destination,
		0, sourceY, 0, destinationY, DisplayWidth, height,
	) {
		return false
	}
	invalidate(destination+uintptr(destinationStart), destinationSize)
	return true
}

// ScrollTerminal uses DMA2D to move portrait framebuffer rows from the current
// front buffer into the back buffer. The buffers never overlap, avoiding the
// large CPU/PSRAM memmove otherwise required by terminal scrolling.
func (s *Screen) ScrollTerminal(surface *graphics.Surface, top, bottom, pixels int) bool {
	if surface == nil || surface.Format != graphics.PixelRGB565 ||
		surface.Width != DisplayWidth || surface.Height != DisplayHeight ||
		len(surface.Pixels) < framebufferSize || !scanoutStarted ||
		top < 0 || bottom > DisplayHeight || top >= bottom ||
		pixels <= 0 || pixels >= bottom-top {
		return false
	}
	if !s.copyTerminalRegion(surface, top+pixels, top, bottom-top-pixels) {
		return false
	}
	s.scrollTop, s.scrollBottom = top, bottom
	s.scrollPrepared = true
	s.scrollBackStale = false
	return true
}

// Milliseconds returns the Tab5 monotonic system timer in milliseconds.
func (*Screen) Milliseconds() uint32 { return Milliseconds() }

// DelayMilliseconds provides cooperative terminal loop pacing.
func (*Screen) DelayMilliseconds(milliseconds uint32) {
	started := Milliseconds()
	for Milliseconds()-started < milliseconds {
		Refresh()
	}
}

// Touchscreen adapts the first active ST7121 contact to pointer input. It
// retains the last coordinate for the release transition.
type Touchscreen struct {
	ready      bool
	primary    int
	x, y       int
	pressed    bool
	lastReport int
	points     [10]TouchPoint
}

// Touch is the Tab5 integrated touch panel.
var Touch = Touchscreen{primary: -1, lastReport: -1}

// InitializePointer initializes the touch controller once.
func (t *Touchscreen) InitializePointer() bool {
	if !t.ready {
		t.ready = InitTouch()
	}
	return t.ready
}

// ReadPointer returns the primary contact in portrait display coordinates.
func (t *Touchscreen) ReadPointer() (x, y int, pressed, ok bool) {
	if !t.InitializePointer() {
		return 0, 0, false, false
	}
	count, ok := ReadTouches(t.points[:])
	if !ok {
		return t.x, t.y, t.pressed, false
	}
	report := TouchLastReportStats().Reports
	if report == t.lastReport {
		return t.x, t.y, t.pressed, true
	}
	t.lastReport = report
	primaryIndex := -1
	for index := 0; index < count; index++ {
		if t.points[index].ID == t.primary {
			primaryIndex = index
			break
		}
	}
	if t.primary >= 0 && primaryIndex < 0 {
		t.primary = -1
		t.pressed = false
		return t.x, t.y, false, true
	}
	if primaryIndex < 0 && count > 0 {
		t.primary = t.points[0].ID
		primaryIndex = 0
	}
	if primaryIndex >= 0 {
		t.x, t.y = PortraitPoint(t.points[primaryIndex])
		t.pressed = true
	}
	return t.x, t.y, t.pressed, true
}
