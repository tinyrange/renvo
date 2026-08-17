package bme688

import (
	"errors"
	"testing"
)

type transaction struct {
	address uint16
	write   []byte
	read    int
}

type fakeBus struct {
	registers [256]byte
	seen      []transaction
	delay     uint32
	failAt    int
	err       error
}

func (b *fakeBus) Tx(address uint16, write, read []byte) error {
	b.seen = append(b.seen, transaction{address: address, write: append([]byte(nil), write...), read: len(read)})
	if b.failAt != 0 && len(b.seen) == b.failAt {
		return b.err
	}
	if len(read) != 0 {
		for index := range read {
			read[index] = b.registers[int(write[0])+index]
		}
	} else if len(write) == 2 {
		b.registers[write[0]] = write[1]
	}
	return nil
}

func (b *fakeBus) DelayMilliseconds(milliseconds uint32) { b.delay += milliseconds }

func sampleCalibration() calibration {
	return calibration{
		t1: 27504, t2: 26435, t3: 3,
		p1: 36477, p2: -10685, p3: 88, p4: 10435, p5: -3024,
		p6: 30, p7: 7, p8: -14600, p9: 6000, p10: 30,
		h1: 75, h2: 362, h3: 0, h4: 45, h5: 20, h6: 120, h7: -100,
		gh1: -30, gh2: -2500, gh3: 4,
		resHeatRange: 1, resHeatValue: 20, rangeSwitchError: -2,
	}
}

func putUnsigned16(data *[42]byte, msb, lsb int, value uint16) {
	data[msb], data[lsb] = byte(value>>8), byte(value)
}

func putSigned16(data *[42]byte, msb, lsb int, value int16) {
	putUnsigned16(data, msb, lsb, uint16(value))
}

func encodeCalibration(c calibration) [42]byte {
	var data [42]byte
	putSigned16(&data, 1, 0, c.t2)
	data[2] = byte(c.t3)
	putUnsigned16(&data, 5, 4, c.p1)
	putSigned16(&data, 7, 6, c.p2)
	data[8] = byte(c.p3)
	putSigned16(&data, 11, 10, c.p4)
	putSigned16(&data, 13, 12, c.p5)
	data[14], data[15] = byte(c.p7), byte(c.p6)
	putSigned16(&data, 19, 18, c.p8)
	putSigned16(&data, 21, 20, c.p9)
	data[22] = c.p10
	data[23] = byte(c.h2 >> 4)
	data[24] = byte(c.h2<<4) | byte(c.h1&0x0f)
	data[25] = byte(c.h1 >> 4)
	data[26], data[27], data[28] = byte(c.h3), byte(c.h4), byte(c.h5)
	data[29], data[30] = c.h6, byte(c.h7)
	putUnsigned16(&data, 32, 31, c.t1)
	putSigned16(&data, 34, 33, c.gh2)
	data[35], data[36] = byte(c.gh1), byte(c.gh3)
	data[37] = byte(c.resHeatValue)
	data[39] = c.resHeatRange << 4
	data[41] = byte(c.rangeSwitchError << 4)
	return data
}

func newInitializedBus(variant byte) (*fakeBus, calibration) {
	bus := &fakeBus{}
	calib := sampleCalibration()
	encoded := encodeCalibration(calib)
	bus.registers[chipIDRegister] = chipID
	bus.registers[variantRegister] = variant
	copy(bus.registers[coeff1Register:], encoded[0:23])
	copy(bus.registers[coeff2Register:], encoded[23:37])
	copy(bus.registers[coeff3Register:], encoded[37:42])
	bus.registers[ctrlGas0] = 0x08
	bus.registers[ctrlGas1] = 0xc7
	return bus, calib
}

func TestInitializeLoadsCalibrationAndConfiguresForcedMode(t *testing.T) {
	bus, wantCalib := newInitializedBus(variantGasHigh)
	device := New(bus, AddressHigh)
	if err := device.Initialize(); err != nil {
		t.Fatal(err)
	}
	device.calib.tFine, wantCalib.tFine = 0, 0
	if device.calib != wantCalib {
		t.Fatalf("decoded calibration = %+v, want %+v", device.calib, wantCalib)
	}
	if bus.delay != 10 {
		t.Fatalf("Initialize delay = %d ms, want 10", bus.delay)
	}
	if bus.registers[ctrlHumidity] != 5 || bus.registers[ctrlMeasurement] != 0x44 {
		t.Fatalf("oversampling registers = hum %#x measurement %#x", bus.registers[ctrlHumidity], bus.registers[ctrlMeasurement])
	}
	if bus.registers[resHeatRegister] != 123 || bus.registers[gasWaitRegister] != 89 {
		t.Fatalf("heater registers = resistance %d wait %d", bus.registers[resHeatRegister], bus.registers[gasWaitRegister])
	}
	if bus.registers[ctrlGas0]&0x08 != 0 || bus.registers[ctrlGas1]&0x3f != 0x20 {
		t.Fatalf("gas control registers = %#x %#x", bus.registers[ctrlGas0], bus.registers[ctrlGas1])
	}
	for _, operation := range bus.seen {
		if operation.address != AddressHigh {
			t.Fatalf("transaction address = %#x, want %#x", operation.address, AddressHigh)
		}
	}
}

func putADC20(field *[17]byte, offset int, value uint32) {
	field[offset] = byte(value >> 12)
	field[offset+1] = byte(value >> 4)
	field[offset+2] = byte(value << 4)
}

func TestReadCompensatesHighVariantMeasurement(t *testing.T) {
	bus, _ := newInitializedBus(variantGasHigh)
	device := New(bus, AddressHigh)
	device.variant = variantGasHigh
	device.calib = sampleCalibration()
	bus.registers[ctrlMeasurement] = 0x44
	var field [17]byte
	field[0] = newDataMask
	putADC20(&field, 2, 415148)
	putADC20(&field, 5, 519888)
	humidityADC := uint16(30000)
	field[8], field[9] = byte(humidityADC>>8), byte(humidityADC)
	gasADC := uint16(700)
	field[15] = byte(gasADC >> 2)
	field[16] = byte(gasADC<<6) | 5 | gasValidMask | heaterStableMask
	copy(bus.registers[field0Register:], field[:])

	reading, err := device.Read()
	if err != nil {
		t.Fatal(err)
	}
	want := Reading{Temperature: 2516, Pressure: 79354, Humidity: 54298, GasResistance: 1757900, GasValid: true, HeaterStable: true}
	if reading != want {
		t.Fatalf("Read() = %+v, want %+v", reading, want)
	}
	if bus.delay != 143 {
		t.Fatalf("Read delay = %d ms, want 143", bus.delay)
	}
	if bus.registers[ctrlMeasurement]&0x03 != forcedMode {
		t.Fatalf("measurement mode = %#x, want forced", bus.registers[ctrlMeasurement])
	}
}

func TestLowVariantGasCompensationAndStatus(t *testing.T) {
	device := New(&fakeBus{}, AddressLow)
	device.calib = sampleCalibration()
	var field [17]byte
	putADC20(&field, 2, 415148)
	putADC20(&field, 5, 519888)
	humidityADC := uint16(30000)
	field[8], field[9] = byte(humidityADC>>8), byte(humidityADC)
	gasADC := uint16(700)
	field[13] = byte(gasADC >> 2)
	field[14] = byte(gasADC<<6) | 5 | gasValidMask
	reading := device.compensate(&field)
	if reading.GasResistance != 217244 || !reading.GasValid || reading.HeaterStable {
		t.Fatalf("low-variant gas reading = %+v", reading)
	}
}

func TestReadReportsNoNewDataAfterPolling(t *testing.T) {
	bus := &fakeBus{}
	device := New(bus, AddressHigh)
	if err := device.ReadInto(&Reading{}); err != ErrNoNewData {
		t.Fatalf("ReadInto error = %v, want %v", err, ErrNoNewData)
	}
	if bus.delay != 193 {
		t.Fatalf("polling delay = %d ms, want 193", bus.delay)
	}
}

func TestInitializeRejectsWrongDeviceAndPropagatesBusErrors(t *testing.T) {
	bus := &fakeBus{}
	if err := New(bus, AddressHigh).Initialize(); err != ErrDeviceID {
		t.Fatalf("Initialize error = %v, want %v", err, ErrDeviceID)
	}
	want := errors.New("bus failed")
	bus = &fakeBus{failAt: 1, err: want}
	if err := New(bus, AddressHigh).Initialize(); !errors.Is(err, want) {
		t.Fatalf("Initialize error = %v, want %v", err, want)
	}
}

func TestEncodeGasWait(t *testing.T) {
	cases := []struct {
		duration uint16
		want     byte
	}{{0, 0}, {63, 63}, {64, 80}, {100, 89}, {0xfc0, 0xff}}
	for _, test := range cases {
		if got := encodeGasWait(test.duration); got != test.want {
			t.Errorf("encodeGasWait(%d) = %d, want %d", test.duration, got, test.want)
		}
	}
}
