package miniscale

import (
	"math"
	"testing"
)

type transaction struct {
	address uint16
	write   []byte
	read    int
}

type testBus struct {
	responses map[byte][]byte
	tx        []transaction
	delays    []uint32
}

func (b *testBus) Tx(address uint16, write, read []byte) error {
	entry := transaction{address: address, write: append([]byte(nil), write...), read: len(read)}
	b.tx = append(b.tx, entry)
	if len(read) != 0 {
		copy(read, b.responses[write[0]])
	}
	return nil
}

func (b *testBus) DelayMilliseconds(milliseconds uint32) {
	b.delays = append(b.delays, milliseconds)
}

func TestReadRegistersDecodeSignedLittleEndianValues(t *testing.T) {
	bus := &testBus{responses: map[byte][]byte{
		rawADCRegister:           {0x78, 0x56, 0x34, 0xf2},
		weightHundredthsRegister: {0xc7, 0xcf, 0xff, 0xff},
	}}
	device := New(bus)
	raw, err := device.ReadRaw()
	if err != nil || raw != int32(-231451016) {
		t.Fatalf("ReadRaw() = %d, %v", raw, err)
	}
	weight, err := device.ReadWeightHundredths()
	if err != nil || weight != -12345 {
		t.Fatalf("ReadWeightHundredths() = %d, %v", weight, err)
	}
	if len(bus.tx) != 2 || bus.tx[0].address != DefaultAddress || bus.tx[0].write[0] != rawADCRegister || bus.tx[0].read != 4 {
		t.Fatalf("transactions = %#v", bus.tx)
	}
}

func TestFloatCalibrationAndControlsUseProtocolEncoding(t *testing.T) {
	weightBits := math.Float32bits(-12.5)
	gapBits := math.Float32bits(417.25)
	bus := &testBus{responses: map[byte][]byte{
		weightRegister: {byte(weightBits), byte(weightBits >> 8), byte(weightBits >> 16), byte(weightBits >> 24)},
		gapRegister:    {byte(gapBits), byte(gapBits >> 8), byte(gapBits >> 16), byte(gapBits >> 24)},
		buttonRegister: {0},
	}}
	device := New(bus)
	weight, err := device.ReadWeight()
	if err != nil || weight != -12.5 {
		t.Fatalf("ReadWeight() = %v, %v", weight, err)
	}
	gap, err := device.Gap()
	if err != nil || gap != 417.25 {
		t.Fatalf("Gap() = %v, %v", gap, err)
	}
	pressed, err := device.Pressed()
	if err != nil || !pressed {
		t.Fatalf("Pressed() = %v, %v", pressed, err)
	}
	if err := device.SetGap(-32.5); err != nil {
		t.Fatal(err)
	}
	if err := device.SetLED(1, 2, 3); err != nil {
		t.Fatal(err)
	}
	if err := device.Tare(); err != nil {
		t.Fatal(err)
	}
	wantGap := math.Float32bits(-32.5)
	if got := bus.tx[3].write; len(got) != 5 || got[0] != gapRegister || got[1] != byte(wantGap) || got[4] != byte(wantGap>>24) {
		t.Fatalf("gap command = %v", got)
	}
	if got := bus.tx[4].write; len(got) != 4 || got[0] != ledRegister || got[1] != 1 || got[2] != 2 || got[3] != 3 {
		t.Fatalf("LED command = %v", got)
	}
	if got := bus.tx[5].write; len(got) != 2 || got[0] != offsetRegister || got[1] != 1 {
		t.Fatalf("tare command = %v", got)
	}
	if len(bus.delays) != 2 || bus.delays[0] != 100 || bus.delays[1] != 100 {
		t.Fatalf("delays = %v", bus.delays)
	}
}

func TestSetFiltersChecksRangesAndWritesEachRegister(t *testing.T) {
	bus := &testBus{}
	device := NewAt(bus, 0x27)
	if err := device.SetFilters(true, 20, 40); err != nil {
		t.Fatal(err)
	}
	if len(bus.tx) != 3 {
		t.Fatalf("transactions = %d", len(bus.tx))
	}
	for index, value := range []byte{1, 20, 40} {
		got := bus.tx[index]
		if got.address != 0x27 || len(got.write) != 2 || got.write[0] != lowPassFilterRegister+byte(index) || got.write[1] != value {
			t.Fatalf("transaction %d = %#v", index, got)
		}
	}
	if err := device.SetFilters(true, 51, 0); err != ErrInvalidFilter {
		t.Fatalf("average range error = %v", err)
	}
	if err := device.SetFilters(true, 0, 100); err != ErrInvalidFilter {
		t.Fatalf("exponential range error = %v", err)
	}
}
