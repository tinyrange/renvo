package ft6336g

import (
	"errors"
	"testing"
)

type registerAction struct {
	operation byte
	register  uint8
	length    int
	value     byte
}

type fakeRegisters struct {
	actions  []registerAction
	identity [6]byte
	report   [13]byte
	failAt   uint8
}

func (registers *fakeRegisters) ReadAt(data []byte, register uint8) (int, error) {
	registers.actions = append(registers.actions, registerAction{operation: 'r', register: register, length: len(data)})
	if register == registers.failAt {
		return 0, errors.New("injected read failure")
	}
	if register == registerCipher {
		copy(data, registers.identity[:])
	} else if register == registerTouchStatus {
		copy(data, registers.report[:])
	}
	return len(data), nil
}

func (registers *fakeRegisters) WriteAt(data []byte, register uint8) (int, error) {
	value := byte(0)
	if len(data) != 0 {
		value = data[0]
	}
	registers.actions = append(registers.actions, registerAction{operation: 'w', register: register, length: len(data), value: value})
	if register == registers.failAt {
		return 0, errors.New("injected write failure")
	}
	return len(data), nil
}

func TestInitializeUsesExactIdentityAndPollingTransactions(t *testing.T) {
	registers := &fakeRegisters{identity: [6]byte{0x64, 1, 0, 0x12, 0x34, 0x56}, failAt: 0xff}
	device := New(registers)
	identity, err := device.Initialize()
	if err != nil {
		t.Fatal(err)
	}
	if identity.Cipher != 0x64 || identity.Firmware != 0x12 || identity.Vendor != 0x56 {
		t.Fatalf("identity = %+v", identity)
	}
	want := []registerAction{
		{operation: 'w', register: 0x00, length: 1, value: 0x00},
		{operation: 'r', register: 0xa3, length: 6},
		{operation: 'w', register: 0xa4, length: 1, value: 0x00},
	}
	assertActions(t, registers.actions, want)
}

func TestInitializeRejectsZeroVendorAndStopsOnBusError(t *testing.T) {
	registers := &fakeRegisters{failAt: 0xff}
	if _, err := New(registers).Initialize(); err != ErrIdentity {
		t.Fatalf("zero identity error = %v", err)
	}
	registers = &fakeRegisters{failAt: registerCipher}
	if _, err := New(registers).Initialize(); err == nil {
		t.Fatal("identity read failure was ignored")
	}
	if len(registers.actions) != 2 {
		t.Fatalf("actions after error = %v", registers.actions)
	}
}

func TestReadDecodesAndNormalizesTwoContacts(t *testing.T) {
	registers := &fakeRegisters{identity: [6]byte{1, 0, 0, 1, 0, 1}, failAt: 0xff}
	device := New(registers)
	if _, err := device.Initialize(); err != nil {
		t.Fatal(err)
	}
	registers.actions = nil
	registers.report[0] = 2
	encodePoint(registers.report[:], 0, RawMinimumX, RawMinimumY, 1, 2)
	encodePoint(registers.report[:], 1, RawMaximumX, RawMaximumY, 2, 1)
	report, err := device.Read()
	if err != nil {
		t.Fatal(err)
	}
	if report.Count != 2 || report.Points[0].X != 0 || report.Points[0].Y != 0 ||
		report.Points[1].X != LogicalWidth-1 || report.Points[1].Y != LogicalHeight-1 {
		t.Fatalf("report = %+v", report)
	}
	if report.Points[0].ID != 1 || report.Points[0].Event != 2 || report.Points[1].ID != 2 || report.Points[1].Event != 1 {
		t.Fatalf("metadata = %+v", report.Points)
	}
	assertActions(t, registers.actions, []registerAction{{operation: 'r', register: 0x02, length: 13}})
}

func TestReadDiscardsInactiveEdgesAndRejectsCorruptCount(t *testing.T) {
	registers := &fakeRegisters{identity: [6]byte{1, 0, 0, 1, 0, 1}, failAt: 0xff}
	device := New(registers)
	if _, err := device.Initialize(); err != nil {
		t.Fatal(err)
	}
	registers.report[0] = 1
	encodePoint(registers.report[:], 0, RawMinimumX-1, RawMinimumY, 0, 0)
	report, err := device.Read()
	if err != nil || report.Count != 0 {
		t.Fatalf("inactive-edge report = %+v, %v", report, err)
	}
	registers.report[0] = 3
	if _, err := device.Read(); err != ErrPointCount {
		t.Fatalf("corrupt-count error = %v", err)
	}
}

func TestNormalizeCornersCenterAndBounds(t *testing.T) {
	tests := []struct {
		rawX, rawY int
		x, y       int
		valid      bool
	}{
		{RawMinimumX, RawMinimumY, 0, 0, true},
		{RawMaximumX, RawMaximumY, LogicalWidth - 1, LogicalHeight - 1, true},
		{240, 400, 240, 400, true},
		{RawMinimumX - 1, RawMinimumY, 0, 0, false},
		{RawMaximumX, RawMaximumY + 1, 0, 0, false},
	}
	for _, test := range tests {
		x, y, valid := Normalize(test.rawX, test.rawY)
		if x != test.x || y != test.y || valid != test.valid {
			t.Fatalf("Normalize(%d,%d) = %d,%d,%v; want %d,%d,%v", test.rawX, test.rawY, x, y, valid, test.x, test.y, test.valid)
		}
	}
}

func TestReadRequiresInitialization(t *testing.T) {
	if _, err := New(&fakeRegisters{}).Read(); err != ErrNotInitialized {
		t.Fatalf("Read error = %v", err)
	}
}

func encodePoint(report []byte, index, x, y int, id, event byte) {
	offset := 1 + index*6
	report[offset] = event<<6 | byte(x>>8)&0x0f
	report[offset+1] = byte(x)
	report[offset+2] = id<<4 | byte(y>>8)&0x0f
	report[offset+3] = byte(y)
}

func assertActions(t *testing.T, got, want []registerAction) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("actions = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("action %d = %+v, want %+v", index, got[index], want[index])
		}
	}
}
