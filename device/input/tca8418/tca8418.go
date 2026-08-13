// Package tca8418 drives the TCA8418 keypad-matrix controller.
package tca8418

const address = uint16(0x34)

const (
	registerConfig          = byte(0x01)
	registerInterruptStatus = byte(0x02)
	registerEventCount      = byte(0x03)
	registerEvent           = byte(0x04)
	registerGPIOInterrupt1  = byte(0x11)
	registerGPIOEvent1      = byte(0x20)
	registerGPIODirection1  = byte(0x23)
	registerGPIOLevel1      = byte(0x26)
	registerKeypadGPIO1     = byte(0x1d)
	configKeyInterrupt      = byte(0x01)
)

// Bus is the minimal I2C capability required by the controller.
type Bus interface {
	Tx(address uint16, write, read []byte) error
}

// Event reports a physical matrix transition. Row is in [0, 7) and Column is
// in [0, 8) for the matrix configured by Initialize.
type Event struct {
	Row, Column uint8
	Pressed     bool
}

// Device is one TCA8418 attached to a bus.
type Device struct {
	bus Bus
}

// New binds a TCA8418 to bus.
func New(bus Bus) *Device { return &Device{bus: bus} }

func (d *Device) writeRegister(register, value byte) error {
	data := [2]byte{register, value}
	return d.bus.Tx(address, data[:], nil)
}

func (d *Device) readRegister(register byte) (byte, error) {
	result := [1]byte{}
	if err := d.bus.Tx(address, []byte{register}, result[:]); err != nil {
		return 0, err
	}
	return result[0], nil
}

// Initialize configures the 7-row by 8-column matrix used by the Cardputer
// Adv, drains stale events, and enables key-event interrupts.
func (d *Device) Initialize() error {
	// Match the controller vendor's recommended GPIO reset state before assigning
	// the lowest seven row and eight column pins to the keypad engine.
	for register := registerGPIODirection1; register < registerGPIODirection1+3; register++ {
		if err := d.writeRegister(register, 0); err != nil {
			return err
		}
	}
	for register := registerGPIOEvent1; register < registerGPIOEvent1+3; register++ {
		if err := d.writeRegister(register, 0xff); err != nil {
			return err
		}
	}
	for register := registerGPIOLevel1; register < registerGPIOLevel1+3; register++ {
		if err := d.writeRegister(register, 0); err != nil {
			return err
		}
	}
	for register := registerGPIOInterrupt1; register < registerGPIOInterrupt1+3; register++ {
		if err := d.writeRegister(register, 0xff); err != nil {
			return err
		}
	}
	if err := d.writeRegister(registerKeypadGPIO1, 0x7f); err != nil {
		return err
	}
	if err := d.writeRegister(registerKeypadGPIO1+1, 0xff); err != nil {
		return err
	}
	for drained := 0; drained < 10; drained++ {
		raw, err := d.readRegister(registerEvent)
		if err != nil {
			return err
		}
		if raw == 0 {
			break
		}
	}
	if err := d.writeRegister(registerInterruptStatus, 0x03); err != nil {
		return err
	}
	config, err := d.readRegister(registerConfig)
	if err != nil {
		return err
	}
	return d.writeRegister(registerConfig, config|configKeyInterrupt)
}

// NextEvent removes the oldest matrix event. ok is false when the FIFO is
// empty. Reading the event counter first avoids treating the empty FIFO's zero
// value as a key transition.
func (d *Device) NextEvent() (event Event, ok bool, err error) {
	count, err := d.readRegister(registerEventCount)
	if err != nil {
		return event, false, err
	}
	if count&0x0f == 0 {
		return event, false, nil
	}
	raw, err := d.readRegister(registerEvent)
	if err != nil {
		return event, false, err
	}
	code := raw & 0x7f
	if code == 0 {
		return event, false, ErrInvalidEvent
	}
	code--
	event.Row = code / 10
	event.Column = code % 10
	event.Pressed = raw&0x80 != 0
	if event.Row >= 7 || event.Column >= 8 {
		return Event{}, false, ErrInvalidEvent
	}
	if err := d.writeRegister(registerInterruptStatus, 0x01); err != nil {
		return Event{}, false, err
	}
	return event, true, nil
}

// controllerError is allocation-free and implements error.
type controllerError string

func (e controllerError) Error() string { return string(e) }

// ErrInvalidEvent reports a FIFO value outside the configured matrix.
const ErrInvalidEvent controllerError = "tca8418 invalid keypad event"
