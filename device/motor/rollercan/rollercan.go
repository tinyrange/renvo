// Package rollercan controls the M5Stack Unit RollerCAN through its Grove I2C
// register protocol. New only binds the bus; enabling output is explicit.
// Speed and position use hundredths of RPM and degrees, current uses hundredths
// of a milliamp. These are motor phase currents, not USB supply current limits.
package rollercan

// Bus is the transaction and settling-time capability supplied by device/i2c.Bus.
type Bus interface {
	Tx(address uint16, write, read []byte) error
	DelayMilliseconds(uint32)
}

const DefaultAddress = uint16(0x64)

type driverError string

func (e driverError) Error() string { return string(e) }

const (
	ErrRange    driverError = "rollercan: value outside protocol range"
	ErrFirmware driverError = "rollercan: operation requires firmware v3 or newer"
)

// Mode selects the motor control loop. EncoderMode provides the haptic dial.
type Mode byte

const (
	SpeedMode    Mode = 1
	PositionMode Mode = 2
	CurrentMode  Mode = 3
	EncoderMode  Mode = 4
)

// CANBaudRate configures the CAN interface through I2C.
type CANBaudRate byte

const (
	CAN1Mbps   CANBaudRate = 0
	CAN500Kbps CANBaudRate = 1
	CAN125Kbps CANBaudRate = 2
)

type RGBMode byte

const (
	RGBSystem RGBMode = 0
	RGBUser   RGBMode = 1
)

type Status byte

const (
	Standby Status = 0
	Running Status = 1
	Fault   Status = 2
)

// Faults is a bit mask, so more than one fault may be present.
type Faults byte

const (
	Overvoltage        Faults = 1
	Stalled            Faults = 2
	PositionOutOfRange Faults = 4
)

// PID preserves the unsigned wire values: P and D are scaled by 100000,
// I by 10000000. Readback follows the PID preset selected on the unit; custom
// writes take effect when the unit's custom PID preset (index zero) is selected.
type PID struct{ P, I, D uint32 }

// ExtendedPosition is Turns*360 degrees plus Angle hundredths of a degree.
// Angle is always 0..35999; for example -1 degree is {-1, 35900}.
type ExtendedPosition struct {
	Turns int32
	Angle uint16
}

// Device is one RollerCAN. Serialize access when changing its address.
type Device struct {
	bus     Bus
	address uint16
}

// New binds a unit at the factory address without changing motor state.
func New(bus Bus) *Device { return NewAt(bus, DefaultAddress) }

// NewAt binds a unit at an existing address. Invalid addresses fail on use.
func NewAt(bus Bus, address uint16) *Device { return &Device{bus: bus, address: address} }

// Address returns the address used for subsequent transactions.
func (d *Device) Address() uint16 { return d.address }

func (d *Device) read(reg byte, data []byte) error {
	if d.address < 0x08 || d.address > 0x77 {
		return ErrRange
	}
	command := [1]byte{reg}
	return d.bus.Tx(d.address, command[:], data)
}

func (d *Device) write(reg byte, data []byte) error {
	if d.address < 0x08 || d.address > 0x77 {
		return ErrRange
	}
	// The largest application register write is a 12-byte PID. A fixed buffer
	// avoids allocating a command slice on every control-loop iteration.
	command := [13]byte{reg}
	copy(command[1:], data)
	return d.bus.Tx(d.address, command[:len(data)+1], nil)
}

func (d *Device) readByte(reg byte) (byte, error) {
	data := [1]byte{}
	if err := d.read(reg, data[:]); err != nil {
		return 0, err
	}
	return data[0], nil
}
func (d *Device) writeByte(reg, value byte) error {
	data := [1]byte{value}
	return d.write(reg, data[:])
}
func (d *Device) readBool(reg byte) (bool, error) {
	value, err := d.readByte(reg)
	return value != 0, err
}
func (d *Device) writeBool(reg byte, value bool) error {
	b := byte(0)
	if value {
		b = 1
	}
	return d.writeByte(reg, b)
}
func decode32(data []byte) uint32 {
	return uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
}
func encode32(data []byte, value uint32) {
	data[0] = byte(value)
	data[1] = byte(value >> 8)
	data[2] = byte(value >> 16)
	data[3] = byte(value >> 24)
}
func (d *Device) readInt32(reg byte) (int32, error) {
	data := [4]byte{}
	if err := d.read(reg, data[:]); err != nil {
		return 0, err
	}
	return int32(decode32(data[:])), nil
}
func (d *Device) writeInt32(reg byte, value int32) error {
	data := [4]byte{}
	encode32(data[:], uint32(value))
	return d.write(reg, data[:])
}
func (d *Device) writeLimited(reg byte, value, limit int32) error {
	if value < -limit || value > limit {
		return ErrRange
	}
	return d.writeInt32(reg, value)
}

// Output reports the requested output state; Status reports actual motor state.
func (d *Device) Output() (bool, error) { return d.readBool(0x00) }

// SetOutput enables or disables motor drive. Configure mode and targets first.
func (d *Device) SetOutput(enabled bool) error { return d.writeBool(0x00, enabled) }
func (d *Device) Mode() (Mode, error)          { v, e := d.readByte(0x01); return Mode(v), e }
func (d *Device) SetMode(mode Mode) error {
	if mode < SpeedMode || mode > EncoderMode {
		return ErrRange
	}
	return d.writeByte(0x01, byte(mode))
}
func (d *Device) PositionProtection() (bool, error)        { return d.readBool(0x0a) }
func (d *Device) SetPositionProtection(enabled bool) error { return d.writeBool(0x0a, enabled) }

// ClearStall clears a latched jam fault; it does not disable stall protection.
func (d *Device) ClearStall() error       { return d.writeByte(0x0b, 1) }
func (d *Device) Status() (Status, error) { v, e := d.readByte(0x0c); return Status(v), e }
func (d *Device) Faults() (Faults, error) { v, e := d.readByte(0x0d); return Faults(v), e }

// ButtonModeSwitch reports whether holding the unit's button can change mode.
func (d *Device) ButtonModeSwitch() (bool, error)        { return d.readBool(0x0e) }
func (d *Device) SetButtonModeSwitch(enabled bool) error { return d.writeBool(0x0e, enabled) }
func (d *Device) StallProtection() (bool, error)         { return d.readBool(0x0f) }
func (d *Device) SetStallProtection(enabled bool) error  { return d.writeBool(0x0f, enabled) }

// CANID is independent of the I2C address and accepts the full byte range.
func (d *Device) CANID() (byte, error)   { return d.readByte(0x10) }
func (d *Device) SetCANID(id byte) error { return d.writeByte(0x10, id) }
func (d *Device) CANBaudRate() (CANBaudRate, error) {
	v, e := d.readByte(0x11)
	return CANBaudRate(v), e
}
func (d *Device) SetCANBaudRate(rate CANBaudRate) error {
	if rate > CAN125Kbps {
		return ErrRange
	}
	return d.writeByte(0x11, byte(rate))
}
func (d *Device) Brightness() (byte, error) { return d.readByte(0x12) }

// SetBrightness accepts 0..100 percent.
func (d *Device) SetBrightness(percent byte) error {
	if percent > 100 {
		return ErrRange
	}
	return d.writeByte(0x12, percent)
}

// PositionMaxCurrent returns the position loop limit in 0.01 mA.
func (d *Device) PositionMaxCurrent() (int32, error) { return d.readInt32(0x20) }

// SetPositionMaxCurrent accepts the protocol's signed range -120000..120000.
// Firmware uses the magnitude as the current limit.
func (d *Device) SetPositionMaxCurrent(value int32) error { return d.writeLimited(0x20, value, 120000) }

// RGB returns eight-bit red, green and blue channels in that order.
func (d *Device) RGB() (byte, byte, byte, error) {
	data := [3]byte{}
	if err := d.read(0x30, data[:]); err != nil {
		return 0, 0, 0, err
	}
	return data[2], data[1], data[0], nil
}

// SetRGB writes the wire's BGR order without changing the RGB mode.
func (d *Device) SetRGB(red, green, blue byte) error {
	data := [3]byte{blue, green, red}
	return d.write(0x30, data[:])
}
func (d *Device) RGBMode() (RGBMode, error) { v, e := d.readByte(0x33); return RGBMode(v), e }
func (d *Device) SetRGBMode(mode RGBMode) error {
	if mode > RGBUser {
		return ErrRange
	}
	return d.writeByte(0x33, byte(mode))
}

// Voltage returns input voltage in hundredths of a volt.
func (d *Device) Voltage() (int32, error) { return d.readInt32(0x34) }

// Temperature returns the internal temperature in whole degrees Celsius.
func (d *Device) Temperature() (int32, error) { return d.readInt32(0x38) }

// DialCounter reads the haptic encoder's signed step count (not shaft degrees).
func (d *Device) DialCounter() (int32, error) { return d.readInt32(0x3c) }

// SetDialCounter changes the count used in EncoderMode.
func (d *Device) SetDialCounter(count int32) error { return d.writeInt32(0x3c, count) }

// SpeedTarget returns the target in hundredths of RPM.
func (d *Device) SpeedTarget() (int32, error)      { return d.readInt32(0x40) }
func (d *Device) SetSpeedTarget(value int32) error { return d.writeLimited(0x40, value, 2100000000) }

// SpeedMaxCurrent returns the speed loop limit in 0.01 mA.
func (d *Device) SpeedMaxCurrent() (int32, error) { return d.readInt32(0x50) }

// SetSpeedMaxCurrent accepts -120000..120000; firmware uses its magnitude.
func (d *Device) SetSpeedMaxCurrent(value int32) error { return d.writeLimited(0x50, value, 120000) }

// Speed reads actual speed in hundredths of RPM.
func (d *Device) Speed() (int32, error) { return d.readInt32(0x60) }

// PositionTarget returns the legacy target in hundredths of a degree.
func (d *Device) PositionTarget() (int32, error)      { return d.readInt32(0x80) }
func (d *Device) SetPositionTarget(value int32) error { return d.writeLimited(0x80, value, 2100000000) }

// Position reads the legacy shaft position in hundredths of a degree.
func (d *Device) Position() (int32, error) { return d.readInt32(0x90) }

// CurrentTarget returns the target phase current in 0.01 mA.
func (d *Device) CurrentTarget() (int32, error)      { return d.readInt32(0xb0) }
func (d *Device) SetCurrentTarget(value int32) error { return d.writeLimited(0xb0, value, 120000) }

// Current reads actual phase current in 0.01 mA.
func (d *Device) Current() (int32, error) { return d.readInt32(0xc0) }

func (d *Device) readPID(reg byte) (PID, error) {
	data := [12]byte{}
	if err := d.read(reg, data[:]); err != nil {
		return PID{}, err
	}
	return PID{P: decode32(data[0:4]), I: decode32(data[4:8]), D: decode32(data[8:12])}, nil
}
func (d *Device) writePID(reg byte, pid PID) error {
	data := [12]byte{}
	encode32(data[0:4], pid.P)
	encode32(data[4:8], pid.I)
	encode32(data[8:12], pid.D)
	return d.write(reg, data[:])
}
func (d *Device) SpeedPID() (PID, error)       { return d.readPID(0x70) }
func (d *Device) SetSpeedPID(pid PID) error    { return d.writePID(0x70, pid) }
func (d *Device) PositionPID() (PID, error)    { return d.readPID(0xa0) }
func (d *Device) SetPositionPID(pid PID) error { return d.writePID(0xa0, pid) }

// SaveConfig explicitly persists configuration to flash and allows 100ms to
// settle. Do not call it in a control loop. Volatile targets need not be saved.
func (d *Device) SaveConfig() error { return d.persist(0xf0, 1) }
func (d *Device) persist(reg, value byte) error {
	if err := d.writeByte(reg, value); err != nil {
		return err
	}
	d.bus.DelayMilliseconds(100)
	return nil
}
func (d *Device) FirmwareVersion() (byte, error) { return d.readByte(0xfe) }

// I2CAddress reads the address stored by the unit.
func (d *Device) I2CAddress() (byte, error) { return d.readByte(0xff) }

// SetI2CAddress changes and persists the address (0x08..0x77). The local address
// changes only after an acknowledged write. If ACK is lost during readdressing,
// probe old and new addresses before retrying. This operation writes flash.
func (d *Device) SetI2CAddress(address uint16) error {
	if address < 0x08 || address > 0x77 {
		return ErrRange
	}
	if err := d.writeByte(0xff, byte(address)); err != nil {
		return err
	}
	d.address = address
	d.bus.DelayMilliseconds(100)
	return nil
}

func (d *Device) requireV3() error {
	version, err := d.FirmwareVersion()
	if err != nil {
		return err
	}
	if version < 3 {
		return ErrFirmware
	}
	return nil
}
func (d *Device) readExtended(reg byte) (ExtendedPosition, error) {
	if err := d.requireV3(); err != nil {
		return ExtendedPosition{}, err
	}
	data := [6]byte{}
	if err := d.read(reg, data[:]); err != nil {
		return ExtendedPosition{}, err
	}
	return ExtendedPosition{Turns: int32(decode32(data[:4])), Angle: uint16(data[4]) | uint16(data[5])<<8}, nil
}

// ExtendedPositionTarget reads both target fields in one coherent v3 snapshot.
func (d *Device) ExtendedPositionTarget() (ExtendedPosition, error) { return d.readExtended(0x88) }

// ExtendedPosition reads both actual position fields in one v3 snapshot.
func (d *Device) ExtendedPosition() (ExtendedPosition, error) { return d.readExtended(0x98) }

// SetExtendedPositionTarget writes turns and in-turn angle together (v3+).
func (d *Device) SetExtendedPositionTarget(position ExtendedPosition) error {
	if position.Angle > 35999 {
		return ErrRange
	}
	if err := d.requireV3(); err != nil {
		return err
	}
	data := [6]byte{}
	encode32(data[:4], uint32(position.Turns))
	data[4] = byte(position.Angle)
	data[5] = byte(position.Angle >> 8)
	return d.write(0x88, data[:])
}

// UID returns the STM32's 96-bit identifier in wire byte order (v3+).
func (d *Device) UID() ([12]byte, error) {
	data := [12]byte{}
	if err := d.requireV3(); err != nil {
		return data, err
	}
	if err := d.read(0xe0, data[:]); err != nil {
		return [12]byte{}, err
	}
	return data, nil
}

// DeviceType returns the product identifier (RollerCAN is 2), available in v3+.
func (d *Device) DeviceType() (byte, error) {
	if err := d.requireV3(); err != nil {
		return 0, err
	}
	return d.readByte(0xf4)
}

// StartEncoderCalibration starts calibration, which can move the motor (v3+).
// Poll CalibrationBusy before explicitly saving the result.
func (d *Device) StartEncoderCalibration() error {
	if err := d.requireV3(); err != nil {
		return err
	}
	return d.writeByte(0xf1, 1)
}

// SaveEncoderCalibration applies the calibrated offset and writes flash (v3+).
func (d *Device) SaveEncoderCalibration() error {
	if err := d.requireV3(); err != nil {
		return err
	}
	return d.persist(0xf2, 1)
}
func (d *Device) CalibrationBusy() (bool, error) {
	if err := d.requireV3(); err != nil {
		return false, err
	}
	return d.readBool(0xf3)
}

// ResetForBootloader writes the application reset command. Entering the
// bootloader also requires a bus-low handshake managed by a firmware updater;
// this method alone does not guarantee bootloader entry or update firmware.
func (d *Device) ResetForBootloader() error { return d.writeByte(0xfd, 1) }
