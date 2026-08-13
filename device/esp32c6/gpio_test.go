package esp32c6

import "testing"

func TestGPIORegisterAddresses(t *testing.T) {
	tests := []struct {
		pin          uint8
		ioMux        uintptr
		outputSelect uintptr
	}{
		{1, 0x60090008, 0x60091558},
		{2, 0x6009000c, 0x6009155c},
		{7, 0x60090020, 0x60091570},
		{9, 0x60090028, 0x60091578},
		{19, 0x60090050, 0x600915a0},
		{20, 0x60090054, 0x600915a4},
	}
	for _, test := range tests {
		pin := GPIO(test.pin)
		if got := pin.ioMux(); got != test.ioMux {
			t.Errorf("GPIO%d IO_MUX address = %#x, want %#x", test.pin, got, test.ioMux)
		}
		if got := pin.outputSelect(); got != test.outputSelect {
			t.Errorf("GPIO%d output-select address = %#x, want %#x", test.pin, got, test.outputSelect)
		}
	}
}
