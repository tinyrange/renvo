//go:build renvo && android && arm64

package graphics

const androidWindowFormatRGBA8888 = 1
const androidInputEventKey = 1
const androidInputEventMotion = 2
const androidKeyActionDown = 0
const androidKeyActionUp = 1
const androidMetaShift = 1
const androidMetaAlt = 2
const androidMetaControl = 0x1000
const androidMetaCommand = 0x10000
const androidMotionActionMask = 255
const androidMotionPointerIndexMask = 255
const androidMotionPointerIndexShift = 8
const androidMotionActionDown = 0
const androidMotionActionUp = 1
const androidMotionActionMove = 2
const androidMotionActionCancel = 3
const androidMotionActionPointerDown = 5
const androidMotionActionPointerUp = 6
const androidLooperInput = 1
const androidNoPointer = -1
const androidInputBatchLimit = 64

var androidWindow *Window
var androidClipboard string
var androidInputQueue int
var androidViewportWidth int
var androidPhysicalWidth int
var androidDeviceScale Scalar = 1.0
var androidNativeBuffer = make([]byte, 48)
var androidActivePointer = androidNoPointer

// The Android target only materializes runtime-owned pointers. Callback-table
// layout and lifecycle policy remain in this Go client.
// renvo:linkstatic librenvo.so,renvoAndroidActivityCallbacks
func androidNativeActivityCallbacks() int { return 0 }

// renvo:linkstatic librenvo.so,renvoAndroidWindowCreatedCallback
func androidWindowCreatedCallback() int { return 0 }

// renvo:linkstatic librenvo.so,renvoAndroidWindowResizedCallback
func androidWindowResizedCallback() int { return 0 }

// renvo:linkstatic librenvo.so,renvoAndroidWindowRedrawCallback
func androidWindowRedrawCallback() int { return 0 }

// renvo:linkstatic librenvo.so,renvoAndroidWindowDestroyedCallback
func androidWindowDestroyedCallback() int { return 0 }

// renvo:linkstatic librenvo.so,renvoAndroidInputQueueCreatedCallback
func androidInputQueueCreatedCallback() int { return 0 }

// renvo:linkstatic librenvo.so,renvoAndroidInputQueueDestroyedCallback
func androidInputQueueDestroyedCallback() int { return 0 }

// renvo:linkstatic librenvo.so,renvoAndroidInputPollCallback
func androidInputPollCallback() int { return 0 }

// renvo:linkstatic libandroid.so,ANativeWindow_setBuffersGeometry
func androidSetBuffersGeometry(window, width, height, format int) int { return -1 }

// renvo:linkstatic libandroid.so,ANativeWindow_getWidth
func androidNativeWindowWidth(window int) int { return 0 }

// renvo:linkstatic libandroid.so,ANativeWindow_getHeight
func androidNativeWindowHeight(window int) int { return 0 }

// renvo:linkstatic libandroid.so,ANativeWindow_lock
func androidLockWindow(window int, buffer *byte, dirty *byte) int { return -1 }

// renvo:linkstatic libandroid.so,ANativeWindow_unlockAndPost
func androidUnlockAndPost(window int) int { return -1 }

// renvo:linkstatic libc.so,memcpy
func androidCopyMemory(destination int, source *byte, size int) int { return 0 }

// renvo:linkstatic libandroid.so,ALooper_forThread
func androidLooperForThread() int { return 0 }

// renvo:linkstatic libandroid.so,AInputQueue_attachLooper
func androidAttachInputQueue(queue, looper, ident, callback, data int) {}

// renvo:linkstatic libandroid.so,AInputQueue_detachLooper
func androidDetachInputQueue(queue int) {}

// renvo:linkstatic libandroid.so,AInputQueue_getEvent
func androidInputQueueGetEvent(queue int, event *int) int { return -1 }

// renvo:linkstatic libandroid.so,AInputQueue_preDispatchEvent
func androidInputQueuePreDispatchEvent(queue, event int) int { return 0 }

// renvo:linkstatic libandroid.so,AInputQueue_finishEvent
func androidInputQueueFinishEvent(queue, event, handled int) {}

// renvo:linkstatic libandroid.so,AInputEvent_getType
func androidInputEventType(event int) int { return 0 }

// renvo:linkstatic libandroid.so,AKeyEvent_getAction
func androidKeyEventAction(event int) int { return 0 }

// renvo:linkstatic libandroid.so,AKeyEvent_getKeyCode
func androidKeyEventKeyCode(event int) int { return 0 }

// renvo:linkstatic libandroid.so,AKeyEvent_getMetaState
func androidKeyEventMetaState(event int) int { return 0 }

// renvo:linkstatic libandroid.so,AKeyEvent_getRepeatCount
func androidKeyEventRepeatCount(event int) int { return 0 }

// renvo:linkstatic libandroid.so,AMotionEvent_getAction
func androidMotionEventAction(event int) int { return 0 }

// renvo:linkstatic libandroid.so,AMotionEvent_getX
func androidMotionEventX(event, pointer int) int { return 0 }

// renvo:linkstatic libandroid.so,AMotionEvent_getY
func androidMotionEventY(event, pointer int) int { return 0 }

// renvo:linkstatic libandroid.so,AMotionEvent_getPointerId
func androidMotionEventPointerID(event, pointer int) int { return -1 }

// renvo:linkstatic libandroid.so,AMotionEvent_getPointerCount
func androidMotionEventPointerCount(event int) int { return 0 }

func androidMotionPointerIndex(event, id int) int {
	count := androidMotionEventPointerCount(event)
	for pointer := 0; pointer < count; pointer++ {
		if androidMotionEventPointerID(event, pointer) == id {
			return pointer
		}
	}
	return -1
}

func androidUint32(data []byte, at int) int {
	return int(data[at]) |
		int(data[at+1])<<8 |
		int(data[at+2])<<16 |
		int(data[at+3])<<24
}

func androidPointer(data []byte, at int) int {
	return androidUint32(data, at) | androidUint32(data, at+4)<<32
}

func androidStorePointer(address, value int) {
	data := make([]byte, 8)
	for i := 0; i < 8; i++ {
		data[i] = byte(value >> (i * 8))
	}
	androidCopyMemory(address, &data[0], len(data))
}

func androidInstallNativeActivityCallbacks() int {
	callbacks := androidNativeActivityCallbacks()
	created := androidWindowCreatedCallback()
	resized := androidWindowResizedCallback()
	redraw := androidWindowRedrawCallback()
	destroyed := androidWindowDestroyedCallback()
	inputCreated := androidInputQueueCreatedCallback()
	inputDestroyed := androidInputQueueDestroyedCallback()
	if callbacks == 0 || created == 0 || resized == 0 || redraw == 0 ||
		destroyed == 0 || inputCreated == 0 || inputDestroyed == 0 {
		return 0
	}
	// These are the public ANativeActivityCallbacks slots 7 through 12.
	androidStorePointer(callbacks+56, created)
	androidStorePointer(callbacks+64, resized)
	androidStorePointer(callbacks+72, redraw)
	androidStorePointer(callbacks+80, destroyed)
	androidStorePointer(callbacks+88, inputCreated)
	androidStorePointer(callbacks+96, inputDestroyed)
	return 1
}

func androidRetainCallbacks() {
	// These ordinary calls keep the callback functions reachable. The condition
	// is never true for a live window, but cannot be decided at compile time.
	if androidWindow != nil && androidWindow.native == -1 {
		androidOnNativeWindowCreated(0, 0)
		androidOnNativeWindowResized(0, 0)
		androidOnNativeWindowRedrawNeeded(0, 0)
		androidOnNativeWindowDestroyed(0, 0)
		androidOnInputQueueCreated(0, 0)
		androidOnInputQueueDestroyed(0, 0)
		androidOnInputReady(0, 0, 0)
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
	androidViewportWidth = options.Width
	androidDeviceScale = 1.0
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
	changedWindow := w.native != native
	w.native = native
	// A zero size selects Android's base window dimensions. Requesting the
	// logical Forms size here makes SurfaceFlinger enlarge a low-resolution
	// buffer, which blurs text and controls.
	if changedWindow {
		if androidSetBuffersGeometry(native, 0, 0,
			androidWindowFormatRGBA8888) != 0 {
			return
		}
	}
	physicalWidth := androidNativeWindowWidth(native)
	physicalHeight := androidNativeWindowHeight(native)
	if physicalWidth <= 0 || physicalHeight <= 0 || androidViewportWidth <= 0 {
		return
	}
	androidDeviceScale = Scalar(physicalWidth) / Scalar(androidViewportWidth)
	androidPhysicalWidth = physicalWidth
	if androidDeviceScale <= 0.0 {
		androidDeviceScale = 1.0
	}
	w.width = androidViewportWidth
	w.height = int(Scalar(physicalHeight) / androidDeviceScale)
	resized := w.surface.Width != physicalWidth || w.surface.Height != physicalHeight
	if resized {
		w.surface.Resize(physicalWidth, physicalHeight)
	}
	w.surface.setDeviceScale(androidDeviceScale)
	if resized {
		resizeEvent := Event{Type: EventWindowResize,
			Dirty: R(0, 0, Scalar(w.width), Scalar(w.height))}
		androidSendEvent(w, &resizeEvent)
		exposeEvent := Event{Type: EventWindowExpose,
			Dirty: R(0, 0, Scalar(w.width), Scalar(w.height))}
		androidSendEvent(w, &exposeEvent)
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
	androidActivePointer = androidNoPointer
}

func androidOnInputQueueCreated(queue, activity int) {
	if queue == 0 {
		return
	}
	if androidInputQueue != 0 {
		androidDetachInputQueue(androidInputQueue)
	}
	looper := androidLooperForThread()
	callback := androidInputPollCallback()
	if looper == 0 || callback == 0 {
		androidInputQueue = 0
		return
	}
	androidInputQueue = queue
	androidAttachInputQueue(queue, looper, androidLooperInput, callback, 0)
}

func androidOnInputQueueDestroyed(queue, activity int) {
	if queue != 0 {
		androidDetachInputQueue(queue)
	}
	if androidInputQueue == queue {
		androidInputQueue = 0
	}
	androidActivePointer = androidNoPointer
}

func androidSendEvent(w *Window, event *Event) {
	if w == nil || w.closed || event == nil {
		return
	}
	w.queue(*event)
	if w.EventHandler != nil {
		w.EventHandler()
	}
}

func androidQueueInputEvent(w *Window, event Event) {
	if event.Type == EventPointerMove && len(w.events) > w.eventHead &&
		w.events[len(w.events)-1].Type == EventPointerMove {
		w.events[len(w.events)-1] = event
		return
	}
	w.queue(event)
}

func androidModifiers(meta int) Modifiers {
	modifiers := Modifiers(0)
	if meta&androidMetaShift != 0 {
		modifiers |= ModifierShift
	}
	if meta&androidMetaAlt != 0 {
		modifiers |= ModifierAlt
	}
	if meta&androidMetaControl != 0 {
		modifiers |= ModifierControl
	}
	if meta&androidMetaCommand != 0 {
		modifiers |= ModifierCommand
	}
	return modifiers
}

func androidKey(code int) Key {
	if code == 67 {
		return KeyBackspace
	}
	if code == 112 {
		return KeyDelete
	}
	if code == 66 {
		return KeyEnter
	}
	if code == 61 {
		return KeyTab
	}
	if code == 111 {
		return KeyEscape
	}
	if code == 62 {
		return KeySpace
	}
	if code == 21 {
		return KeyLeft
	}
	if code == 22 {
		return KeyRight
	}
	if code == 19 {
		return KeyUp
	}
	if code == 20 {
		return KeyDown
	}
	if code == 122 {
		return KeyHome
	}
	if code == 123 {
		return KeyEnd
	}
	if code == 92 {
		return KeyPageUp
	}
	if code == 93 {
		return KeyPageDown
	}
	if code == 29 {
		return KeyA
	}
	if code == 30 {
		return KeyB
	}
	if code == 31 {
		return KeyC
	}
	if code == 37 {
		return KeyI
	}
	if code == 42 {
		return KeyN
	}
	if code == 43 {
		return KeyO
	}
	if code == 45 {
		return KeyQ
	}
	if code == 47 {
		return KeyS
	}
	if code == 50 {
		return KeyV
	}
	if code == 52 {
		return KeyX
	}
	if code == 53 {
		return KeyY
	}
	if code == 54 {
		return KeyZ
	}
	return KeyUnknown
}

func androidKeyText(code, meta int) string {
	shift := meta&androidMetaShift != 0
	if code >= 29 && code <= 54 {
		letters := "abcdefghijklmnopqrstuvwxyz"
		if shift {
			letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		}
		at := code - 29
		return letters[at : at+1]
	}
	if code >= 7 && code <= 16 {
		digits := "0123456789"
		shifted := ")!@#$%^&*("
		at := code - 7
		if shift {
			return shifted[at : at+1]
		}
		return digits[at : at+1]
	}
	if code == 62 {
		return " "
	}
	if code == 66 {
		// TextArea handles Enter from EventKeyDown. Emitting text as well would
		// insert two newlines for one physical key press.
		return ""
	}
	if code == 55 {
		if shift {
			return "<"
		}
		return ","
	}
	if code == 56 {
		if shift {
			return ">"
		}
		return "."
	}
	if code == 69 {
		if shift {
			return "_"
		}
		return "-"
	}
	if code == 70 {
		if shift {
			return "+"
		}
		return "="
	}
	if code == 71 {
		if shift {
			return "{"
		}
		return "["
	}
	if code == 72 {
		if shift {
			return "}"
		}
		return "]"
	}
	if code == 73 {
		if shift {
			return "|"
		}
		return "\\"
	}
	if code == 74 {
		if shift {
			return ":"
		}
		return ";"
	}
	if code == 75 {
		if shift {
			return "\""
		}
		return "'"
	}
	if code == 76 {
		if shift {
			return "?"
		}
		return "/"
	}
	if code == 68 {
		if shift {
			return "~"
		}
		return "`"
	}
	return ""
}

func androidHandleInputEvent(input int) int {
	w := androidWindow
	if w == nil || w.closed {
		return 0
	}
	inputType := androidInputEventType(input)
	if inputType == androidInputEventKey {
		action := androidKeyEventAction(input)
		if action != androidKeyActionDown && action != androidKeyActionUp {
			return 0
		}
		code := androidKeyEventKeyCode(input)
		meta := androidKeyEventMetaState(input)
		eventType := EventKeyDown
		if action == androidKeyActionUp {
			eventType = EventKeyUp
		}
		key := androidKey(code)
		keyText := androidKeyText(code, meta)
		if key == KeyUnknown && keyText == "" {
			// Leave Android system keys such as Back and volume unclaimed.
			return 0
		}
		event := Event{Type: eventType, Key: key,
			Modifiers: androidModifiers(meta),
			Repeat:    androidKeyEventRepeatCount(input) > 0}
		androidQueueInputEvent(w, event)
		if action == androidKeyActionDown && event.Modifiers&(ModifierControl|ModifierAlt|ModifierCommand) == 0 {
			if keyText != "" {
				androidQueueInputEvent(w, Event{Type: EventTextInput,
					Text: keyText, Modifiers: event.Modifiers})
			}
		}
		return 1
	}
	if inputType != androidInputEventMotion {
		return 0
	}
	action := androidMotionEventAction(input)
	kind := action & androidMotionActionMask
	actionPointer := (action >> androidMotionPointerIndexShift) & androidMotionPointerIndexMask
	eventType := EventNone
	pointer := actionPointer
	if kind == androidMotionActionDown {
		androidActivePointer = androidMotionEventPointerID(input, pointer)
		eventType = EventPointerDown
	} else if kind == androidMotionActionPointerDown {
		// Forms currently exposes one logical pointer. Keep the original pointer
		// active and consume additional contacts without disturbing its gesture.
		return 1
	} else if kind == androidMotionActionUp || kind == androidMotionActionPointerUp {
		if androidActivePointer == androidNoPointer ||
			androidMotionEventPointerID(input, pointer) != androidActivePointer {
			return 1
		}
		eventType = EventPointerUp
	} else if kind == androidMotionActionMove {
		if androidActivePointer == androidNoPointer {
			return 1
		}
		pointer = androidMotionPointerIndex(input, androidActivePointer)
		if pointer < 0 {
			return 1
		}
		eventType = EventPointerMove
	} else if kind == androidMotionActionCancel {
		if androidActivePointer == androidNoPointer {
			return 1
		}
		pointer = androidMotionPointerIndex(input, androidActivePointer)
		if pointer < 0 {
			pointer = 0
		}
		eventType = EventPointerCancel
	}
	if eventType == EventNone {
		return 0
	}
	physicalWidth := androidPhysicalWidth
	if physicalWidth <= 0 || androidViewportWidth <= 0 {
		return 0
	}
	x := androidMotionEventX(input, pointer) * androidViewportWidth / physicalWidth
	y := androidMotionEventY(input, pointer) * androidViewportWidth / physicalWidth
	event := Event{
		Type:   eventType,
		X:      Scalar(x),
		Y:      Scalar(y),
		Button: 1,
	}
	androidQueueInputEvent(w, event)
	if eventType == EventPointerUp || eventType == EventPointerCancel {
		androidActivePointer = androidNoPointer
	}
	return 1
}

// ALooper invokes this callback whenever the NativeActivity input queue has
// work. Finish a bounded batch before dispatching it to the client. An input
// producer can otherwise keep the native queue perpetually non-empty, grow the
// client event arena without bound, and starve both painting and the looper.
func androidOnInputReady(data, events, fd int) int {
	queue := androidInputQueue
	if queue == 0 {
		return 1
	}
	processed := 0
	for processed < androidInputBatchLimit {
		var input int
		if androidInputQueueGetEvent(queue, &input) < 0 || input == 0 {
			break
		}
		processed++
		if androidInputQueuePreDispatchEvent(queue, input) != 0 {
			continue
		}
		handled := androidHandleInputEvent(input)
		androidInputQueueFinishEvent(queue, input, handled)
	}
	// Native events are acknowledged before client painting. This prevents a
	// slow software-present from holding Android's input dispatcher hostage.
	w := androidWindow
	if w != nil && !w.closed && w.EventHandler != nil && len(w.events) > w.eventHead {
		w.EventHandler()
	}
	return 1
}

func (w *Window) Present() bool {
	if w == nil || w.closed || w.surface == nil {
		return false
	}
	if w.native == 0 {
		return true
	}
	buffer := androidNativeBuffer
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

func (w *Window) PollInto(event *Event) bool {
	if w == nil || event == nil {
		return false
	}
	next, ok := w.nextQueuedEvent()
	if !ok {
		return false
	}
	*event = next
	return true
}

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
	androidViewportWidth = width
	if w.native != 0 {
		androidAttachWindow(w.native)
		return true
	}
	w.surface.Resize(width, height)
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
	w.EventHandler = nil
	androidActivePointer = androidNoPointer
	if androidWindow == w {
		androidWindow = nil
	}
}
