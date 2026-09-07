package rollercan

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
)

type transfer struct {
	address uint16
	write   []byte
	read    int
}
type testBus struct {
	transfers    []transfer
	responses    map[byte][]byte
	delays       []uint32
	failRegister byte
	fail         error
}

func (b *testBus) Tx(address uint16, write, read []byte) error {
	b.transfers = append(b.transfers, transfer{address, append([]byte(nil), write...), len(read)})
	copy(read, b.responses[write[0]])
	if write[0] == b.failRegister {
		return b.fail
	}
	return nil
}
func (b *testBus) DelayMilliseconds(ms uint32) { b.delays = append(b.delays, ms) }

// The expected bytes below come from the firmware register layout, including
// reserved gaps and separate target/readback blocks. Do not derive them from
// driver constants or its encoding helpers.
func TestScalarReads(t *testing.T) {
	cases := []struct {
		name string
		reg  byte
		data []byte
		read func(*Device) (any, error)
		want any
	}{
		{"output", 0x00, []byte{1}, func(d *Device) (any, error) { return d.Output() }, true},
		{"mode", 0x01, []byte{4}, func(d *Device) (any, error) { return d.Mode() }, EncoderMode},
		{"position protection", 0x0a, []byte{1}, func(d *Device) (any, error) { return d.PositionProtection() }, true},
		{"status", 0x0c, []byte{2}, func(d *Device) (any, error) { return d.Status() }, Fault},
		{"fault mask", 0x0d, []byte{7}, func(d *Device) (any, error) { return d.Faults() }, Overvoltage | Stalled | PositionOutOfRange},
		{"button", 0x0e, []byte{0}, func(d *Device) (any, error) { return d.ButtonModeSwitch() }, false},
		{"stall protection", 0x0f, []byte{1}, func(d *Device) (any, error) { return d.StallProtection() }, true},
		{"CAN ID", 0x10, []byte{255}, func(d *Device) (any, error) { return d.CANID() }, byte(255)},
		{"CAN baud", 0x11, []byte{2}, func(d *Device) (any, error) { return d.CANBaudRate() }, CAN125Kbps},
		{"brightness", 0x12, []byte{100}, func(d *Device) (any, error) { return d.Brightness() }, byte(100)},
		{"position limit", 0x20, []byte{0xc0, 0xd4, 1, 0}, func(d *Device) (any, error) { return d.PositionMaxCurrent() }, int32(120000)},
		{"RGB mode", 0x33, []byte{1}, func(d *Device) (any, error) { return d.RGBMode() }, RGBUser},
		{"voltage", 0x34, []byte{0xf4, 1, 0, 0}, func(d *Device) (any, error) { return d.Voltage() }, int32(500)},
		{"temperature", 0x38, []byte{0xf6, 0xff, 0xff, 0xff}, func(d *Device) (any, error) { return d.Temperature() }, int32(-10)},
		{"dial", 0x3c, []byte{0, 0, 0, 0x80}, func(d *Device) (any, error) { return d.DialCounter() }, int32(-2147483648)},
		{"speed target", 0x40, []byte{0xc7, 0xcf, 0xff, 0xff}, func(d *Device) (any, error) { return d.SpeedTarget() }, int32(-12345)},
		{"speed limit", 0x50, []byte{0x40, 0x2b, 0xfe, 0xff}, func(d *Device) (any, error) { return d.SpeedMaxCurrent() }, int32(-120000)},
		{"speed", 0x60, []byte{0xd2, 4, 0, 0}, func(d *Device) (any, error) { return d.Speed() }, int32(1234)},
		{"position target", 0x80, []byte{0xa0, 0x8c, 0, 0}, func(d *Device) (any, error) { return d.PositionTarget() }, int32(36000)},
		{"position", 0x90, []byte{0x60, 0x73, 0xff, 0xff}, func(d *Device) (any, error) { return d.Position() }, int32(-36000)},
		{"current target", 0xb0, []byte{0x40, 0x2b, 0xfe, 0xff}, func(d *Device) (any, error) { return d.CurrentTarget() }, int32(-120000)},
		{"current", 0xc0, []byte{0x10, 0x27, 0, 0}, func(d *Device) (any, error) { return d.Current() }, int32(10000)},
		{"version", 0xfe, []byte{3}, func(d *Device) (any, error) { return d.FirmwareVersion() }, byte(3)},
		{"address", 0xff, []byte{0x64}, func(d *Device) (any, error) { return d.I2CAddress() }, byte(0x64)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := &testBus{responses: map[byte][]byte{tc.reg: tc.data}}
			d := New(b)
			if len(b.transfers) != 0 {
				t.Fatal("constructor touched hardware")
			}
			got, err := tc.read(d)
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("read = %v, %v; want %v", got, err, tc.want)
			}
			want := []transfer{{0x64, []byte{tc.reg}, len(tc.data)}}
			if !reflect.DeepEqual(b.transfers, want) {
				t.Fatalf("transfers = %#v", b.transfers)
			}
			// A controller may fill the destination before reporting failure. No
			// partial value should be reported as a valid sensor measurement.
			b.failRegister, b.fail = tc.reg, errors.New("NAK")
			got, err = tc.read(d)
			if err != b.fail || !reflect.ValueOf(got).IsZero() {
				t.Fatalf("failed read = %v, %v", got, err)
			}
		})
	}
}

func TestWrites(t *testing.T) {
	cases := []struct {
		name  string
		write func(*Device) error
		bytes []byte
		delay bool
	}{
		{"disable", func(d *Device) error { return d.SetOutput(false) }, []byte{0x00, 0}, false},
		{"enable", func(d *Device) error { return d.SetOutput(true) }, []byte{0x00, 1}, false},
		{"mode", func(d *Device) error { return d.SetMode(CurrentMode) }, []byte{0x01, 3}, false},
		{"position protection", func(d *Device) error { return d.SetPositionProtection(true) }, []byte{0x0a, 1}, false},
		{"clear stall", func(d *Device) error { return d.ClearStall() }, []byte{0x0b, 1}, false},
		{"button", func(d *Device) error { return d.SetButtonModeSwitch(false) }, []byte{0x0e, 0}, false},
		{"stall protection", func(d *Device) error { return d.SetStallProtection(true) }, []byte{0x0f, 1}, false},
		{"CAN ID", func(d *Device) error { return d.SetCANID(255) }, []byte{0x10, 255}, false},
		{"CAN baud", func(d *Device) error { return d.SetCANBaudRate(CAN500Kbps) }, []byte{0x11, 1}, false},
		{"brightness", func(d *Device) error { return d.SetBrightness(100) }, []byte{0x12, 100}, false},
		{"position limit", func(d *Device) error { return d.SetPositionMaxCurrent(120000) }, []byte{0x20, 0xc0, 0xd4, 1, 0}, false},
		{"RGB", func(d *Device) error { return d.SetRGB(0x12, 0x34, 0x56) }, []byte{0x30, 0x56, 0x34, 0x12}, false},
		{"RGB mode", func(d *Device) error { return d.SetRGBMode(RGBUser) }, []byte{0x33, 1}, false},
		{"dial", func(d *Device) error { return d.SetDialCounter(-2147483648) }, []byte{0x3c, 0, 0, 0, 0x80}, false},
		{"speed target", func(d *Device) error { return d.SetSpeedTarget(-12345) }, []byte{0x40, 0xc7, 0xcf, 0xff, 0xff}, false},
		{"speed limit", func(d *Device) error { return d.SetSpeedMaxCurrent(-120000) }, []byte{0x50, 0x40, 0x2b, 0xfe, 0xff}, false},
		{"speed PID", func(d *Device) error { return d.SetSpeedPID(PID{100000, 10000000, 0xffffffff}) }, []byte{0x70, 0xa0, 0x86, 1, 0, 0x80, 0x96, 0x98, 0, 0xff, 0xff, 0xff, 0xff}, false},
		{"position target", func(d *Device) error { return d.SetPositionTarget(-36000) }, []byte{0x80, 0x60, 0x73, 0xff, 0xff}, false},
		{"position PID", func(d *Device) error { return d.SetPositionPID(PID{100000, 10000000, 0xffffffff}) }, []byte{0xa0, 0xa0, 0x86, 1, 0, 0x80, 0x96, 0x98, 0, 0xff, 0xff, 0xff, 0xff}, false},
		{"current target", func(d *Device) error { return d.SetCurrentTarget(120000) }, []byte{0xb0, 0xc0, 0xd4, 1, 0}, false},
		{"save", func(d *Device) error { return d.SaveConfig() }, []byte{0xf0, 1}, true},
		{"reset", func(d *Device) error { return d.ResetForBootloader() }, []byte{0xfd, 1}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := &testBus{}
			d := NewAt(b, 0x65)
			if err := tc.write(d); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(b.transfers, []transfer{{0x65, tc.bytes, 0}}) {
				t.Fatalf("transfers = %#v", b.transfers)
			}
			if tc.delay && !reflect.DeepEqual(b.delays, []uint32{100}) || !tc.delay && len(b.delays) != 0 {
				t.Fatalf("delays = %v", b.delays)
			}
			b.failRegister, b.fail = tc.bytes[0], errors.New("timeout")
			b.delays = nil
			if err := tc.write(d); err != b.fail {
				t.Fatalf("write error = %v", err)
			}
			if len(b.delays) != 0 {
				t.Fatal("failed write waited")
			}
		})
	}
}

func TestCompositeReads(t *testing.T) {
	b := &testBus{responses: map[byte][]byte{
		0x30: {0x56, 0x34, 0x12},
		0x70: {0xa0, 0x86, 1, 0, 0x80, 0x96, 0x98, 0, 0xff, 0xff, 0xff, 0xff},
		0xa0: {0x78, 0x56, 0x34, 0x12, 0, 0, 0, 0x80, 0, 0, 0, 0},
	}}
	d := New(b)
	r, g, blue, err := d.RGB()
	if err != nil || r != 0x12 || g != 0x34 || blue != 0x56 {
		t.Fatalf("RGB = %x %x %x %v", r, g, blue, err)
	}
	p, err := d.SpeedPID()
	if err != nil || p != (PID{100000, 10000000, 0xffffffff}) {
		t.Fatalf("speed PID = %v %v", p, err)
	}
	p, err = d.PositionPID()
	if err != nil || p != (PID{0x12345678, 0x80000000, 0}) {
		t.Fatalf("position PID = %v %v", p, err)
	}
	if !reflect.DeepEqual(b.transfers, []transfer{{0x64, []byte{0x30}, 3}, {0x64, []byte{0x70}, 12}, {0x64, []byte{0xa0}, 12}}) {
		t.Fatalf("transfers = %#v", b.transfers)
	}
	b.fail = errors.New("bus failure")
	b.failRegister = 0x30
	r, g, blue, err = d.RGB()
	if err != b.fail || r != 0 || g != 0 || blue != 0 {
		t.Fatal("partial RGB returned")
	}
	b.failRegister = 0x70
	p, err = d.SpeedPID()
	if err != b.fail || p != (PID{}) {
		t.Fatal("partial PID returned")
	}
	b.failRegister = 0xa0
	p, err = d.PositionPID()
	if err != b.fail || p != (PID{}) {
		t.Fatal("partial PID returned")
	}
}

func TestAddressChange(t *testing.T) {
	b := &testBus{responses: map[byte][]byte{0xff: {0x77}}}
	d := New(b)
	if err := d.SetI2CAddress(0x77); err != nil {
		t.Fatal(err)
	}
	v, err := d.I2CAddress()
	if err != nil || v != 0x77 || d.Address() != 0x77 {
		t.Fatal("address did not change")
	}
	if !reflect.DeepEqual(b.transfers, []transfer{{0x64, []byte{0xff, 0x77}, 0}, {0x77, []byte{0xff}, 1}}) {
		t.Fatalf("transfers = %#v", b.transfers)
	}
	if !reflect.DeepEqual(b.delays, []uint32{100}) {
		t.Fatalf("delays = %v", b.delays)
	}
	b.failRegister, b.fail = 0xff, errors.New("NAK")
	if err := d.SetI2CAddress(0x08); err != b.fail || d.Address() != 0x77 {
		t.Fatal("failed write changed address")
	}
}

func TestInvalidValuesNeverTouchBus(t *testing.T) {
	b := &testBus{}
	d := New(b)
	actions := []func() error{
		func() error { return d.SetMode(0) }, func() error { return d.SetMode(5) },
		func() error { return d.SetCANBaudRate(3) }, func() error { return d.SetRGBMode(2) }, func() error { return d.SetBrightness(101) },
		func() error { return d.SetSpeedTarget(2100000001) }, func() error { return d.SetSpeedTarget(-2100000001) },
		func() error { return d.SetPositionTarget(2147483647) }, func() error { return d.SetPositionTarget(-2147483648) },
		func() error { return d.SetSpeedMaxCurrent(120001) }, func() error { return d.SetSpeedMaxCurrent(-120001) },
		func() error { return d.SetPositionMaxCurrent(120001) }, func() error { return d.SetPositionMaxCurrent(-120001) },
		func() error { return d.SetCurrentTarget(120001) }, func() error { return d.SetCurrentTarget(-120001) },
		func() error { return d.SetI2CAddress(7) }, func() error { return d.SetI2CAddress(0x78) }, func() error { return d.SetI2CAddress(0x164) },
		func() error { return d.SetExtendedPositionTarget(ExtendedPosition{Angle: 36000}) },
		func() error { _, err := NewAt(b, 0x164).FirmwareVersion(); return err },
		func() error { return NewAt(b, 0).SetOutput(true) },
	}
	for i, action := range actions {
		if err := action(); err != ErrRange {
			t.Fatalf("action %d: %v", i, err)
		}
	}
	if len(b.transfers) != 0 || len(b.delays) != 0 {
		t.Fatal("invalid argument touched hardware")
	}
}

func TestV3Extensions(t *testing.T) {
	cases := []struct {
		name   string
		reg    byte
		data   []byte
		write  []byte
		action func(*Device) error
		delay  bool
	}{
		{"target", 0x88, []byte{0xff, 0xff, 0xff, 0xff, 0x3c, 0x8c}, nil, func(d *Device) error {
			v, e := d.ExtendedPositionTarget()
			if e == nil && v != (ExtendedPosition{-1, 35900}) {
				t.Errorf("target = %v", v)
			}
			return e
		}, false},
		{"actual", 0x98, []byte{0, 0, 0, 0x80, 0x9f, 0x8c}, nil, func(d *Device) error {
			v, e := d.ExtendedPosition()
			if e == nil && v != (ExtendedPosition{-2147483648, 35999}) {
				t.Errorf("actual = %v", v)
			}
			return e
		}, false},
		{"set target", 0x88, nil, []byte{0x88, 0xff, 0xff, 0xff, 0xff, 0x3c, 0x8c}, func(d *Device) error { return d.SetExtendedPositionTarget(ExtendedPosition{-1, 35900}) }, false},
		{"UID", 0xe0, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}, nil, func(d *Device) error {
			v, e := d.UID()
			if e == nil && v != ([12]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}) {
				t.Errorf("UID = %v", v)
			}
			return e
		}, false},
		{"type", 0xf4, []byte{2}, nil, func(d *Device) error {
			v, e := d.DeviceType()
			if e == nil && v != 2 {
				t.Errorf("type = %v", v)
			}
			return e
		}, false},
		{"start calibration", 0xf1, nil, []byte{0xf1, 1}, func(d *Device) error { return d.StartEncoderCalibration() }, false},
		{"save calibration", 0xf2, nil, []byte{0xf2, 1}, func(d *Device) error { return d.SaveEncoderCalibration() }, true},
		{"busy", 0xf3, []byte{1}, nil, func(d *Device) error {
			v, e := d.CalibrationBusy()
			if e == nil && !v {
				t.Error("not busy")
			}
			return e
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := &testBus{responses: map[byte][]byte{0xfe: {3}, tc.reg: tc.data}}
			d := New(b)
			if err := tc.action(d); err != nil {
				t.Fatal(err)
			}
			command := tc.write
			if command == nil {
				command = []byte{tc.reg}
			}
			if !reflect.DeepEqual(b.transfers, []transfer{{0x64, []byte{0xfe}, 1}, {0x64, command, len(tc.data)}}) {
				t.Fatalf("transfers = %#v", b.transfers)
			}
			if tc.delay && !reflect.DeepEqual(b.delays, []uint32{100}) {
				t.Fatal("missing flash delay")
			}
			b.failRegister, b.fail = tc.reg, errors.New("read/write failed")
			if err := tc.action(d); err != b.fail {
				t.Fatalf("operation failure = %v", err)
			}
			b.failRegister = 0xfe
			if err := tc.action(d); err != b.fail {
				t.Fatalf("version failure = %v", err)
			}
			b.fail = nil
			for _, version := range []byte{0, 1, 2} {
				b.responses[0xfe] = []byte{version}
				b.transfers = nil
				if err := tc.action(d); err != ErrFirmware {
					t.Fatalf("legacy firmware error = %v", err)
				}
				if len(b.transfers) != 1 || !bytes.Equal(b.transfers[0].write, []byte{0xfe}) {
					t.Fatal("accessed unsupported register")
				}
			}
		})
	}
}
