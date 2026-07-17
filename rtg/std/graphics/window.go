package graphics

type WindowOptions struct {
	Title  string
	Width  int
	Height int
	Hidden bool
}

type EventType int

const (
	EventNone EventType = iota
	EventWindowClose
	EventWindowResize
	EventWindowFocusGained
	EventWindowFocusLost
	EventWindowExpose
	EventPointerMove
	EventPointerDown
	EventPointerUp
	EventPointerWheel
	EventPointerLeave
	EventKeyDown
	EventKeyUp
	EventTextInput
	EventTimer
)

type Modifiers int

const (
	ModifierShift Modifiers = 1 << iota
	ModifierControl
	ModifierAlt
	ModifierCommand
)

type Cursor int

const (
	CursorArrow Cursor = iota
	CursorIBeam
	CursorCrosshair
	CursorPointingHand
	CursorResizeHorizontal
	CursorResizeVertical
)

// Key is a platform-neutral physical key used for commands. Printable text is
// delivered separately through EventTextInput so keyboard layout and composed
// characters do not leak into editor shortcut handling.
type Key int

const (
	KeyUnknown Key = iota
	KeyBackspace
	KeyDelete
	KeyEnter
	KeyTab
	KeyEscape
	KeySpace
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyA
	KeyC
	KeyS
	KeyV
	KeyX
	KeyY
	KeyZ
)

type Event struct {
	Type      EventType
	Dirty     Rect
	X         Scalar
	Y         Scalar
	WheelX    Scalar
	WheelY    Scalar
	Key       Key
	Button    int
	Modifiers Modifiers
	Text      string
	TimerID   int
	Repeat    bool
}

const wheelStepPixels Scalar = 48.0

func wheelDeltaPixels(delta Scalar, precise bool) Scalar {
	if precise {
		return delta
	}
	return delta * wheelStepPixels
}

// textInputForKey keeps non-text physical keys out of the text-input stream.
// Cocoa reports navigation keys both as key events and as private-use Unicode
// characters, so forwarding their "characters" value would insert arrows into
// an editor document.
func textInputForKey(key Key, text string) string {
	if key == KeyBackspace || key == KeyDelete || key == KeyEscape {
		return ""
	}
	if key == KeyLeft || key == KeyRight || key == KeyUp || key == KeyDown {
		return ""
	}
	if key == KeyHome || key == KeyEnd || key == KeyPageUp || key == KeyPageDown {
		return ""
	}
	return text
}

type Error struct {
	Message string
	Code    int
}

var lastWindowError Error
var hasLastWindowError bool

// LastWindowError reports why the most recent NewWindow call failed.
// It returns nil after a successful call or before window creation is attempted.
func LastWindowError() *Error {
	if !hasLastWindowError {
		return nil
	}
	return &lastWindowError
}

func clearLastWindowError() {
	hasLastWindowError = false
	lastWindowError = Error{}
}

func setLastWindowError(message string, code int) {
	hasLastWindowError = true
	lastWindowError = Error{Message: message, Code: code}
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// Window owns a software RGBA surface and a platform presentation target.
type Window struct {
	width          int
	height         int
	surface        *Surface
	events         []Event
	closed         bool
	active         bool
	shown          bool
	wasVisible     bool
	focused        bool
	captured       bool
	pointerInside  bool
	cursor         Cursor
	timerIDs       [16]int
	timerDeadlines [16]Scalar
	timerActive    [16]bool

	app          int
	native       int
	view         int
	context      int
	device       int
	instance     int
	pool         int
	eventMode    int
	backingScale int
	bottomUp     []byte
	tracking     bool
	pendingUTF16 int
}

func (w *Window) Surface() *Surface { return w.surface }
func (w *Window) Size() (int, int)  { return w.width, w.height }

func (w *Window) nextQueuedEvent() (Event, bool) {
	if len(w.events) == 0 {
		return Event{}, false
	}
	e := w.events[0]
	w.events = w.events[1:]
	return e, true
}

func (w *Window) queue(e Event) { w.events = append(w.events, e) }

func windowsKeyFromVirtual(key int) Key {
	if key == 8 {
		return KeyBackspace
	}
	if key == 9 {
		return KeyTab
	}
	if key == 13 {
		return KeyEnter
	}
	if key == 27 {
		return KeyEscape
	}
	if key == 32 {
		return KeySpace
	}
	if key == 33 {
		return KeyPageUp
	}
	if key == 34 {
		return KeyPageDown
	}
	if key == 35 {
		return KeyEnd
	}
	if key == 36 {
		return KeyHome
	}
	if key == 37 {
		return KeyLeft
	}
	if key == 38 {
		return KeyUp
	}
	if key == 39 {
		return KeyRight
	}
	if key == 40 {
		return KeyDown
	}
	if key == 46 {
		return KeyDelete
	}
	if key == 65 {
		return KeyA
	}
	if key == 67 {
		return KeyC
	}
	if key == 83 {
		return KeyS
	}
	if key == 86 {
		return KeyV
	}
	if key == 88 {
		return KeyX
	}
	if key == 89 {
		return KeyY
	}
	if key == 90 {
		return KeyZ
	}
	return KeyUnknown
}

func darwinKeyFromCode(key int) Key {
	if key == 51 {
		return KeyBackspace
	}
	if key == 117 {
		return KeyDelete
	}
	if key == 36 {
		return KeyEnter
	}
	if key == 48 {
		return KeyTab
	}
	if key == 53 {
		return KeyEscape
	}
	if key == 49 {
		return KeySpace
	}
	if key == 123 {
		return KeyLeft
	}
	if key == 124 {
		return KeyRight
	}
	if key == 126 {
		return KeyUp
	}
	if key == 125 {
		return KeyDown
	}
	if key == 115 {
		return KeyHome
	}
	if key == 119 {
		return KeyEnd
	}
	if key == 116 {
		return KeyPageUp
	}
	if key == 121 {
		return KeyPageDown
	}
	if key == 0 {
		return KeyA
	}
	if key == 8 {
		return KeyC
	}
	if key == 1 {
		return KeyS
	}
	if key == 9 {
		return KeyV
	}
	if key == 7 {
		return KeyX
	}
	if key == 16 {
		return KeyY
	}
	if key == 6 {
		return KeyZ
	}
	return KeyUnknown
}
