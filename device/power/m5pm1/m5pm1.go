// Package m5pm1 drives the M5Stack M5PM1 power-management controller.
package m5pm1

const Address = uint16(0x6e)

const (
	registerDeviceID     = byte(0x00)
	registerI2CConfig    = byte(0x09)
	registerWatchdog     = byte(0x0a)
	registerGPIOMode     = byte(0x10)
	registerGPIOOutput   = byte(0x11)
	registerGPIODrive    = byte(0x13)
	registerGPIOFunction = byte(0x16)
	deviceID             = uint16(0x2050)
)

// Pin identifies one of the M5PM1's five GPIOs.
type Pin uint8

const (
	Pin0 Pin = iota
	Pin1
	Pin2
	Pin3
	Pin4
)

// Bus is the minimal I2C capability required by the controller.
type Bus interface {
	Tx(address uint16, write, read []byte) error
}

// Device is one M5PM1 attached to a bus.
type Device struct {
	bus Bus
}

// New binds an M5PM1 to bus without touching hardware.
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

func (d *Device) update(register, mask, value byte) error {
	current, err := d.read(register)
	if err != nil {
		return err
	}
	return d.write(register, current&^mask|value&mask)
}

// Identify reads and validates the fixed M5PM1 device ID without changing any
// register. The returned ID is meaningful even when ErrDeviceID is returned.
func (d *Device) Identify() (uint16, error) {
	data := [2]byte{}
	if err := d.bus.Tx(Address, []byte{registerDeviceID}, data[:]); err != nil {
		return 0, err
	}
	id := uint16(data[0]) | uint16(data[1])<<8
	if id != deviceID {
		return id, ErrDeviceID
	}
	return id, nil
}

// Initialize validates the controller, disables I2C idle sleep, and disables
// its watchdog. It does not change power rails, charging, or GPIO state.
func (d *Device) Initialize() error {
	if _, err := d.Identify(); err != nil {
		return err
	}
	if err := d.write(registerI2CConfig, 0); err != nil {
		return err
	}
	return d.write(registerWatchdog, 0)
}

// ConfigureOutput assigns pin to ordinary push-pull GPIO and establishes
// level before enabling its output driver. M5PM1 state can survive a battery
// shutdown, so every field is set explicitly rather than assumed from reset.
func (d *Device) ConfigureOutput(pin Pin, level bool) error {
	if !validPin(pin) {
		return ErrInvalidPin
	}
	mask := byte(1 << uint8(pin))
	value := byte(0)
	if level {
		value = mask
	}
	if err := d.update(registerGPIOOutput, mask, value); err != nil {
		return err
	}
	if err := d.update(registerGPIODrive, mask, 0); err != nil {
		return err
	}
	functionRegister := registerGPIOFunction
	functionIndex := uint8(pin)
	if pin == Pin4 {
		functionRegister++
		functionIndex = 0
	}
	shift := functionIndex * 2
	functionMask := byte(3 << shift)
	if err := d.update(functionRegister, functionMask, 0); err != nil {
		return err
	}
	return d.update(registerGPIOMode, mask, mask)
}

func validPin(pin Pin) bool { return pin <= Pin4 }

type deviceError string

func (e deviceError) Error() string { return string(e) }

const (
	// ErrDeviceID reports an acknowledgement from something other than M5PM1.
	ErrDeviceID deviceError = "unexpected M5PM1 device ID"
	// ErrInvalidPin reports a pin outside the M5PM1 GPIO0..GPIO4 range.
	ErrInvalidPin deviceError = "M5PM1 GPIO must be between 0 and 4"
)
