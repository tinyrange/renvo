package backendcompiled

import "testing"

func TestARMImmediateEncoding(t *testing.T) {
	decode := func(encoded int) uint32 {
		payload, shift := uint32(encoded&255), uint((encoded>>8&15)*2)
		return payload>>shift | payload<<((32-shift)&31)
	}
	check := func(value uint32, expected bool) {
		t.Helper()
		encoded := renvoArmAsmImmediate(int(int32(value)))
		if (encoded >= 0) != expected || encoded >= 0 && (encoded > 4095 || decode(encoded) != value) {
			t.Fatalf("value=%08x encoded=%x representable=%v", value, encoded, expected)
		}
	}
	// Every payload/rotation combination is a valid ARM operand. Duplicate
	// representations are fine; the compiler must reproduce the same word.
	for rotation := 0; rotation < 16; rotation++ {
		for payload := 0; payload < 256; payload++ {
			check(decode(rotation<<8|payload), true)
		}
	}
	// Compare arbitrary words with the complete ISA operand set, rather than
	// repeating the compiler's bounded bit-position algorithm.
	state := uint32(0x12345678)
	for sample := 0; sample < 20000; sample++ {
		state = state*1664525 + 1013904223
		expected := false
		for rotation := 0; rotation < 16; rotation++ {
			shift := uint(rotation * 2)
			rolled := state<<shift | state>>((32-shift)&31)
			if rolled <= 255 {
				expected = true
				break
			}
		}
		check(state, expected)
	}
}
