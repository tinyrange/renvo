package omnibus

import "testing"

func TestCumulativeStages(t *testing.T) {
	tests := []struct {
		name  string
		run   func() bool
		lo    int32
		hi    int32
		count int
	}{
		{name: "stage0", run: Stage0, lo: 603994284, hi: -2048144777, count: 1},
		{name: "stage1", run: Stage1, lo: -424271508, hi: -2055016039, count: 4},
		{name: "stage2", run: Stage2, lo: 705802163, hi: 915712923, count: 7},
		{name: "run_all", run: RunAll, lo: -3996892, hi: -351219453, count: 11},
	}
	for _, test := range tests {
		if !test.run() {
			t.Fatalf("%s failed", test.name)
		}
		if !Passed(test.lo, test.hi, test.count) {
			t.Fatalf("%s result mismatch: state=%d lo=%d hi=%d completed=%d", test.name, state, signatureLo, signatureHi, completed)
		}
	}
}

func TestRunAllResultBlock(t *testing.T) {
	if !RunAll() {
		t.Fatal("RunAll failed")
	}
	if got := read32(0); got != 1380406354 {
		t.Fatalf("magic = %#x", got)
	}
	if got := read16(offsetVersion); got != 1 {
		t.Fatalf("version = %d", got)
	}
	if got := read16(offsetSize); got != ResultSize {
		t.Fatalf("size = %d", got)
	}
	if got := read32(offsetProfile); got != ProfileCore {
		t.Fatalf("profile = %#x", got)
	}
	if got := read32(offsetState); got != statePassed {
		t.Fatalf("state = %d", got)
	}
	if got := read32(offsetCurrentProbe); got != 304 {
		t.Fatalf("current probe = %d", got)
	}
	if got := read32(offsetCompletedProbes); got != 11 {
		t.Fatalf("completed probes = %d", got)
	}
	if got := read32(offsetSignature); got != -3996892 {
		t.Fatalf("signature low = %#x", got)
	}
	if got := read32(offsetSignature + 4); got != -351219453 {
		t.Fatalf("signature high = %#x", got)
	}
}

func read16(offset int) int {
	return int(renvores[offset]) | int(renvores[offset+1])<<8
}

func read32(offset int) int32 {
	return int32(renvores[offset]) | int32(renvores[offset+1])<<8 | int32(renvores[offset+2])<<16 | int32(renvores[offset+3])<<24
}
