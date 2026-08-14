// Package adxl345 drives an Analog Devices ADXL345 accelerometer over I2C.
package adxl345

const (
	// AddressLow is selected when SDO/ALT ADDRESS is grounded.
	AddressLow = uint16(0x53)
	// AddressHigh is selected when SDO/ALT ADDRESS is tied high.
	AddressHigh = uint16(0x1d)

	deviceIDRegister = byte(0x00)
	powerCTLRegister = byte(0x2d)
	dataRegister     = byte(0x32)
	deviceID         = byte(0xe5)
	measurementMode  = byte(0x08)
)

// Bus is the minimal I2C and timing capability required by the sensor.
type Bus interface {
	Tx(address uint16, write, read []byte) error
	DelayMilliseconds(uint32)
}

// Reading is one signed raw acceleration sample. In the default ±2 g,
// 10-bit mode each least-significant bit represents approximately 3.9 mg.
type Reading struct {
	X int16
	Y int16
	Z int16
}

// Device is one ADXL345 attached to a bus.
type Device struct {
	bus     Bus
	address uint16
}

// New binds an ADXL345 at the selected seven-bit I2C address to bus.
func New(bus Bus, address uint16) *Device {
	return &Device{bus: bus, address: address}
}

// Initialize verifies the device ID and leaves standby mode.
func (d *Device) Initialize() error {
	// The data sheet requires 1.1 ms from power-on before communication.
	d.bus.DelayMilliseconds(2)

	response := [1]byte{}
	register := [1]byte{deviceIDRegister}
	if err := d.bus.Tx(d.address, register[:], response[:]); err != nil {
		return err
	}
	if response[0] != deviceID {
		return ErrDeviceID
	}

	command := [2]byte{powerCTLRegister, measurementMode}
	return d.bus.Tx(d.address, command[:], nil)
}

func decode(low, high byte) int16 {
	return int16(uint16(low) | uint16(high)<<8)
}

// Read returns one coherent X, Y, and Z sample using a single burst read.
func (d *Device) Read() (Reading, error) {
	register := [1]byte{dataRegister}
	response := [6]byte{}
	if err := d.bus.Tx(d.address, register[:], response[:]); err != nil {
		return Reading{}, err
	}
	return Reading{
		X: decode(response[0], response[1]),
		Y: decode(response[2], response[3]),
		Z: decode(response[4], response[5]),
	}, nil
}

type sensorError string

func (e sensorError) Error() string { return string(e) }

// ErrDeviceID reports that the device at the selected address is not an
// ADXL345.
const ErrDeviceID sensorError = "adxl345 device id mismatch"
