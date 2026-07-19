//go:build !renvo

package graphics

var hostClipboard string

// The host implementation is deliberately headless. It makes the portable
// renderer directly testable with the Go toolchain; Renvo target files own native
// windows and presentation.
func NewWindow(options WindowOptions) *Window {
	clearLastWindowError()
	if options.Width <= 0 || options.Height <= 0 {
		setLastWindowError("window dimensions must be positive", 0)
		return nil
	}
	w := &Window{width: options.Width, height: options.Height, active: true, shown: !options.Hidden}
	w.surface = NewSurface(w.width, w.height)
	w.queue(Event{Type: EventWindowExpose, Dirty: R(0, 0, Scalar(w.width), Scalar(w.height))})
	return w
}

func (w *Window) Poll() (Event, bool) { return w.nextQueuedEvent() }
func (w *Window) Wait() (Event, bool) { return w.Poll() }
func (w *Window) Present() bool {
	if w == nil || w.closed {
		return false
	}
	w.surface.ResetDirty()
	return true
}

// ReadPixels captures the current window contents as a top-down RGBA8 image.
// The headless host backend copies its software surface.
func (w *Window) ReadPixels() *Image {
	if w == nil || w.closed || w.surface == nil {
		return nil
	}
	return NewImage(w.surface.Width, w.surface.Height, w.surface.Pixels)
}
func (w *Window) SetTitle(title string) bool { return w != nil && !w.closed }
func (w *Window) Show() bool {
	if w == nil || w.closed {
		return false
	}
	w.shown = true
	return true
}
func (w *Window) Hide() bool {
	if w == nil || w.closed {
		return false
	}
	w.shown = false
	return true
}
func (w *Window) SetSize(width, height int) bool {
	if w == nil || w.closed || width <= 0 || height <= 0 {
		return false
	}
	w.width, w.height = width, height
	w.surface.Resize(width, height)
	dirty := R(0, 0, Scalar(width), Scalar(height))
	w.queue(Event{Type: EventWindowResize, Dirty: dirty})
	w.queue(Event{Type: EventWindowExpose, Dirty: dirty})
	return true
}
func (w *Window) RequestRepaint(rect Rect) {
	if w != nil && !w.closed {
		w.queue(Event{Type: EventWindowExpose, Dirty: rect})
	}
}
func (w *Window) SetCursor(cursor Cursor) bool {
	if w == nil || w.closed {
		return false
	}
	w.cursor = cursor
	return true
}
func (w *Window) SetPointerCapture(captured bool) bool {
	if w == nil || w.closed {
		return false
	}
	w.captured = captured
	return true
}
func (w *Window) SetTimer(id int, seconds Scalar) bool {
	return w != nil && !w.closed && id != 0 && seconds >= 0
}
func (w *Window) CancelTimer(id int)    {}
func SetClipboardText(text string) bool { hostClipboard = text; return true }
func ClipboardText() (string, bool)     { return hostClipboard, true }
func (w *Window) Close() {
	if w != nil {
		w.closed = true
		w.active = false
	}
}
