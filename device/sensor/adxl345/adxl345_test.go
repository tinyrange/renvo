package adxl345

import "testing"

type transaction struct {
	address uint16
	write   []byte
	read    int
}

type fakeBus struct {
	responses [2][]byte
	count     int
	delay     uint32
	seen      [2]transaction
}

func (b *fakeBus) Tx(address uint16, write, read []byte) error {
	index := b.count
	b.seen[index] = transaction{
		address: address,
		write:   append([]byte(nil), write...),
		read:    len(read),
	}
	for at := range read {
		read[at] = b.responses[index][at]
	}
	b.count++
	return nil
}

func (b *fakeBus) DelayMilliseconds(milliseconds uint32) { b.delay += milliseconds }

func TestInitializeChecksIDAndEnablesMeasurement(t *testing.T) {
	bus := &fakeBus{responses: [2][]byte{{deviceID}, nil}}
	device := New(bus, AddressLow)
	if err := device.Initialize(); err != nil {
		t.Fatal(err)
	}
	if bus.delay != 2 || bus.count != 2 {
		t.Fatalf("Initialize delay/count = %d/%d, want 2/2", bus.delay, bus.count)
	}
	if got := bus.seen[0]; got.address != AddressLow || len(got.write) != 1 || got.write[0] != deviceIDRegister || got.read != 1 {
		t.Fatalf("ID transaction = %#v", got)
	}
	if got := bus.seen[1]; got.address != AddressLow || len(got.write) != 2 || got.write[0] != powerCTLRegister || got.write[1] != measurementMode || got.read != 0 {
		t.Fatalf("measurement transaction = %#v", got)
	}
}

func TestReadUsesOneBurstAndDecodesSignedLittleEndianValues(t *testing.T) {
	bus := &fakeBus{responses: [2][]byte{{0x34, 0x12, 0x00, 0x80, 0xfe, 0xff}, nil}}
	device := New(bus, AddressHigh)
	reading, err := device.Read()
	if err != nil {
		t.Fatal(err)
	}
	if reading != (Reading{X: 0x1234, Y: -32768, Z: -2}) {
		t.Fatalf("Read() = %#v", reading)
	}
	if bus.count != 1 || bus.seen[0].address != AddressHigh || len(bus.seen[0].write) != 1 || bus.seen[0].write[0] != dataRegister || bus.seen[0].read != 6 {
		t.Fatalf("data transaction = %#v", bus.seen[0])
	}
}

func TestInitializeRejectsWrongDevice(t *testing.T) {
	bus := &fakeBus{responses: [2][]byte{{0x00}, nil}}
	if err := New(bus, AddressLow).Initialize(); err != ErrDeviceID {
		t.Fatalf("Initialize() error = %v, want %v", err, ErrDeviceID)
	}
	if bus.count != 1 {
		t.Fatalf("Initialize made %d transactions after bad ID, want 1", bus.count)
	}
}
