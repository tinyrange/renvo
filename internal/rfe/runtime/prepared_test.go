package runtime

import "testing"

func TestPreparedIRImmutableAndReusable(t *testing.T) {
	var b Builder
	old := b.Load(0)
	b.Store(0, b.Binary(Add, old, b.Constant(1)))
	b.Store(1, old)
	ops := b.Finish(2)
	prepared, err := Prepare(ops, 2)
	if err != nil {
		t.Fatal(err)
	}
	// Mutating caller-owned IR after preparation must not corrupt validated code.
	for i := range ops {
		ops[i] = Op{Kind: -99}
	}
	var machine IRMachine
	for _, initial := range []uint64{0, 1, 1 << 63, ^uint64(0)} {
		state := []uint64{initial, 123}
		if err := machine.Run(prepared, state); err != nil || state[0] != initial+1 || state[1] != initial {
			t.Fatalf("old-state forwarding/reuse: %x %v", state, err)
		}
	}
	state := []uint64{0, 0}
	var runErr error
	allocations := testing.AllocsPerRun(100, func() { runErr = machine.Run(prepared, state) })
	if runErr != nil || allocations != 0 {
		t.Fatalf("prepared execution allocated: %f, %v", allocations, runErr)
	}
	if state[0] != 101 || state[1] != 100 {
		t.Fatal("scratch retained stale state")
	}
	var other IRMachine
	separate := []uint64{7, 0}
	if err := other.Run(prepared, separate); err != nil || separate[0] != 8 || separate[1] != 7 {
		t.Fatal("prepared block owns state")
	}
}
func TestPreparedIRValidationAndDimensions(t *testing.T) {
	invalid := [][]Op{
		nil,
		{{Kind: LoadState, Imm: 2}},
		{{Kind: Const, Imm: 3}, {Kind: StoreState, A: 2}},
		{{Kind: Const, Imm: 3}, {Kind: StoreState, A: 0}, {Kind: Add, A: 1, B: 0}},
	}
	for _, ops := range invalid {
		if _, err := Prepare(ops, 2); err == nil {
			t.Fatal("prepared invalid IR")
		}
	}
	good, err := Prepare([]Op{{Kind: Const, Imm: 17}, {Kind: StoreState, A: 0, Imm: 0}}, 2)
	if err != nil {
		t.Fatal(err)
	}
	var machine IRMachine
	for _, block := range []*PreparedBlock{nil, {}, good} {
		state := []uint64{99}
		if err := machine.Run(block, state); err == nil || state[0] != 99 {
			t.Fatal("dimension failure changed state")
		}
	}
	if err := machine.Run(&PreparedBlock{}, nil); err == nil {
		t.Fatal("accepted zero-value block")
	}
	var absent *IRMachine
	if err := absent.Run(good, []uint64{1, 2}); err == nil {
		t.Fatal("accepted nil machine")
	}
}
