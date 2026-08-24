// Package miniscale drives the M5Stack Unit Mini Scales over I2C.
package miniscale

import "math"

const (
	// DefaultAddress is the unit's factory seven-bit I2C address.
	DefaultAddress = uint16(0x26)

	rawADCRegister            = byte(0x00)
	weightRegister            = byte(0x10)
	buttonRegister            = byte(0x20)
	ledRegister               = byte(0x30)
	gapRegister               = byte(0x40)
	offsetRegister            = byte(0x50)
	weightHundredthsRegister  = byte(0x60)
	lowPassFilterRegister     = byte(0x80)
	averageFilterRegister     = byte(0x81)
	exponentialFilterRegister = byte(0x82)
	firmwareVersionRegister   = byte(0xfe)
)

// Bus is the minimal I2C and timing capability required by the unit.
type Bus interface {
	Tx(address uint16, write, read []byte) error
	DelayMilliseconds(uint32)
}

// Device is one Unit Mini Scales attached to an I2C bus.
type Device struct {
	bus     Bus
	address uint16
}

// New binds a Unit Mini Scales at its factory address to bus.
func New(bus Bus) *Device { return NewAt(bus, DefaultAddress) }

// NewAt binds a Unit Mini Scales at address to bus.
func NewAt(bus Bus, address uint16) *Device {
	return &Device{bus: bus, address: address}
}

func (d *Device) read(register byte, data []byte) error {
	command := [1]byte{register}
	return d.bus.Tx(d.address, command[:], data)
}

func (d *Device) write(register byte, data []byte) error {
	command := make([]byte, len(data)+1)
	command[0] = register
	copy(command[1:], data)
	return d.bus.Tx(d.address, command, nil)
}

func decodeInt32(data *[4]byte) int32 {
	bits := uint32(data[0]) |
		uint32(data[1])<<8 |
		uint32(data[2])<<16 |
		uint32(data[3])<<24
	return int32(bits)
}

func decodeFloat32(data *[4]byte) float32 {
	bits := uint32(data[0]) |
		uint32(data[1])<<8 |
		uint32(data[2])<<16 |
		uint32(data[3])<<24
	return math.Float32frombits(bits)
}

func encodeFloat32(value float32) [4]byte {
	bits := math.Float32bits(value)
	return [4]byte{byte(bits), byte(bits >> 8), byte(bits >> 16), byte(bits >> 24)}
}

// FirmwareVersion reads the firmware revision reported by the unit.
func (d *Device) FirmwareVersion() (byte, error) {
	response := [1]byte{}
	err := d.read(firmwareVersionRegister, response[:])
	return response[0], err
}

// ReadRaw returns the signed HX711 ADC sample reported by the unit.
func (d *Device) ReadRaw() (int32, error) {
	response := [4]byte{}
	if err := d.read(rawADCRegister, response[:]); err != nil {
		return 0, err
	}
	return decodeInt32(&response), nil
}

// ReadWeight returns the calibrated weight in grams.
func (d *Device) ReadWeight() (float32, error) {
	response := [4]byte{}
	if err := d.read(weightRegister, response[:]); err != nil {
		return 0, err
	}
	return decodeFloat32(&response), nil
}

// ReadWeightHundredths returns the calibrated weight in hundredths of a gram.
// It preserves the unit's signed fixed-point register without float rounding.
func (d *Device) ReadWeightHundredths() (int32, error) {
	response := [4]byte{}
	if err := d.read(weightHundredthsRegister, response[:]); err != nil {
		return 0, err
	}
	return decodeInt32(&response), nil
}

// Pressed reports whether the unit's own button is currently pressed.
func (d *Device) Pressed() (bool, error) {
	response := [1]byte{}
	if err := d.read(buttonRegister, response[:]); err != nil {
		return false, err
	}
	return response[0] == 0, nil
}

// SetLED changes the unit's integrated RGB status LED.
func (d *Device) SetLED(red, green, blue byte) error {
	color := [3]byte{red, green, blue}
	return d.write(ledRegister, color[:])
}

// Gap returns the current ADC counts-per-gram calibration value.
func (d *Device) Gap() (float32, error) {
	response := [4]byte{}
	if err := d.read(gapRegister, response[:]); err != nil {
		return 0, err
	}
	return decodeFloat32(&response), nil
}

// SetGap changes the ADC counts-per-gram calibration value.
func (d *Device) SetGap(value float32) error {
	data := encodeFloat32(value)
	if err := d.write(gapRegister, data[:]); err != nil {
		return err
	}
	d.bus.DelayMilliseconds(100)
	return nil
}

// Tare makes the current load the zero offset.
func (d *Device) Tare() error {
	command := [1]byte{1}
	if err := d.write(offsetRegister, command[:]); err != nil {
		return err
	}
	d.bus.DelayMilliseconds(100)
	return nil
}

// SetFilters configures the unit's low-pass, averaging, and exponential
// filters. average must be 0..50 and exponential must be 0..99.
func (d *Device) SetFilters(lowPass bool, average, exponential byte) error {
	if average > 50 || exponential > 99 {
		return ErrInvalidFilter
	}
	lowPassValue := byte(0)
	if lowPass {
		lowPassValue = 1
	}
	values := [3]byte{lowPassValue, average, exponential}
	for index := 0; index < len(values); index++ {
		if err := d.write(lowPassFilterRegister+byte(index), values[index:index+1]); err != nil {
			return err
		}
	}
	return nil
}

type scaleError string

func (e scaleError) Error() string { return string(e) }

// ErrInvalidFilter reports a filter value outside the unit firmware's range.
const ErrInvalidFilter scaleError = "mini scales filter setting is out of range"
