//go:build renvo && browser

package runtime

import "fmt"

// BrowserCompilerProbe executes generated modules, not their IR evaluator.
// It checks modular arithmetic/flags against the portable reference and checks
// both successful memory effects and a precise no-write permission exit.
func BrowserCompilerProbe() error {
	n, err := NewNative(1 << 20)
	if err != nil {
		return err
	}
	defer n.Close()
	ops := []Op{{Kind: LoadState, Imm: 0}, {Kind: Const, Imm: 1}, {Kind: Add, A: 0, B: 1}, {Kind: StoreState, A: 2, Imm: 0}, {Kind: ArithmeticFlags, A: 0, B: 1, Imm: 64}, {Kind: StoreState, A: 4, Imm: 1}}
	entry, err := n.Compile(ops, 2)
	if err != nil {
		return err
	}
	for _, seed := range []uint64{0, ^uint64(0), 0x7fffffffffffffff, 0xffffffff} {
		state := []uint64{seed, 0}
		want := append([]uint64(nil), state...)
		if err = Interpret(ops, want); err != nil {
			return err
		}
		if err = n.Call(entry, state); err != nil {
			return err
		}
		if state[0] != want[0] || state[1] != want[1] {
			return fmt.Errorf("compiled WASM arithmetic mismatch")
		}
	}
	if err = browserRecycleProbe(); err != nil {
		return err
	}
	if err = browserGateProbe(); err != nil {
		return err
	}
	if err = browserCheckpointProbe(n); err != nil {
		return err
	}
	if err = browserLoopProbe(); err != nil {
		return err
	}
	if err = browserIntegerProbe(n); err != nil {
		return err
	}
	var b Builder
	b.Checkpoint(3, 0)
	address := b.Load(0)
	b.MemoryStore(address, b.Load(1), 8)
	b.Store(2, b.MemoryLoad(address, 8))
	b.Checkpoint(3, 1)
	entry, err = n.CompileMemory(b.FinishMemory(3), 3)
	if err != nil {
		return err
	}
	var data [4096]byte
	clock, epoch := uint64(5), uint64(3)
	m := &MemoryContext{}
	m.Clock = &clock
	m.Fill(1, &data, 3, &epoch)
	state := []uint64{4096, 0x8877665544332211, 0}
	if err = n.CallMemory(entry, state, m); err != nil {
		return err
	}
	if m.Status != 0 || m.Retired != 1 || state[2] != state[1] || clock != 6 || epoch != 6 {
		return fmt.Errorf("compiled WASM memory mismatch")
	}
	m.Fill(1, &data, 1, &epoch)
	before := data
	state[1] = 99
	if err = n.CallMemory(entry, state, m); err != nil {
		return err
	}
	if m.Status != 1 || m.Retired != 0 || m.Address != 4096 || data != before || clock != 6 || epoch != 6 {
		return fmt.Errorf("compiled WASM memory guard mismatch")
	}
	// Full 64-bit page keys must not alias a low page or a colliding slot.
	// Every rejected guard leaves state, memory and both generations untouched.
	for _, address := range [3]uint64{4096 + (1 << 48), 4096 + 4092, 4096} {
		m.Fill(1, &data, 3, &epoch)
		if address == 4096 {
			m.Fill(NativePageSlots+1, &data, 3, &epoch)
		}
		state[0], state[1] = address, 0x1122334455667788
		previous := append([]uint64(nil), state...)
		before = data
		if err = n.CallMemory(entry, state, m); err != nil {
			return err
		}
		if m.Status != 1 || m.Retired != 0 || m.Address != address || data != before || state[0] != previous[0] || state[1] != previous[1] || state[2] != previous[2] || clock != 6 || epoch != 6 {
			return fmt.Errorf("compiled WASM full-key/cross-page guard mismatch")
		}
	}
	highAddress := uint64(4096) + (uint64(1) << 48)
	m.Fill(highAddress>>12, &data, 3, &epoch)
	state[0], state[1] = highAddress, 0x123456789abcdef0
	if err = n.CallMemory(entry, state, m); err != nil {
		return err
	}
	if m.Status != 0 || m.Retired != 1 || state[2] != state[1] || clock != 7 || epoch != 7 {
		return fmt.Errorf("compiled WASM high page identity mismatch")
	}
	for _, mode := range [4]int{0, 1, 2, 3} {
		m.Fill(1, &data, 3, &epoch)
		m.Clock = &clock
		switch mode {
		case 0:
			m.PageSlot(1).Data = nil
		case 1:
			m.PageSlot(1).Permissions = 7
		case 2:
			m.Clock = nil
		case 3:
			m.PageSlot(1).Epoch = nil
		}
		state[0], state[1] = 4096, 99
		before = data
		if err = n.CallMemory(entry, state, m); err != nil {
			return err
		}
		if m.Status != 1 || m.Retired != 0 || data != before || clock != 7 || epoch != 7 {
			return fmt.Errorf("compiled WASM nullable/permission guard mismatch")
		}
	}
	if err = browserPageWorkingSetProbe(); err != nil {
		return err
	}
	if err = browserRefillProbe(); err != nil {
		return err
	}
	return browserLinkProbe()
}
