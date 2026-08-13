package sgp30

import "testing"

func TestCRC(t *testing.T) {
	if got := crc(0xbe, 0xef); got != 0x92 {
		t.Fatalf("crc(0xBEEF) = 0x%02x, want 0x92", got)
	}
}
