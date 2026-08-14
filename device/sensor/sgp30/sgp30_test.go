package sgp30

import "testing"

type fakeBus struct {
	response [6]byte
	writes   [2][2]byte
	count    int
	delay    uint32
}

func (b *fakeBus) Tx(address uint16, write, read []byte) error {
	if address != 0x58 {
		return ErrCRC
	}
	if len(write) == 2 {
		b.writes[b.count][0], b.writes[b.count][1] = write[0], write[1]
		b.count++
	}
	for index := range read {
		read[index] = b.response[index]
	}
	return nil
}

func (b *fakeBus) DelayMilliseconds(milliseconds uint32) { b.delay += milliseconds }

func TestCRC(t *testing.T) {
	if got := crc(0xbe, 0xef); got != 0x92 {
		t.Fatalf("crc(0xBEEF) = 0x%02x, want 0x92", got)
	}
}

func TestReadReturnsBothAirQualityWords(t *testing.T) {
	bus := &fakeBus{}
	bus.response = [6]byte{0x01, 0xf4, crc(0x01, 0xf4), 0x00, 0x2a, crc(0x00, 0x2a)}
	device := New(bus)
	reading, err := device.Read()
	if err != nil {
		t.Fatal(err)
	}
	if reading.ECO2 != 500 || reading.TVOC != 42 {
		t.Fatalf("Read() = %+v, want eCO2 500 and TVOC 42", reading)
	}
	if bus.count != 1 || bus.writes[0] != [2]byte{0x20, 0x08} || bus.delay != 15 {
		t.Fatalf("transaction = writes %v delay %d", bus.writes, bus.delay)
	}
}
