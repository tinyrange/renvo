//go:build renvo && browser

package runtime

import "fmt"

// Check both straight-line overwritten stores and every precise publication
// boundary using actual emitted WASM. The second instruction faults only after
// the first instruction's state and progress have become observable.
func browserCheckpointProbe(n *Native) error {
	ops := []Op{
		{Kind: Const, Imm: 11}, {Kind: StoreState, A: 0, Imm: 0},
		{Kind: Const, Imm: 22}, {Kind: StoreState, A: 2, Imm: 0},
		{Kind: LoadState, Imm: 0}, {Kind: StoreState, A: 4, Imm: 1},
		{Kind: Const, Imm: 33}, {Kind: StoreState, A: 6, Imm: 0},
	}
	entry, err := n.Compile(ops, 3)
	if err != nil {
		return err
	}
	state := []uint64{99, 98, 97}
	want := append([]uint64(nil), state...)
	if err = Interpret(ops, want); err != nil {
		return err
	}
	if err = n.Call(entry, state); err != nil {
		return err
	}
	for i := range state {
		if state[i] != want[i] {
			return fmt.Errorf("WASM observed state store mismatch")
		}
	}
	for _, mode := range [3]int{0, 1, 2} {
		var b Builder
		b.MemoryInstructionProgress = true
		b.Checkpoint(3, 0)
		b.Store(0, b.Constant(11))
		b.Checkpoint(3, 1)
		b.Store(0, b.Constant(22))
		b.Checkpoint(3, 2)
		if mode == 0 {
			b.Store(1, b.MemoryLoad(b.Load(2), 8))
		}
		if mode == 1 {
			b.Guard(b.Constant(0))
		}
		if mode == 2 {
			b.LoopContinue(b.Constant(0))
		}
		b.Store(0, b.Constant(33))
		b.Checkpoint(3, 3)
		entry, err = n.CompileMemory(b.FinishMemory(3), 3)
		if err != nil {
			return err
		}
		m := &MemoryContext{}
		state[0], state[1], state[2] = 99, 98, 4096
		if err = n.CallMemory(entry, state, m); err != nil {
			return err
		}
		expectedStatus := uint64(1)
		if mode == 1 {
			expectedStatus = 2
		}
		if mode == 2 {
			expectedStatus = RegionExit
		}
		if state[0] != 22 || state[1] != 98 || m.Retired != 2 || m.wasmMemoryRetired != 0 || m.Status != expectedStatus {
			return fmt.Errorf("WASM precise checkpoint mismatch")
		}
	}
	return nil
}
