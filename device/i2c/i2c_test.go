package i2c

import (
	"errors"
	"testing"
)

type fakeController struct {
	configured int
	address    uint16
	writes     int
	reads      int
	err        error
	writeData  []byte
}

type fakeDelay struct{}

func (*fakeDelay) DelayMicroseconds(uint32) {}
func (*fakeDelay) DelayMilliseconds(uint32) {}

func (c *fakeController) Configure() error { c.configured++; return nil }
func (c *fakeController) Tx(address uint16, write, read []byte) error {
	c.address, c.writes, c.reads = address, len(write), len(read)
	c.writeData = append(c.writeData[:0], write...)
	return c.err
}

func TestDeviceReadAtAndWriteAtSeparateAddressFromRegister(t *testing.T) {
	controller := &fakeController{}
	device := New(DefinePort(controller, &fakeDelay{})).Device(0x53)
	result := make([]byte, 2)
	if n, err := device.ReadAt(result, 0x32); err != nil || n != 2 {
		t.Fatalf("ReadAt = %d, %v", n, err)
	}
	if controller.address != 0x53 || controller.reads != 2 || string(controller.writeData) != string([]byte{0x32}) {
		t.Fatalf("read transaction = address %#x, write %v, read %d", controller.address, controller.writeData, controller.reads)
	}
	if n, err := device.WriteAt([]byte{0x08}, 0x2d); err != nil || n != 1 {
		t.Fatalf("WriteAt = %d, %v", n, err)
	}
	if controller.address != 0x53 || controller.reads != 0 || string(controller.writeData) != string([]byte{0x2d, 0x08}) {
		t.Fatalf("write transaction = address %#x, write %v, read %d", controller.address, controller.writeData, controller.reads)
	}
}

func TestDeviceErrorsIdentifyOperationAddressRegisterAndCause(t *testing.T) {
	controller := &fakeController{err: ErrNAK}
	device := New(DefinePort(controller, &fakeDelay{})).Device(0x53)
	_, err := device.ReadAt(make([]byte, 1), 0x00)
	if err == nil || err.Error() != "i2c read at address 0x53, register 0x00: target did not acknowledge the I2C byte" {
		t.Fatalf("ReadAt error = %v", err)
	}
	if !errors.Is(err, ErrNAK) {
		t.Fatalf("ReadAt error does not wrap ErrNAK: %v", err)
	}
}

func TestInvalidAddressIsReportedBeforeControllerConfiguration(t *testing.T) {
	controller := &fakeController{}
	device := New(DefinePort(controller, &fakeDelay{})).Device(0x80)
	_, err := device.Read(make([]byte, 1))
	if !errors.Is(err, ErrInvalidAddress) || controller.configured != 0 {
		t.Fatalf("invalid address = %v, configured %d", err, controller.configured)
	}
}

func TestBoardPortConstructsLazyBus(t *testing.T) {
	controller := &fakeController{}
	bus := New(DefinePort(controller, &fakeDelay{}))
	if controller.configured != 0 {
		t.Fatal("constructor touched hardware")
	}
	read := make([]byte, 3)
	if err := bus.Tx(0x58, []byte{1, 2}, read); err != nil {
		t.Fatal(err)
	}
	if controller.configured != 1 || controller.address != 0x58 || controller.writes != 2 || controller.reads != 3 {
		t.Fatalf("controller = %+v", controller)
	}
}
