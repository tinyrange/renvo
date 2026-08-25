package bits

import "testing"

func TestOnesCount8(t *testing.T) {
	tests := []struct {
		value uint8
		want  int
	}{
		{0x00, 0},
		{0x01, 1},
		{0x55, 4},
		{0x81, 2},
		{0xff, 8},
	}
	for _, test := range tests {
		if got := OnesCount8(test.value); got != test.want {
			t.Fatalf("OnesCount8(%#x) = %d, want %d", test.value, got, test.want)
		}
	}
}
