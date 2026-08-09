//go:build renvo && android && arm64

package graphics

const androidWindowFormatRGBA8888 = 1

var androidWindow *Window
var androidClipboard string

// The Android CompilerJIT target handles this private self-import by wiring the
// NativeActivity callback table to the functions below.
// renvo:linkstatic librenvo.so,renvoAndroidInstallCallbacks
func androidInstallNativeActivityCallbacks() int { return 0 }

// renvo:linkstatic libandroid.so,ANativeWindow_setBuffersGeometry
func androidSetBuffersGeometry(window, width, height, format int) int { return -1 }

// renvo:linkstatic libandroid.so,ANativeWindow_lock
func androidLockWindow(window int, buffer *byte, dirty *byte) int { return -1 }

// renvo:linkstatic libandroid.so,ANativeWindow_unlockAndPost
func androidUnlockAndPost(window int) int { return -1 }

// renvo:linkstatic libc.so,memcpy
func androidCopyMemory(destination int, source *byte, size int) int { return 0 }

func androidUint32(data []byte, at int) int {
	return int(data[at]) |
		int(data[at+1])<<8 |
		int(data[at+2])<<16 |
		int(data[at+3])<<24
}

func androidPointer(data []byte, at int) int {
	return androidUint32(data, at) | androidUint32(data, at+4)<<32
}

func androidRetainCallbacks() {
	// These ordinary calls keep the callback functions reachable. The condition
	// is never true for a live window, but cannot be decided at compile time.
	if androidWindow != nil && androidWindow.native == -1 {
		androidOnNativeWindowCreated(0, 0)
		androidOnNativeWindowResized(0, 0)
		androidOnNativeWindowRedrawNeeded(0, 0)
		androidOnNativeWindowDestroyed(0, 0)
	}
}

func NewWindow(options WindowOptions) *Window {
	clearLastWindowError()
	if options.Width <= 0 || options.Height <= 0 {
		setLastWindowError("window dimensions must be positive", 0)
		return nil
	}
	if androidWindow != nil && !androidWindow.closed {
		setLastWindowError("Android NativeActivity supports one window", 0)
		return nil
	}
	w := &Window{
		width: options.Width, height: options.Height,
		active: true, shown: !options.Hidden,
	}
	w.surface = NewSurface(w.width, w.height)
	androidWindow = w
	androidRetainCallbacks()
	installed := androidInstallNativeActivityCallbacks()
	if installed == 0 {
		androidWindow = nil
		setLastWindowError("NativeActivity callback installation failed", 0)
		return nil
	}
	return w
}

func androidAttachWindow(native int) {
	w := androidWindow
	if w == nil || w.closed || native == 0 {
		return
	}
	changed := w.native != native
	w.native = native
	if changed && androidSetBuffersGeometry(native, w.width, w.height,
		androidWindowFormatRGBA8888) != 0 {
		return
	}
	w.Present()
}

// Renvo's AArch64 internal call convention assigns source parameters to
// registers from right to left. Declare callback parameters in reverse C ABI
// order so Android's (activity, window) x0/x1 pair reaches the right names.
func androidOnNativeWindowCreated(native, activity int) {
	androidAttachWindow(native)
}

func androidOnNativeWindowResized(native, activity int) {
	androidAttachWindow(native)
}

func androidOnNativeWindowRedrawNeeded(native, activity int) {
	w := androidWindow
	if w == nil || w.closed {
		return
	}
	if w.native != native {
		androidAttachWindow(native)
		return
	}
	w.Present()
}

func androidOnNativeWindowDestroyed(native, activity int) {
	w := androidWindow
	if w != nil && w.native == native {
		w.native = 0
	}
}

func (w *Window) Present() bool {
	if w == nil || w.closed || w.surface == nil {
		return false
	}
	if w.native == 0 {
		return true
	}
	buffer := make([]byte, 48)
	if androidLockWindow(w.native, &buffer[0], nil) != 0 {
		return false
	}
	width := androidUint32(buffer, 0)
	height := androidUint32(buffer, 4)
	stride := androidUint32(buffer, 8)
	bits := androidPointer(buffer, 16)
	copyWidth := w.surface.Width
	if width < copyWidth {
		copyWidth = width
	}
	copyHeight := w.surface.Height
	if height < copyHeight {
		copyHeight = height
	}
	if bits == 0 || stride < copyWidth || copyWidth <= 0 || copyHeight <= 0 {
		androidUnlockAndPost(w.native)
		return false
	}
	rowBytes := copyWidth * 4
	for y := 0; y < copyHeight; y++ {
		androidCopyMemory(bits+y*stride*4,
			&w.surface.Pixels[y*w.surface.Stride], rowBytes)
	}
	if androidUnlockAndPost(w.native) != 0 {
		return false
	}
	w.surface.ResetDirty()
	return true
}

func (w *Window) Poll() (Event, bool) { return w.nextQueuedEvent() }
func (w *Window) Wait() (Event, bool) { return w.Poll() }

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
	if w.native != 0 {
		if androidSetBuffersGeometry(w.native, width, height,
			androidWindowFormatRGBA8888) != 0 {
			return false
		}
	}
	return w.Present()
}
func (w *Window) RequestRepaint(rect Rect) {
	if w != nil && !w.closed {
		w.Present()
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
func SetClipboardText(text string) bool { androidClipboard = text; return true }
func ClipboardText() (string, bool)     { return androidClipboard, true }
func (w *Window) Close() {
	if w == nil {
		return
	}
	w.closed = true
	w.active = false
	w.native = 0
	if androidWindow == w {
		androidWindow = nil
	}
}
