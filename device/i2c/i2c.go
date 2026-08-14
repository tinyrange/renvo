// Package i2c provides board-independent I2C ports, buses, and controllers.
package i2c

type busError string

func (e busError) Error() string { return string(e) }

const (
	// ErrBusy reports that a controller or bus line could not be acquired.
	ErrBusy busError = "i2c bus busy"
	// ErrNAK reports that a target did not acknowledge an address or byte.
	ErrNAK busError = "i2c not acknowledged"
	// ErrTimeout reports that a clock-stretched line did not become high.
	ErrTimeout busError = "i2c timeout"
)

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
	if err := b.initialize(); err != nil {
		return err
	}
	return b.controller.Tx(address, write, read)
}

// Write performs a write-only transaction.
func (b *Bus) Write(address uint16, data []byte) error { return b.Tx(address, data, nil) }

// Read performs a read-only transaction.
func (b *Bus) Read(address uint16, data []byte) error { return b.Tx(address, nil, data) }

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
		return ErrNAK
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
