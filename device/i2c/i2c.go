// Package i2c provides board-independent I2C ports, buses, and controllers.
// Register-oriented peripherals are addressed through Bus.Device; for example,
// bus.Device(0x53).ReadAt(data, 0x00) reads register 0x00 from target 0x53.
package i2c

type busError string

func (e busError) Error() string { return string(e) }

const (
	// ErrBusy reports that a controller or bus line could not be acquired.
	ErrBusy busError = "i2c bus busy"
	// ErrNAK reports that a target did not acknowledge an address or byte.
	ErrNAK busError = "target did not acknowledge the I2C byte"
	// ErrTimeout reports that a clock-stretched line did not become high.
	ErrTimeout busError = "I2C clock line did not become high before the timeout"
	// ErrInvalidAddress reports an address outside the seven-bit I2C range.
	ErrInvalidAddress busError = "I2C target address must be between 0x00 and 0x7f"
)

// OperationError describes the transaction that failed and retains the
// controller error in Err.
type OperationError struct {
	// Operation is read, write, write/read, or bus configuration.
	Operation string
	// Address is the seven-bit target address supplied by the caller.
	Address uint16
	// Register is the eight-bit register used by ReadAt or WriteAt.
	Register uint8
	// HasRegister reports whether Register is meaningful for this operation.
	HasRegister bool
	// Err is the controller or validation error that caused the failure.
	Err error
}

// Error reports the operation, target address, optional register, and cause.
func (e *OperationError) Error() string {
	text := "i2c " + e.Operation + " at address " + hexAddress(e.Address)
	if e.HasRegister {
		text += ", register " + hexByte(e.Register)
	}
	if e.Err != nil {
		text += ": " + e.Err.Error()
	}
	return text
}

// Unwrap returns the controller error that caused the operation to fail.
func (e *OperationError) Unwrap() error { return e.Err }

// Controller is the capability supplied by hardware or software I2C engines.
type Controller interface {
	Configure() error
	Tx(address uint16, write, read []byte) error
}

// Port is a board-defined I2C connector. Its fields remain private so
// applications cannot accidentally substitute pins or controller ownership.
type Port struct {
	controller Controller
	delay      Delay
}

// DefinePort constructs a valid board port around controller.
func DefinePort(controller Controller, delay Delay) Port {
	return Port{controller: controller, delay: delay}
}

// Bus is a configured view of a board port.
type Bus struct {
	controller  Controller
	delay       Delay
	initialized bool
	err         error
}

// Device is an address-scoped I2C target. Its ReadAt and WriteAt methods use
// an eight-bit register address and follow the argument order of io.ReaderAt
// and io.WriterAt: data first, offset second.
type Device struct {
	bus     *Bus
	address uint16
}

// New binds an I2C bus to a named board connector. Hardware initialization is
// deferred to the first transaction.
func New(port Port) *Bus { return &Bus{controller: port.controller, delay: port.delay} }

func (b *Bus) initialize() error {
	if b.initialized {
		return b.err
	}
	b.err = b.controller.Configure()
	b.initialized = true
	return b.err
}

// Tx performs an optional write followed by an optional repeated-start read.
func (b *Bus) Tx(address uint16, write, read []byte) error {
	operation := "transaction"
	if len(write) == 0 {
		operation = "read"
	} else if len(read) == 0 {
		operation = "write"
	} else {
		operation = "write/read"
	}
	return b.transaction(operation, address, 0, false, write, read)
}

func (b *Bus) transaction(operation string, address uint16, register uint8, hasRegister bool, write, read []byte) error {
	if address > 0x7f {
		return &OperationError{Operation: operation, Address: address, Register: register, HasRegister: hasRegister, Err: ErrInvalidAddress}
	}
	if err := b.initialize(); err != nil {
		return &OperationError{Operation: "configure bus for " + operation, Address: address, Register: register, HasRegister: hasRegister, Err: err}
	}
	if err := b.controller.Tx(address, write, read); err != nil {
		return &OperationError{Operation: operation, Address: address, Register: register, HasRegister: hasRegister, Err: err}
	}
	return nil
}

// Write performs a write-only transaction.
// Deprecated: use b.Device(address).Write(data) to keep address and data roles
// distinct at the call site.
func (b *Bus) Write(address uint16, data []byte) error { return b.Tx(address, data, nil) }

// Read performs a read-only transaction.
// Deprecated: use b.Device(address).Read(data) to keep address and data roles
// distinct at the call site.
func (b *Bus) Read(address uint16, data []byte) error { return b.Tx(address, nil, data) }

// Device returns an address-scoped target on the bus. Prefer its ReadAt and
// WriteAt methods for register-oriented devices: they make the device address
// and register address distinct in both source code and errors.
func (b *Bus) Device(address uint16) *Device { return &Device{bus: b, address: address} }

// Read reads len(data) bytes without first selecting a register. On success it
// returns len(data); on failure it returns zero and a contextual error.
func (d *Device) Read(data []byte) (int, error) {
	if err := d.bus.transaction("read", d.address, 0, false, nil, data); err != nil {
		return 0, err
	}
	return len(data), nil
}

// Write writes data without first selecting a register. On success it returns
// len(data); on failure it returns zero and a contextual error.
func (d *Device) Write(data []byte) (int, error) {
	if err := d.bus.transaction("write", d.address, 0, false, data, nil); err != nil {
		return 0, err
	}
	return len(data), nil
}

// ReadAt selects register, issues a repeated start, and reads len(data) bytes.
// It is the usual operation for reading registers from an I2C peripheral.
func (d *Device) ReadAt(data []byte, register uint8) (int, error) {
	command := [1]byte{register}
	if err := d.bus.transaction("read", d.address, register, true, command[:], data); err != nil {
		return 0, err
	}
	return len(data), nil
}

// WriteAt writes register followed by data in one transaction. It is the usual
// operation for changing registers on an I2C peripheral.
func (d *Device) WriteAt(data []byte, register uint8) (int, error) {
	command := make([]byte, len(data)+1)
	command[0] = register
	copy(command[1:], data)
	if err := d.bus.transaction("write", d.address, register, true, command, nil); err != nil {
		return 0, err
	}
	return len(data), nil
}

// DelayMilliseconds exposes the board clock associated with this port to
// transaction-level device drivers that require command settling time.
func (b *Bus) DelayMilliseconds(milliseconds uint32) {
	b.delay.DelayMilliseconds(milliseconds)
}

// OpenDrainPin is the GPIO capability required by a software controller.
type OpenDrainPin interface {
	ConfigureOpenDrain() error
	PullLow()
	Release()
	High() bool
}

// Delay is the timing capability required by a software controller.
type Delay interface {
	DelayMicroseconds(uint32)
	DelayMilliseconds(uint32)
}

// BitBang is a deterministic standard-mode software I2C controller.
type BitBang struct {
	sda, scl   OpenDrainPin
	clock      Delay
	halfPeriod uint32
}

// NewBitBang constructs a software controller. frequency is rounded down to a
// whole-microsecond half period and capped at a safe nonzero value.
func NewBitBang(sda, scl OpenDrainPin, clock Delay, frequency uint32) BitBang {
	halfPeriod := uint32(1)
	if frequency != 0 && frequency < 500000 {
		halfPeriod = 500000 / frequency
	}
	return BitBang{sda: sda, scl: scl, clock: clock, halfPeriod: halfPeriod}
}

func (b *BitBang) pause() { b.clock.DelayMicroseconds(b.halfPeriod) }

func (b *BitBang) waitHigh(pin OpenDrainPin) bool {
	for attempt := 0; attempt < 1000; attempt++ {
		if pin.High() {
			return true
		}
		b.pause()
	}
	return false
}

// Configure initializes both open-drain lines and recovers a target left in a
// partial byte by an interrupted transaction.
func (b *BitBang) Configure() error {
	if err := b.sda.ConfigureOpenDrain(); err != nil {
		return err
	}
	if err := b.scl.ConfigureOpenDrain(); err != nil {
		return err
	}
	for pulse := 0; pulse < 9; pulse++ {
		b.scl.PullLow()
		b.pause()
		b.scl.Release()
		if !b.waitHigh(b.scl) {
			return ErrTimeout
		}
		b.pause()
	}
	b.stop()
	return nil
}

func (b *BitBang) start() error {
	b.sda.Release()
	b.scl.Release()
	if !b.waitHigh(b.scl) {
		return ErrTimeout
	}
	if !b.sda.High() {
		return ErrBusy
	}
	b.pause()
	b.sda.PullLow()
	b.pause()
	b.scl.PullLow()
	return nil
}

func (b *BitBang) stop() {
	b.sda.PullLow()
	b.pause()
	b.scl.Release()
	_ = b.waitHigh(b.scl)
	b.pause()
	b.sda.Release()
	b.pause()
}

func (b *BitBang) writeByte(value byte) error {
	for mask := byte(0x80); mask != 0; mask >>= 1 {
		if value&mask == 0 {
			b.sda.PullLow()
		} else {
			b.sda.Release()
		}
		b.pause()
		b.scl.Release()
		if !b.waitHigh(b.scl) {
			return ErrTimeout
		}
		b.pause()
		b.scl.PullLow()
	}
	b.sda.Release()
	b.pause()
	b.scl.Release()
	if !b.waitHigh(b.scl) {
		return ErrTimeout
	}
	b.pause()
	acknowledged := !b.sda.High()
	b.scl.PullLow()
	if !acknowledged {
		return ErrNAK
	}
	return nil
}

func (b *BitBang) readByte(acknowledge bool) (byte, error) {
	value := byte(0)
	b.sda.Release()
	for bit := 0; bit < 8; bit++ {
		value <<= 1
		b.pause()
		b.scl.Release()
		if !b.waitHigh(b.scl) {
			return 0, ErrTimeout
		}
		if b.sda.High() {
			value |= 1
		}
		b.pause()
		b.scl.PullLow()
	}
	if acknowledge {
		b.sda.PullLow()
	} else {
		b.sda.Release()
	}
	b.pause()
	b.scl.Release()
	if !b.waitHigh(b.scl) {
		return 0, ErrTimeout
	}
	b.pause()
	b.scl.PullLow()
	b.sda.Release()
	return value, nil
}

// Tx executes the complete transaction and always attempts a stop after a
// successful start.
func (b *BitBang) Tx(address uint16, write, read []byte) error {
	if address > 0x7f {
		return ErrInvalidAddress
	}
	if err := b.start(); err != nil {
		b.stop()
		return err
	}
	if len(write) != 0 || len(read) == 0 {
		if err := b.writeByte(byte(address << 1)); err != nil {
			b.stop()
			return err
		}
		for index := 0; index < len(write); index++ {
			if err := b.writeByte(write[index]); err != nil {
				b.stop()
				return err
			}
		}
		if len(read) != 0 {
			if err := b.start(); err != nil {
				b.stop()
				return err
			}
		}
	}
	if len(read) != 0 {
		if err := b.writeByte(byte(address<<1 | 1)); err != nil {
			b.stop()
			return err
		}
		for index := 0; index < len(read); index++ {
			value, err := b.readByte(index+1 < len(read))
			if err != nil {
				b.stop()
				return err
			}
			read[index] = value
		}
	}
	b.stop()
	return nil
}

func hexByte(value byte) string {
	digits := "0123456789abcdef"
	text := []byte{'0', 'x', digits[value>>4], digits[value&15]}
	return string(text)
}

func hexAddress(value uint16) string {
	if value <= 0xff {
		return hexByte(byte(value))
	}
	digits := "0123456789abcdef"
	text := []byte{'0', 'x', digits[value>>12], digits[value>>8&15], digits[value>>4&15], digits[value&15]}
	return string(text)
}
