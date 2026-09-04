package m5ioe1

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

func TestInitializeReadsUIDBeforeDisablingIdleSleep(t *testing.T) {
	bus := &fakeBus{transactions: []transaction{
		{write: []byte{0x00}, read: []byte{0x34, 0x12}},
		{write: []byte{0x23, 0x00}},
	}}
	device := New(bus)
	if err := device.Initialize(); err != nil {
		t.Fatal(err)
	}
	if bus.index != len(bus.transactions) {
		t.Fatalf("performed %d transactions, want %d", bus.index, len(bus.transactions))
	}
}

func TestIdentifyIsReadOnly(t *testing.T) {
	bus := &fakeBus{transactions: []transaction{{write: []byte{0x00}, read: []byte{0xcd, 0xab}}}}
	uid, err := New(bus).Identify()
	if err != nil || uid != 0xabcd || bus.index != 1 {
		t.Fatalf("Identify() = %#x, %v after %d transactions", uid, err, bus.index)
	}
}

func TestConfigureHighBankOutputEstablishesLatchBeforeDriver(t *testing.T) {
	bus := &fakeBus{transactions: []transaction{
		{write: []byte{0x06}, read: []byte{0xff}},
		{write: []byte{0x06, 0xef}},
		{write: []byte{0x14}, read: []byte{0xff}},
		{write: []byte{0x14, 0xef}},
		{write: []byte{0x04}, read: []byte{0x00}},
		{write: []byte{0x04, 0x10}},
	}}
	if err := New(bus).ConfigureOutput(Pin13, false); err != nil {
		t.Fatal(err)
	}
	if bus.index != len(bus.transactions) {
		t.Fatalf("performed %d transactions, want %d", bus.index, len(bus.transactions))
	}
}

func TestSetAndReadOutputUsePrintedPinNumbers(t *testing.T) {
	bus := &fakeBus{transactions: []transaction{
		{write: []byte{0x05}, read: []byte{0x00}},
		{write: []byte{0x05, 0x10}},
		{write: []byte{0x05}, read: []byte{0x10}},
	}}
	device := New(bus)
	if err := device.SetOutput(Pin5, true); err != nil {
		t.Fatal(err)
	}
	level, err := device.Output(Pin5)
	if err != nil || !level {
		t.Fatalf("Output() = %t, %v", level, err)
	}
}

func TestOperationsRejectInvalidPinsWithoutBusAccess(t *testing.T) {
	bus := &fakeBus{}
	device := New(bus)
	if err := device.ConfigureOutput(Pin(0), false); err != ErrInvalidPin {
		t.Fatalf("ConfigureOutput() error = %v", err)
	}
	if err := device.SetOutput(Pin(15), false); err != ErrInvalidPin {
		t.Fatalf("SetOutput() error = %v", err)
	}
	if _, err := device.Output(Pin(15)); err != ErrInvalidPin {
		t.Fatalf("Output() error = %v", err)
	}
	if bus.index != 0 {
		t.Fatalf("invalid pins performed %d transactions", bus.index)
	}
}

func TestConfigureOutputStopsAtFirstBusError(t *testing.T) {
	want := errors.New("bus failed")
	bus := &fakeBus{transactions: []transaction{
		{write: []byte{0x05}, read: []byte{0x00}},
		{write: []byte{0x05, 0x04}, err: want},
	}}
	if err := New(bus).ConfigureOutput(Pin3, true); err != want {
		t.Fatalf("ConfigureOutput() error = %v, want %v", err, want)
	}
	if bus.index != 2 {
		t.Fatalf("failure performed %d transactions, want 2", bus.index)
	}
}

func TestConfigureAndSetPWMUsesPaperMonoRGBChannels(t *testing.T) {
	bus := &fakeBus{transactions: []transaction{
		{write: []byte{0x1d, 0x00, 0x00}},
		{write: []byte{0x05}, read: []byte{0xff}},
		{write: []byte{0x05, 0x7f}},
		{write: []byte{0x13}, read: []byte{0xff}},
		{write: []byte{0x13, 0x7f}},
		{write: []byte{0x03}, read: []byte{0x00}},
		{write: []byte{0x03, 0x80}},
		{write: []byte{0x25, 0x88, 0x13}},
		{write: []byte{0x1d, 0x00, 0x88}},
		{write: []byte{0x1b, 0x00, 0x00}},
	}}
	device := New(bus)
	if err := device.ConfigurePWM(Pin8, 5000); err != nil {
		t.Fatal(err)
	}
	if err := device.SetPWMDuty(Pin8, 2048); err != nil {
		t.Fatal(err)
	}
	if err := device.SetPWMDuty(Pin9, 0); err != nil {
		t.Fatal(err)
	}
}

func TestPWMRejectsInvalidArgumentsWithoutBusAccess(t *testing.T) {
	bus := &fakeBus{}
	device := New(bus)
	if err := device.ConfigurePWM(Pin7, 5000); err != ErrInvalidPWMPin {
		t.Fatalf("ConfigurePWM(Pin7) error = %v", err)
	}
	if err := device.ConfigurePWM(Pin8, 0); err != ErrInvalidPWMFrequency {
		t.Fatalf("ConfigurePWM(0 Hz) error = %v", err)
	}
	if err := device.SetPWMDuty(Pin9, 4096); err != ErrInvalidPWMDuty {
		t.Fatalf("SetPWMDuty(4096) error = %v", err)
	}
	if bus.index != 0 {
		t.Fatalf("invalid PWM arguments performed %d transactions", bus.index)
	}
}
