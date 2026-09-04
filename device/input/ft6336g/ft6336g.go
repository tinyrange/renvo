// Package ft6336g implements bounded register transactions and touch-report
// decoding for the FocalTech FT6336G controller.
package ft6336g

const (
	// Address is the fixed seven-bit I2C address used by FT6336G.
	Address = uint16(0x38)

	RawMinimumX = 5
	RawMaximumX = 475
	RawMinimumY = 5
	RawMaximumY = 795

	LogicalWidth  = 480
	LogicalHeight = 800

	maximumPoints = 2
)

const (
	registerDeviceMode    = uint8(0x00)
	registerTouchStatus   = uint8(0x02)
	registerCipher        = uint8(0xa3)
	registerInterruptMode = uint8(0xa4)
)

// RegisterDevice is the bounded I2C register capability used by the driver.
type RegisterDevice interface {
	ReadAt(data []byte, register uint8) (int, error)
	WriteAt(data []byte, register uint8) (int, error)
}

// Identity contains the six identification and mode registers beginning at
// 0xa3. Firmware and Vendor correspond to registers 0xa6 and 0xa8.
type Identity struct {
	Cipher        byte
	InterruptMode byte
	PowerMode     byte
	Firmware      byte
	Release       byte
	Vendor        byte
}

// Point is one decoded contact normalized into the 480x800 visible space.
type Point struct {
	X, Y       int
	RawX, RawY int
	ID         byte
	Event      byte
}

// Report contains at most the two contacts supported by FT6336G.
type Report struct {
	Points [maximumPoints]Point
	Count  int
}

// Device is an address-scoped FT6336G controller.
type Device struct {
	registers   RegisterDevice
	initialized bool
}

// New binds a driver to an I2C register device without touching hardware.
func New(registers RegisterDevice) *Device { return &Device{registers: registers} }

// Initialize selects normal operating mode, validates the identity block, and
// selects polling interrupt behavior as used by M5Stack's reference driver.
func (device *Device) Initialize() (Identity, error) {
	identity := Identity{}
	if _, err := device.registers.WriteAt([]byte{0x00}, registerDeviceMode); err != nil {
		return identity, err
	}
	var data [6]byte
	if _, err := device.registers.ReadAt(data[:], registerCipher); err != nil {
		return identity, err
	}
	identity = Identity{
		Cipher:        data[0],
		InterruptMode: data[1],
		PowerMode:     data[2],
		Firmware:      data[3],
		Release:       data[4],
		Vendor:        data[5],
	}
	if identity.Vendor == 0 {
		return identity, ErrIdentity
	}
	if _, err := device.registers.WriteAt([]byte{0x00}, registerInterruptMode); err != nil {
		return identity, err
	}
	device.initialized = true
	return identity, nil
}

// Read fetches one complete two-contact report in a single bounded register
// transaction. Contacts outside the documented active area are discarded.
func (device *Device) Read() (Report, error) {
	report := Report{}
	if !device.initialized {
		return report, ErrNotInitialized
	}
	var data [13]byte
	if _, err := device.registers.ReadAt(data[:], registerTouchStatus); err != nil {
		return report, err
	}
	count := int(data[0] & 0x0f)
	if count > maximumPoints {
		return report, ErrPointCount
	}
	for index := 0; index < count; index++ {
		offset := 1 + index*6
		rawX := int(data[offset]&0x0f)<<8 | int(data[offset+1])
		rawY := int(data[offset+2]&0x0f)<<8 | int(data[offset+3])
		x, y, valid := Normalize(rawX, rawY)
		if !valid {
			continue
		}
		report.Points[report.Count] = Point{
			X: x, Y: y, RawX: rawX, RawY: rawY,
			ID: data[offset+2] >> 4, Event: data[offset] >> 6,
		}
		report.Count++
	}
	return report, nil
}

// Normalize maps the documented active area to every pixel in the 480x800
// logical panel space. Coordinates outside that area are rejected.
func Normalize(rawX, rawY int) (x, y int, valid bool) {
	if rawX < RawMinimumX || rawX > RawMaximumX || rawY < RawMinimumY || rawY > RawMaximumY {
		return 0, 0, false
	}
	xRange := RawMaximumX - RawMinimumX
	yRange := RawMaximumY - RawMinimumY
	x = ((rawX-RawMinimumX)*(LogicalWidth-1) + xRange/2) / xRange
	y = ((rawY-RawMinimumY)*(LogicalHeight-1) + yRange/2) / yRange
	return x, y, true
}

type touchError string

func (err touchError) Error() string { return string(err) }

const (
	ErrIdentity       touchError = "FT6336G identity is invalid"
	ErrNotInitialized touchError = "FT6336G is not initialized"
	ErrPointCount     touchError = "FT6336G report contains too many contacts"
)
