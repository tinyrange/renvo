package m5pm1

import (
	"errors"
	"testing"
)

type transaction struct {
	write []byte
	read  []byte
	err   error
}

type fakeBus struct {
	transactions []transaction
	index        int
	delays       []uint32
}

func (b *fakeBus) DelayMilliseconds(milliseconds uint32) {
	b.delays = append(b.delays, milliseconds)
}

func (b *fakeBus) Tx(address uint16, write, read []byte) error {
	if address != Address {
		return errors.New("wrong address")
	}
	if b.index >= len(b.transactions) {
		return errors.New("unexpected transaction")
	}
	want := b.transactions[b.index]
	b.index++
	if !sameBytes(write, want.write) {
		return errors.New("wrong write")
	}
	copy(read, want.read)
	return want.err
}

func sameBytes(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func TestIdentifyIsReadOnlyAndValidatesExactID(t *testing.T) {
	bus := &fakeBus{transactions: []transaction{
		{err: errors.New("sleeping")},
		{write: []byte{0x00}, read: []byte{0x50, 0x20}},
	}}
	id, err := New(bus).Identify()
	if err != nil || id != 0x2050 || bus.index != 2 {
		t.Fatalf("Identify() = %#x, %v after %d transactions", id, err, bus.index)
	}
	if len(bus.delays) != 1 || bus.delays[0] != 10 {
		t.Fatalf("wake delays = %v, want [10]", bus.delays)
	}

	bus = &fakeBus{transactions: []transaction{
		{},
		{write: []byte{0x00}, read: []byte{0x50, 0x21}},
	}}
	id, err = New(bus).Identify()
	if id != 0x2150 || err != ErrDeviceID {
		t.Fatalf("wrong Identify() = %#x, %v", id, err)
	}
}

func TestInitializeMakesOnlyCommunicationSafetyWrites(t *testing.T) {
	bus := &fakeBus{transactions: []transaction{
		{},
		{write: []byte{0x00}, read: []byte{0x50, 0x20}},
		{write: []byte{0x09, 0x00}},
		{write: []byte{0x0a, 0x00}},
	}}
	device := New(bus)
	if err := device.Initialize(); err != nil {
		t.Fatal(err)
	}
	if bus.index != len(bus.transactions) {
		t.Fatalf("performed %d transactions, want %d", bus.index, len(bus.transactions))
	}
}

func TestIdentifyAlwaysPerformsWakeHandshake(t *testing.T) {
	bus := &fakeBus{transactions: []transaction{
		{},
		{write: []byte{0x00}, read: []byte{0x50, 0x20}},
		{},
		{write: []byte{0x00}, read: []byte{0x50, 0x20}},
	}}
	device := New(bus)
	if _, err := device.Identify(); err != nil {
		t.Fatal(err)
	}
	if _, err := device.Identify(); err != nil {
		t.Fatal(err)
	}
	if bus.index != 4 || len(bus.delays) != 2 || bus.delays[0] != 10 || bus.delays[1] != 10 {
		t.Fatalf("transactions = %d, delays = %v", bus.index, bus.delays)
	}
}

func TestConfigureOutputEstablishesLatchBeforeDriver(t *testing.T) {
	bus := &fakeBus{transactions: []transaction{
		{write: []byte{0x11}, read: []byte{0x00}},
		{write: []byte{0x11, 0x04}},
		{write: []byte{0x13}, read: []byte{0xff}},
		{write: []byte{0x13, 0xfb}},
		{write: []byte{0x16}, read: []byte{0xff}},
		{write: []byte{0x16, 0xcf}},
		{write: []byte{0x10}, read: []byte{0x00}},
		{write: []byte{0x10, 0x04}},
	}}
	if err := New(bus).ConfigureOutput(Pin2, true); err != nil {
		t.Fatal(err)
	}
	if bus.index != len(bus.transactions) {
		t.Fatalf("performed %d transactions, want %d", bus.index, len(bus.transactions))
	}
}

func TestConfigureOutputUsesSecondFunctionRegisterForPin4(t *testing.T) {
	bus := &fakeBus{transactions: []transaction{
		{write: []byte{0x11}, read: []byte{0x10}},
		{write: []byte{0x11, 0x00}},
		{write: []byte{0x13}, read: []byte{0x10}},
		{write: []byte{0x13, 0x00}},
		{write: []byte{0x17}, read: []byte{0xff}},
		{write: []byte{0x17, 0xfc}},
		{write: []byte{0x10}, read: []byte{0x00}},
		{write: []byte{0x10, 0x10}},
	}}
	if err := New(bus).ConfigureOutput(Pin4, false); err != nil {
		t.Fatal(err)
	}
}

func TestConfigureOutputRejectsInvalidPinWithoutBusAccess(t *testing.T) {
	bus := &fakeBus{}
	if err := New(bus).ConfigureOutput(Pin(5), false); err != ErrInvalidPin {
		t.Fatalf("ConfigureOutput() error = %v", err)
	}
	if bus.index != 0 {
		t.Fatalf("invalid pin performed %d transactions", bus.index)
	}
}

func TestConfigureOutputStopsAtFirstBusError(t *testing.T) {
	want := errors.New("bus failed")
	bus := &fakeBus{transactions: []transaction{
		{write: []byte{0x11}, read: []byte{0x00}},
		{write: []byte{0x11, 0x04}, err: want},
	}}
	if err := New(bus).ConfigureOutput(Pin2, true); err != want {
		t.Fatalf("ConfigureOutput() error = %v, want %v", err, want)
	}
	if bus.index != 2 {
		t.Fatalf("failure performed %d transactions, want 2", bus.index)
	}
}

func TestConfigurePWMDisablesChannelBeforeSelectingPinFunction(t *testing.T) {
	bus := &fakeBus{transactions: []transaction{
		{write: []byte{0x30, 0x00, 0x00}},
		{write: []byte{0x13}, read: []byte{0xff}},
		{write: []byte{0x13, 0xf7}},
		{write: []byte{0x16}, read: []byte{0x00}},
		{write: []byte{0x16, 0xc0}},
		{write: []byte{0x34, 0x88, 0x13}},
	}}
	if err := New(bus).ConfigurePWM(Pin3, 5000); err != nil {
		t.Fatal(err)
	}
	if bus.index != len(bus.transactions) {
		t.Fatalf("performed %d transactions, want %d", bus.index, len(bus.transactions))
	}
}

func TestConfigurePWMUsesPWM1ForPin4(t *testing.T) {
	bus := &fakeBus{transactions: []transaction{
		{write: []byte{0x32, 0x00, 0x00}},
		{write: []byte{0x13}, read: []byte{0x10}},
		{write: []byte{0x13, 0x00}},
		{write: []byte{0x17}, read: []byte{0x00}},
		{write: []byte{0x17, 0x03}},
		{write: []byte{0x34, 0xe8, 0x03}},
	}}
	if err := New(bus).ConfigurePWM(Pin4, 1000); err != nil {
		t.Fatal(err)
	}
}

func TestSetPWMDutyUsesNormalPolarityAndDisableEncoding(t *testing.T) {
	bus := &fakeBus{transactions: []transaction{
		{write: []byte{0x30, 0x00, 0x14}},
		{write: []byte{0x30, 0x00, 0x00}},
	}}
	device := New(bus)
	if err := device.SetPWMDuty(Pin3, 1024); err != nil {
		t.Fatal(err)
	}
	if err := device.SetPWMDuty(Pin3, 0); err != nil {
		t.Fatal(err)
	}
}

func TestPWMRejectsInvalidArgumentsWithoutBusAccess(t *testing.T) {
	bus := &fakeBus{}
	device := New(bus)
	if err := device.ConfigurePWM(Pin2, 5000); err != ErrInvalidPWMPin {
		t.Fatalf("ConfigurePWM(Pin2) error = %v", err)
	}
	if err := device.ConfigurePWM(Pin3, 0); err != ErrInvalidPWMFrequency {
		t.Fatalf("ConfigurePWM(0 Hz) error = %v", err)
	}
	if err := device.SetPWMDuty(Pin4, 4096); err != ErrInvalidPWMDuty {
		t.Fatalf("SetPWMDuty(4096) error = %v", err)
	}
	if bus.index != 0 {
		t.Fatalf("invalid arguments performed %d transactions", bus.index)
	}
}

func TestSetLEDEnabledPreservesOtherPowerBits(t *testing.T) {
	bus := &fakeBus{transactions: []transaction{
		{write: []byte{0x06}, read: []byte{0x0f}},
		{write: []byte{0x06, 0x1f}},
		{write: []byte{0x06}, read: []byte{0x1f}},
		{write: []byte{0x06, 0x0f}},
	}}
	device := New(bus)
	if err := device.SetLEDEnabled(true); err != nil {
		t.Fatal(err)
	}
	if err := device.SetLEDEnabled(false); err != nil {
		t.Fatal(err)
	}
}
