//go:build !renvo

package runtime

import "testing"

func TestNativeEntryBoundaries(t *testing.T) {
	n, err := NewNative(16384)
	if err != nil {
		t.Skip(err)
	}
	defer n.Close()
	var b Builder
	b.Store(1, b.Binary(Add, b.Load(0), b.Constant(7)))
	entry, err := n.Compile(b.Finish(2), 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, offset := range []int{-16, -1, entry + 1, entry + 16, 16384, int(^uint(0) >> 1)} {
		state := []uint64{9, 42}
		if err := n.Call(offset, state); err == nil || state[0] != 9 || state[1] != 42 {
			t.Fatalf("invalid offset %d changed state or executed: %v %v", offset, state, err)
		}
	}
	for _, state := range [][]uint64{nil, {9}, {9, 42, 23}} {
		if err := n.Call(entry, state); err == nil {
			t.Fatalf("accepted wrong size %d", len(state))
		}
	}
	state := []uint64{9, 42}
	if err := n.Call(entry, state); err != nil || state[1] != 16 {
		t.Fatalf("valid entry: %v %v", state, err)
	}
	// An unsuccessful installation must not corrupt the existing entry table.
	for i := 0; i < 1000; i++ {
		if _, err = n.Compile(b.Finish(2), 2); err != nil {
			break
		}
	}
	if err == nil {
		t.Fatal("arena did not exhaust")
	}
	if err := n.Call(entry, state); err != nil || state[1] != 16 {
		t.Fatal("exhaustion damaged first entry", err)
	}
	if err := n.Close(); err != nil {
		t.Fatal(err)
	}
	if err := n.Call(entry, state); err == nil {
		t.Fatal("closed arena executed")
	}
}
