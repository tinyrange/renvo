package main

import "testing"

func TestStressRandomIsDeterministicAndNonzero(t *testing.T) {
	state := uint32(0x6d2b79f5)
	first := nextRandom(&state)
	second := nextRandom(&state)
	if first == 0 || second == 0 || first == second {
		t.Fatalf("stress random sequence = %08x, %08x", first, second)
	}
	state = 0x6d2b79f5
	if repeated := nextRandom(&state); repeated != first {
		t.Fatalf("repeated first value = %08x, want %08x", repeated, first)
	}
}
