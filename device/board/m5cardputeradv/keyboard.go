package board

import "renvo.dev/device/input/tca8418"

// Key identifies a printable key or one of the named non-printable keys. A
// printable Key has the same value as its lower-case ASCII character.
type Key uint16

const (
	KeyNone Key = 0

	KeyEscape Key = 0x100 + iota
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	KeyBackspace
	KeyDelete
	KeyTab
	KeyFunction
	KeyShift
	KeyUp
	KeyEnter
	KeyControl
	KeyOption
	KeyAlt
	KeyLeft
	KeyDown
	KeyRight
)

// KeyEvent describes one keyboard transition. Character is nonzero for a
// printable press or release and reflects the Shift state at that transition.
type KeyEvent struct {
	Row, Column int
	Key         Key
	Character   byte
	Pressed     bool
}

type keypad interface {
	Initialize() error
	NextEvent() (tca8418.Event, bool, error)
}

// KeyboardDevice translates the Cardputer Adv's physical matrix into keys.
type KeyboardDevice struct {
	device               keypad
	ready                bool
	function, shift      bool
	control, option, alt bool
}

func newKeyboard(device keypad) *KeyboardDevice { return &KeyboardDevice{device: device} }

// Initialize configures the keypad controller. NextEvent calls it
// automatically.
func (k *KeyboardDevice) Initialize() error {
	if k.ready {
		return nil
	}
	if err := k.device.Initialize(); err != nil {
		return err
	}
	k.ready = true
	return nil
}

var baseKeys = [4][14]Key{
	{'`', '1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '-', '=', KeyBackspace},
	{KeyTab, 'q', 'w', 'e', 'r', 't', 'y', 'u', 'i', 'o', 'p', '[', ']', '\\'},
	{KeyFunction, KeyShift, 'a', 's', 'd', 'f', 'g', 'h', 'j', 'k', 'l', ';', '\'', KeyEnter},
	{KeyControl, KeyOption, KeyAlt, 'z', 'x', 'c', 'v', 'b', 'n', 'm', ',', '.', '/', ' '},
}

var shiftedCharacters = [4][14]byte{
	{'~', '!', '@', '#', '$', '%', '^', '&', '*', '(', ')', '_', '+', 0},
	{0, 'Q', 'W', 'E', 'R', 'T', 'Y', 'U', 'I', 'O', 'P', '{', '}', '|'},
	{0, 0, 'A', 'S', 'D', 'F', 'G', 'H', 'J', 'K', 'L', ':', '"', 0},
	{0, 0, 0, 'Z', 'X', 'C', 'V', 'B', 'N', 'M', '<', '>', '?', ' '},
}

var functionKeys = [4][14]Key{
	{KeyEscape, KeyF1, KeyF2, KeyF3, KeyF4, KeyF5, KeyF6, KeyF7, KeyF8, KeyF9, KeyF10, KeyF11, KeyF12, KeyDelete},
	{},
	{KeyFunction, KeyNone, KeyNone, KeyNone, KeyNone, KeyNone, KeyNone, KeyNone, KeyNone, KeyNone, KeyNone, KeyUp},
	{KeyNone, KeyNone, KeyNone, KeyNone, KeyNone, KeyNone, KeyNone, KeyNone, KeyNone, KeyNone, KeyLeft, KeyDown, KeyRight},
}

func remapKey(event tca8418.Event) (row, column int) {
	column = int(event.Row) * 2
	if event.Column > 3 {
		column++
	}
	row = int(event.Column % 4)
	return row, column
}

func printable(key Key) bool { return key >= 0x20 && key < 0x7f }

func (k *KeyboardDevice) translate(row, column int) (Key, byte) {
	key := baseKeys[row][column]
	if k.function && functionKeys[row][column] != KeyNone {
		return functionKeys[row][column], 0
	}
	if !printable(key) {
		return key, 0
	}
	character := byte(key)
	if k.shift {
		character = shiftedCharacters[row][column]
	}
	return key, character
}

func (k *KeyboardDevice) setModifier(key Key, pressed bool) {
	switch key {
	case KeyFunction:
		k.function = pressed
	case KeyShift:
		k.shift = pressed
	case KeyControl:
		k.control = pressed
	case KeyOption:
		k.option = pressed
	case KeyAlt:
		k.alt = pressed
	}
}

// NextEvent returns the oldest pending key transition. ok is false when there
// is no queued event. Polling the controller means applications remain correct
// even if several transitions occur before the interrupt line is sampled.
func (k *KeyboardDevice) NextEvent() (event KeyEvent, ok bool, err error) {
	if err = k.Initialize(); err != nil {
		return event, false, err
	}
	raw, ok, err := k.device.NextEvent()
	if err != nil || !ok {
		return event, ok, err
	}
	event.Row, event.Column = remapKey(raw)
	event.Pressed = raw.Pressed
	event.Key, event.Character = k.translate(event.Row, event.Column)
	k.setModifier(baseKeys[event.Row][event.Column], event.Pressed)
	return event, true, nil
}

// Modifier state can be queried after each event.
func (k *KeyboardDevice) FunctionPressed() bool { return k.function }
func (k *KeyboardDevice) ShiftPressed() bool    { return k.shift }
func (k *KeyboardDevice) ControlPressed() bool  { return k.control }
func (k *KeyboardDevice) OptionPressed() bool   { return k.option }
func (k *KeyboardDevice) AltPressed() bool      { return k.alt }
