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

type Event struct {
	Type      EventType
	Dirty     Rect
	X         Scalar
	Y         Scalar
	WheelX    Scalar
	WheelY    Scalar
	Key       int
	Button    int
	Modifiers Modifiers
	Text      string
	TimerID   int
	Repeat    bool
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
