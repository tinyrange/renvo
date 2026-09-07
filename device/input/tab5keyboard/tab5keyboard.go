// Package tab5keyboard implements M5Stack Tab5 Keyboard I2C, including polling
// without INT. A Device must not be used concurrently.
package tab5keyboard

type Bus interface {
	Tx(address uint16, write, read []byte) error
	DelayMilliseconds(uint32)
}

const DefaultAddress = uint16(0x6d)

type driverError string

func (e driverError) Error() string { return string(e) }

const (
	ErrRange    driverError = "tab5keyboard: value outside protocol range"
	ErrProtocol driverError = "tab5keyboard: invalid event"
)

type Mode byte

const (
	Normal Mode = iota
	HID
	Character
)

type RGBMode byte

const (
	RGBSystem RGBMode = iota
	RGBCustom
)

const (
	InterruptNormal    byte = 1
	InterruptHID       byte = 2
	InterruptCharacter byte = 4
	ModifierControl    byte = 1
	ModifierAlt        byte = 4
)

type KeyEvent struct {
	Row, Column byte
	Pressed     bool
}

// HIDEvent has USB keyboard-page usage codes; Keycode zero is a release.
type HIDEvent struct{ Modifier, Keycode byte }

// CharacterEvent is a press (no release/repeat). Text is a printable byte or
// key name, after firmware Aa/Sym translation. Length excludes Modifier.
type CharacterEvent struct {
	Modifier byte
	Length   int
	Text     [9]byte
}

type Device struct {
	bus     Bus
	address uint16
	command [8]byte
	data    [10]byte
}

// New binds a bus without changing the keyboard configuration.
func New(bus Bus) *Device { return &Device{bus: bus, address: DefaultAddress} }

// NewAt binds a keyboard whose address was previously changed.
func NewAt(bus Bus, address uint16) (*Device, error) {
	if address < 0x08 || address > 0x77 {
		return nil, ErrRange
	}
	return &Device{bus: bus, address: address}, nil
}

func (d *Device) Address() uint16 { return d.address }

func (d *Device) read(register byte, size int) error {
	d.command[0] = register
	return d.bus.Tx(d.address, d.command[:1], d.data[:size])
}

func (d *Device) readByte(register byte) (byte, error) {
	if err := d.read(register, 1); err != nil {
		return 0, err
	}
	return d.data[0], nil
}

func (d *Device) writeByte(register, value byte) error {
	d.command[0], d.command[1] = register, value
	return d.bus.Tx(d.address, d.command[:2], nil)
}

// Initialize selects a mode and clears input, leaving RGB settings unchanged.
func (d *Device) Initialize(mode Mode) error {
	if mode > Character {
		return ErrRange
	}
	if _, err := d.FirmwareVersion(); err != nil {
		return err
	}
	if err := d.SetMode(mode); err != nil {
		return err
	}
	return d.ClearEvents()
}

func (d *Device) FirmwareVersion() (byte, error) { return d.readByte(0xfe) }
func (d *Device) Mode() (Mode, error)            { value, err := d.readByte(0x10); return Mode(value), err }

// SetMode clears the previous mode's queue when the mode changes.
func (d *Device) SetMode(mode Mode) error {
	if mode > Character {
		return ErrRange
	}
	return d.writeByte(0x10, byte(mode))
}

func (d *Device) InterruptEnable() (byte, error) { return d.readByte(0x00) }
func (d *Device) SetInterruptEnable(mask byte) error {
	if mask & ^byte(7) != 0 {
		return ErrRange
	}
	return d.writeByte(0x00, mask)
}
func (d *Device) InterruptStatus() (byte, error) { return d.readByte(0x01) }
func (d *Device) ClearInterrupt() error          { return d.writeByte(0x01, 0) }
func (d *Device) EventCount() (byte, error)      { return d.readByte(0x02) }
func (d *Device) ClearEvents() error             { return d.writeByte(0x02, 0) }
func (d *Device) Brightness() (byte, error)      { return d.readByte(0x03) }
func (d *Device) SetBrightness(percent byte) error {
	if percent > 100 {
		return ErrRange
	}
	return d.writeByte(0x03, percent)
}
func (d *Device) RGBMode() (RGBMode, error) {
	value, err := d.readByte(0x11)
	return RGBMode(value), err
}
func (d *Device) SetRGBMode(mode RGBMode) error {
	if mode > RGBCustom {
		return ErrRange
	}
	return d.writeByte(0x11, byte(mode))
}

// RGB reads a custom LED buffer (index 0 or 1), regardless of active RGB mode.
func (d *Device) RGB(index byte) (red, green, blue byte, err error) {
	if index > 1 {
		return 0, 0, 0, ErrRange
	}
	if err := d.read(0x60+index*4, 3); err != nil {
		return 0, 0, 0, err
	}
	return d.data[2], d.data[1], d.data[0], nil
}
func (d *Device) SetRGB(index, red, green, blue byte) error {
	if index > 1 {
		return ErrRange
	}
	d.command[0], d.command[1], d.command[2], d.command[3] = 0x60+index*4, blue, green, red
	return d.bus.Tx(d.address, d.command[:4], nil)
}

// SetAddress persists a new address in device flash. Avoid repeated writes.
// The local address changes only after an acknowledged write; allow 50ms for flash.
func (d *Device) SetAddress(address uint16) error {
	if address < 0x08 || address > 0x77 {
		return ErrRange
	}
	if address == d.address {
		return nil
	}
	if err := d.writeByte(0xff, byte(address)); err != nil {
		return err
	}
	d.address = address
	d.bus.DelayMilliseconds(50)
	return nil
}

// NextKey reads one Normal-mode event. ok=false with nil error means empty.
// Call only while Normal mode is active; each successful read consumes an event.
func (d *Device) NextKey() (KeyEvent, bool, error) {
	value, err := d.readByte(0x20)
	if err != nil {
		return KeyEvent{}, false, err
	}
	if value == 0xff {
		return KeyEvent{}, false, nil
	}
	row, column := (value>>4)&7, value&15
	if row >= 5 || column >= 14 {
		return KeyEvent{}, false, ErrProtocol
	}
	return KeyEvent{row, column, value&0x80 != 0}, true, nil
}

// NextHID reads one HID-mode event, including release events with Keycode zero.
func (d *Device) NextHID() (HIDEvent, bool, error) {
	if err := d.read(0x30, 2); err != nil {
		return HIDEvent{}, false, err
	}
	if d.data[0] == 0xff && d.data[1] == 0xff {
		return HIDEvent{}, false, nil
	}
	return HIDEvent{d.data[0], d.data[1]}, true, nil
}

// NextCharacter consumes one Character-mode event. Firmware length includes
// the modifier, despite the original PDF's erroneous +1 instruction.
func (d *Device) NextCharacter() (CharacterEvent, bool, error) {
	length, err := d.readByte(0x40)
	if err != nil {
		return CharacterEvent{}, false, err
	}
	if length == 0 {
		return CharacterEvent{}, false, nil
	}
	if length < 2 || length > 10 {
		return CharacterEvent{}, false, ErrProtocol
	}
	if err := d.read(0x50, int(length)); err != nil {
		return CharacterEvent{}, false, err
	}
	event := CharacterEvent{Modifier: d.data[0], Length: int(length) - 1}
	for i := 0; i < event.Length; i++ {
		event.Text[i] = d.data[i+1]
	}
	return event, true, nil
}

// TerminalBytes translates a Character event to VT input. dst needs 5 bytes.
// Unknown key names return zero. Alt prefixes ESC; Ctrl maps ASCII controls.
func (e CharacterEvent) TerminalBytes(dst []byte) int {
	if len(dst) < 5 || e.Length < 1 || e.Length > len(e.Text) {
		return 0
	}
	sequence := ""
	if e.Length == 1 {
		value := e.Text[0]
		if e.Modifier&ModifierControl != 0 {
			if value >= 'a' && value <= 'z' {
				value -= 'a' - 'A'
			}
			if value >= '@' && value <= '_' {
				value &= 31
			} else if value == ' ' {
				value = 0
			} else if value == '?' {
				value = 127
			}
		}
		if e.Modifier&ModifierAlt != 0 {
			dst[0], dst[1] = 27, value
			return 2
		}
		dst[0] = value
		return 1
	}
	switch string(e.Text[:e.Length]) {
	case "esc", "ESC":
		sequence = "\x1b"
	case "tab", "TAB":
		sequence = "\t"
	case "enter", "ENTER":
		sequence = "\r\n"
	case "backspace", "BACKSPACE":
		sequence = "\x08"
	case "del", "DEL":
		sequence = "\x1b[3~"
	case "up", "UP":
		sequence = "\x1b[A"
	case "down", "DOWN":
		sequence = "\x1b[B"
	case "right", "RIGHT":
		sequence = "\x1b[C"
	case "left", "LEFT":
		sequence = "\x1b[D"
	}
	if len(sequence) == 0 {
		return 0
	}
	n := 0
	if e.Modifier&ModifierAlt != 0 {
		dst[0] = 27
		n = 1
	}
	for i := 0; i < len(sequence); i++ {
		dst[n] = sequence[i]
		n++
	}
	return n
}
