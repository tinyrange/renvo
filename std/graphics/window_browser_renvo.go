//go:build renvo && browser && wasm32

package graphics

var browserClipboard string
var browserEventBuffer [4136]byte
var browserFrameHeader [20]byte
var browserNextWindowID = 1

func NewWindow(options WindowOptions) *Window {
	clearLastWindowError()
	if options.Width <= 0 || options.Height <= 0 {
		setLastWindowError("window dimensions must be positive", 0)
		return nil
	}
	w := &Window{width: options.Width, height: options.Height, active: true, shown: !options.Hidden}
	w.native = browserNextWindowID
	browserNextWindowID++
	w.surface = NewSurface(w.width, w.height)
	browserCreateWindow(w, options.Title)
	w.queue(Event{Type: EventWindowExpose, Dirty: R(0, 0, Scalar(w.width), Scalar(w.height))})
	return w
}

func (w *Window) Poll() (Event, bool) { return w.nextQueuedEvent() }

func (w *Window) Wait() (Event, bool) {
	if event, ok := w.Poll(); ok {
		return event, true
	}
	buf := browserEventBuffer[:]
	n := read(0, buf, -1)
	if n < 40 {
		return Event{}, false
	}
	event := Event{
		Type:      EventType(browserInt32(buf, 0)),
		X:         Scalar(browserInt32(buf, 4)),
		Y:         Scalar(browserInt32(buf, 8)),
		WheelX:    Scalar(browserInt32(buf, 12)),
		WheelY:    Scalar(browserInt32(buf, 16)),
		Key:       Key(browserInt32(buf, 20)),
		Button:    browserInt32(buf, 24),
		Modifiers: Modifiers(browserInt32(buf, 28)),
		Repeat:    browserInt32(buf, 32) != 0,
	}
	event.TimerID = browserInt32(buf, 20)
	textLen := browserInt32(buf, 36)
	if textLen > n-40 {
		textLen = n - 40
	}
	if textLen > 0 {
		event.Text = string(buf[40 : 40+textLen])
	}
	if event.Type == EventWindowResize {
		width, height := int(event.X), int(event.Y)
		if width > 0 && height > 0 && (width != w.width || height != w.height) {
			w.width, w.height = width, height
			w.surface.Resize(width, height)
			event.Dirty = R(0, 0, Scalar(width), Scalar(height))
		}
	}
	return event, true
}

func browserInt32(data []byte, at int) int {
	return int(data[at]) | int(data[at+1])<<8 | int(data[at+2])<<16 | int(data[at+3])<<24
}

func browserPut32(data []byte, at, value int) {
	data[at] = byte(value)
	data[at+1] = byte(value >> 8)
	data[at+2] = byte(value >> 16)
	data[at+3] = byte(value >> 24)
}

func (w *Window) Present() bool {
	if w == nil || w.closed || w.surface == nil {
		return false
	}
	pixels := w.surface.Pixels
	header := browserFrameHeader[:20]
	copy(header, []byte("RNVF"))
	browserPut32(header, 4, w.native)
	browserPut32(header, 8, w.surface.Width)
	browserPut32(header, 12, w.surface.Height)
	browserPut32(header, 16, len(pixels))
	if write(3, header, -1) != len(header) || write(3, pixels, -1) != len(pixels) {
		return false
	}
	w.surface.ResetDirty()
	return true
}

func (w *Window) ReadPixels() *Image {
	if w == nil || w.closed || w.surface == nil {
		return nil
	}
	return NewImage(w.surface.Width, w.surface.Height, w.surface.Pixels)
}

func browserCreateWindow(w *Window, title string) {
	header := make([]byte, 20)
	copy(header, []byte("RNVW"))
	browserPut32(header, 4, w.native)
	browserPut32(header, 8, w.width)
	browserPut32(header, 12, w.height)
	browserPut32(header, 16, len(title))
	write(3, header, -1)
	write(3, []byte(title), -1)
}

func (w *Window) SetTitle(title string) bool {
	if w == nil || w.closed {
		return false
	}
	header := make([]byte, 12)
	copy(header, []byte("RNVT"))
	browserPut32(header, 4, w.native)
	browserPut32(header, 8, len(title))
	write(3, header, -1)
	write(3, []byte(title), -1)
	return true
}
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
	w.queue(Event{Type: EventWindowResize, Dirty: R(0, 0, Scalar(width), Scalar(height))})
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
	if w == nil || w.closed || id == 0 || seconds < 0 {
		return false
	}
	milliseconds := int(seconds * 1000)
	if seconds > 0 && milliseconds < 1 {
		milliseconds = 1
	}
	header := browserFrameHeader[:16]
	copy(header, []byte("RNVM"))
	browserPut32(header, 4, w.native)
	browserPut32(header, 8, id)
	browserPut32(header, 12, milliseconds)
	return write(3, header, -1) == len(header)
}
func (w *Window) CancelTimer(id int) {
	if w == nil || w.closed || id == 0 {
		return
	}
	header := browserFrameHeader[:12]
	copy(header, []byte("RNVC"))
	browserPut32(header, 4, w.native)
	browserPut32(header, 8, id)
	write(3, header, -1)
}
func SetClipboardText(text string) bool { browserClipboard = text; return true }
func ClipboardText() (string, bool)     { return browserClipboard, true }
func (w *Window) Close() {
	if w != nil {
		if !w.closed {
			header := make([]byte, 8)
			copy(header, []byte("RNVX"))
			browserPut32(header, 4, w.native)
			write(3, header, -1)
		}
		w.closed = true
		w.active = false
	}
}
