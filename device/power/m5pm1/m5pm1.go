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
	registerPWM0Duty     = byte(0x30)
	registerPWMFrequency = byte(0x34)
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

// ConfigurePWM assigns GPIO3 to PWM0 or GPIO4 to PWM1 and sets their shared
// counter frequency. The channel is disabled before its pin function changes,
// so persistent M5PM1 state cannot momentarily drive an unknown duty cycle.
func (d *Device) ConfigurePWM(pin Pin, frequency uint16) error {
	if !validPWMPin(pin) {
		return ErrInvalidPWMPin
	}
	if frequency == 0 {
		return ErrInvalidPWMFrequency
	}
	if err := d.SetPWMDuty(pin, 0); err != nil {
		return err
	}
	mask := byte(1 << uint8(pin))
	if err := d.update(registerGPIODrive, mask, 0); err != nil {
		return err
	}
	functionRegister, shift := pwmFunction(pin)
	functionMask := byte(3 << shift)
	if err := d.update(functionRegister, functionMask, functionMask); err != nil {
		return err
	}
	return d.write16(registerPWMFrequency, frequency)
}

// SetPWMDuty sets a configured PWM channel's 12-bit duty cycle. Zero disables
// the channel; nonzero values use normal (non-inverted) polarity.
func (d *Device) SetPWMDuty(pin Pin, duty uint16) error {
	if !validPWMPin(pin) {
		return ErrInvalidPWMPin
	}
	if duty > 0x0fff {
		return ErrInvalidPWMDuty
	}
	control := uint16(0)
	if duty != 0 {
		control = 0x1000
	}
	register := registerPWM0Duty
	if pin == Pin4 {
		register += 2
	}
	return d.write16(register, duty|control)
}

func validPWMPin(pin Pin) bool { return pin == Pin3 || pin == Pin4 }

func pwmFunction(pin Pin) (byte, uint8) {
	if pin == Pin3 {
		return registerGPIOFunction, 6
	}
	return registerGPIOFunction + 1, 0
}

func validPin(pin Pin) bool { return pin <= Pin4 }

type deviceError string

func (e deviceError) Error() string { return string(e) }

const (
	// ErrDeviceID reports an acknowledgement from something other than M5PM1.
	ErrDeviceID deviceError = "unexpected M5PM1 device ID"
	// ErrInvalidPin reports a pin outside the M5PM1 GPIO0..GPIO4 range.
	ErrInvalidPin deviceError = "M5PM1 GPIO must be between 0 and 4"
	// ErrInvalidPWMPin reports a pin that has no M5PM1 PWM channel.
	ErrInvalidPWMPin deviceError = "M5PM1 PWM is available only on GPIO3 and GPIO4"
	// ErrInvalidPWMFrequency reports a zero PWM counter frequency.
	ErrInvalidPWMFrequency deviceError = "M5PM1 PWM frequency must be nonzero"
	// ErrInvalidPWMDuty reports a value outside the PWM counter's 12-bit range.
	ErrInvalidPWMDuty deviceError = "M5PM1 PWM duty must be between 0 and 4095"
)
