// Package sgp30 drives a Sensirion SGP30 air-quality sensor.
package sgp30

const address = uint16(0x58)

// Bus is the minimal I2C and timing capability required by the sensor.
type Bus interface {
	Tx(address uint16, write, read []byte) error
	DelayMilliseconds(uint32)
}

// Device is one SGP30 attached to a bus.
type Device struct {
	bus Bus
}

// Reading contains one air-quality sample from the sensor.
type Reading struct {
	ECO2 uint16
	TVOC uint16
}

// New binds an SGP30 to bus.
func New(bus Bus) *Device { return &Device{bus: bus} }

func (d *Device) command(code uint16) error {
	data := [2]byte{byte(code >> 8), byte(code)}
	return d.bus.Tx(address, data[:], nil)
}

// Initialize starts the sensor's on-chip air-quality algorithm.
func (d *Device) Initialize() error {
	if err := d.command(0x2003); err != nil {
		return err
	}
	d.bus.DelayMilliseconds(10)
	return nil
}

func crc(first, second byte) byte {
	result := byte(0xff)
	data := [2]byte{first, second}
	for index := 0; index < 2; index++ {
		result ^= data[index]
		for bit := 0; bit < 8; bit++ {
			if result&0x80 != 0 {
				result = result<<1 ^ 0x31
			} else {
				result <<= 1
			}
		}
	}
	return result
}

// Read returns equivalent CO2 in ppm and total volatile organic compounds in
// ppb. Call it once per second to keep the sensor's IAQ algorithm running.
func (d *Device) Read() (Reading, error) {
	if err := d.command(0x2008); err != nil {
		return Reading{}, err
	}
	d.bus.DelayMilliseconds(15)
	response := [6]byte{}
	if err := d.bus.Tx(address, nil, response[:]); err != nil {
		return Reading{}, err
	}
	if crc(response[0], response[1]) != response[2] || crc(response[3], response[4]) != response[5] {
		return Reading{}, ErrCRC
	}
	return Reading{
		ECO2: uint16(response[0])<<8 | uint16(response[1]),
		TVOC: uint16(response[3])<<8 | uint16(response[4]),
	}, nil
}

// Measure returns total volatile organic compounds in parts per billion.
// Deprecated: use Read to obtain both values.
func (d *Device) Measure() (uint16, error) {
	reading, err := d.Read()
	return reading.TVOC, err
}

// sensorError is allocation-free and implements error.
type sensorError string

func (e sensorError) Error() string { return string(e) }

// ErrCRC reports a corrupt sensor response.
const ErrCRC sensorError = "sgp30 crc mismatch"
