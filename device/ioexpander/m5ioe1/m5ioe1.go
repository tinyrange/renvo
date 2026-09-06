// Package m5ioe1 drives the M5Stack M5IOE1 14-pin I/O expander.
package m5ioe1

const Address = uint16(0x4f)

const (
	registerUID       = byte(0x00)
	registerMode      = byte(0x03)
	registerOutput    = byte(0x05)
	registerDrive     = byte(0x13)
	registerPWM1Duty  = byte(0x1b)
	registerI2CConfig = byte(0x23)
	registerPWMFreq   = byte(0x25)
)

// Pin identifies one of the M5IOE1 pins using the number printed in M5Stack
// schematics and documentation.
type Pin uint8

const (
	Pin1 Pin = iota + 1
	Pin2
	Pin3
	Pin4
	Pin5
	Pin6
	Pin7
	Pin8
	Pin9
	Pin10
	Pin11
	Pin12
	Pin13
	Pin14
)

// Bus is the minimal I2C capability required by the expander.
type Bus interface {
	Tx(address uint16, write, read []byte) error
}

// Device is one M5IOE1 attached to a bus.
type Device struct {
	bus Bus
}

// New binds an M5IOE1 to bus without touching hardware.
func New(bus Bus) *Device { return &Device{bus: bus} }

func (d *Device) read(register byte) (byte, error) {
	data := [1]byte{}
	if err := d.bus.Tx(Address, []byte{register}, data[:]); err != nil {
		return 0, err
	}
	return data[0], nil
}

func (d *Device) write(register, value byte) error {
	data := [2]byte{register, value}
	return d.bus.Tx(Address, data[:], nil)
}

func (d *Device) write16(register byte, value uint16) error {
	data := [3]byte{register, byte(value), byte(value >> 8)}
	return d.bus.Tx(Address, data[:], nil)
}

func (d *Device) update(register, mask, value byte) error {
	current, err := d.read(register)
	if err != nil {
		return err
	}
	return d.write(register, current&^mask|value&mask)
}

// Identify reads the per-device UID without writing any register. M5Stack does
// not publish a fixed expected value, so successful acknowledgement and read
// are the identity check.
func (d *Device) Identify() (uint16, error) {
	data := [2]byte{}
	if err := d.bus.Tx(Address, []byte{registerUID}, data[:]); err != nil {
		return 0, err
	}
	return uint16(data[0]) | uint16(data[1])<<8, nil
}

// Initialize verifies communication and disables I2C idle sleep. It does not
// change any GPIO direction, drive mode, or output latch.
func (d *Device) Initialize() error {
	if _, err := d.Identify(); err != nil {
		return err
	}
	return d.write(registerI2CConfig, 0)
}

// ConfigureOutput establishes level before selecting push-pull output mode.
// This ordering avoids momentarily driving a stale persistent latch value.
func (d *Device) ConfigureOutput(pin Pin, level bool) error {
	register, mask, err := pinRegister(registerOutput, pin)
	if err != nil {
		return err
	}
	value := byte(0)
	if level {
		value = mask
	}
	if err := d.update(register, mask, value); err != nil {
		return err
	}
	register, mask, _ = pinRegister(registerDrive, pin)
	if err := d.update(register, mask, 0); err != nil {
		return err
	}
	register, mask, _ = pinRegister(registerMode, pin)
	return d.update(register, mask, mask)
}

// ConfigurePWM prepares one of the expander's fixed PWM pins as a push-pull
// output and programs the frequency shared by all four PWM channels.
func (d *Device) ConfigurePWM(pin Pin, frequency uint16) error {
	if pin != Pin8 && pin != Pin9 {
		return ErrInvalidPWMPin
	}
	if frequency == 0 {
		return ErrInvalidPWMFrequency
	}
	// The expander retains state across application resets. Disable the channel
	// before changing GPIO mode so a stale duty cannot flash the LED.
	if err := d.SetPWMDuty(pin, 0); err != nil {
		return err
	}
	if err := d.ConfigureOutput(pin, false); err != nil {
		return err
	}
	return d.write16(registerPWMFreq, frequency)
}

// SetPWMDuty sets a configured fixed PWM pin's 12-bit duty. Zero disables the
// channel; nonzero values use normal polarity. Pin8 is PWM2 and Pin9 is PWM1.
func (d *Device) SetPWMDuty(pin Pin, duty uint16) error {
	register := registerPWM1Duty
	if pin == Pin8 {
		register += 2
	} else if pin != Pin9 {
		return ErrInvalidPWMPin
	}
	if duty > 0x0fff {
		return ErrInvalidPWMDuty
	}
	value := duty
	if duty != 0 {
		value |= 0x8000
	}
	return d.write16(register, value)
}

// SetOutput changes the persistent output latch of a configured pin.
func (d *Device) SetOutput(pin Pin, level bool) error {
	register, mask, err := pinRegister(registerOutput, pin)
	if err != nil {
		return err
	}
	value := byte(0)
	if level {
		value = mask
	}
	return d.update(register, mask, value)
}

// Output reports the persistent output-latch value, not the physical pin.
func (d *Device) Output(pin Pin) (bool, error) {
	register, mask, err := pinRegister(registerOutput, pin)
	if err != nil {
		return false, err
	}
	value, err := d.read(register)
	if err != nil {
		return false, err
	}
	return value&mask != 0, nil
}

func pinRegister(base byte, pin Pin) (byte, byte, error) {
	if pin < Pin1 || pin > Pin14 {
		return 0, 0, ErrInvalidPin
	}
	index := byte(pin - 1)
	return base + index/8, byte(1 << (index & 7)), nil
}

type deviceError string

func (e deviceError) Error() string { return string(e) }

// ErrInvalidPin reports a pin outside the documented IO1..IO14 range.
const ErrInvalidPin deviceError = "M5IOE1 pin must be between 1 and 14"

const (
	ErrInvalidPWMPin       deviceError = "M5IOE1 PWM pin must be pin 8 or 9"
	ErrInvalidPWMFrequency deviceError = "M5IOE1 PWM frequency must be nonzero"
	ErrInvalidPWMDuty      deviceError = "M5IOE1 PWM duty must fit in 12 bits"
)
