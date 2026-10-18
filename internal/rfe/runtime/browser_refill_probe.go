//go:build renvo && browser

package runtime

import "fmt"

// Runs the real exported AOT service from a generated WASM memory guard while
// the original AOT caller and generated leaf both retain live frames/SSA values.
func browserRefillProbe() error {
	n, err := NewNative(1 << 20)
	if err != nil {
		return err
	}
	defer n.Close()
	m := &MemoryContext{}
	clock := uint64(9)
	m.Clock = &clock
	m.ClaimLinks(n)
	if !n.AdmitCodeVersionedContext(m) {
		return fmt.Errorf("refill owner admission failed")
	}
	if err = n.PrepareLinks(5, 0); err != nil {
		return err
	}
	var b Builder
	b.MemoryInstructionProgress = true
	b.Checkpoint(5, 0)
	a := b.Load(2)
	// A committed store prefix must never be replayed on a later memory miss.
	v := b.Binary(Add, b.MemoryLoad(a, 8), b.Constant(1))
	b.MemoryStore(a, v, 8)
	b.Store(1, b.Binary(Add, b.Load(1), b.Constant(1)))
	b.Store(0, b.Constant(4))
	b.Checkpoint(5, 1)
	loaded := b.MemoryLoad(b.Load(3), 8)
	b.Store(4, loaded)
	b.Store(0, b.Constant(8))
	b.Checkpoint(5, 2)
	entry, err := n.CompileMemory(b.FinishMemory(5), 5)
	if err != nil {
		return err
	}
	if err = n.PrepareTargetLink(0, entry, 5, 2); err != nil {
		return err
	}
	m.PublishLink(0, entry, 2, [17]uint8{0, 1, 2})
	first, second := new([4096]byte), new([4096]byte)
	first[0], second[0] = 40, 79
	m.Fill(1, first, 3, &clock)
	provider := &browserRefillProbeProvider{context: m, second: second, clock: &clock}
	if !n.BeginFastRefill(m, provider.refill) {
		return fmt.Errorf("refill begin failed")
	}
	if BrowserRefill() {
		return fmt.Errorf("refill granted outside native call")
	}
	state := []uint64{0, 0, 4096, 8192, 99}
	err = n.CallLinked(state, m, 3)
	n.EndFastRefill()
	if err != nil {
		return err
	}
	if provider.calls != 1 || state[0] != 8 || state[1] != 1 || state[4] != 79 || first[0] != 41 || m.Total != 2 || m.Remaining != 1 || m.MemoryTotal != 2 || m.Status != 0 || browserRefillOwner != nil || m.wasmServiceSP != 0 {
		return fmt.Errorf("refill prefix/budget/state mismatch calls=%d pc=%d count=%d value=%d data=%d total=%d remaining=%d status=%d", provider.calls, state[0], state[1], state[4], first[0], m.Total, m.Remaining, m.Status)
	}
	copied := *m
	if n.BeginFastRefill(&copied, provider.refill) {
		return fmt.Errorf("copied context inherited refill authority")
	}
	// An invalid provider claiming success without filling cannot spin. Every
	// retry rechecks the full tag and permissions; the original fault prefix stays.
	m.Forget(2)
	provider.calls, provider.fake = 0, true
	if !n.BeginFastRefill(m, provider.refill) {
		return fmt.Errorf("bounded refill begin failed")
	}
	state = []uint64{0, 0, 4096, 8192, 99}
	err = n.CallLinked(state, m, 3)
	n.EndFastRefill()
	if err != nil {
		return err
	}
	if provider.calls != 64 || state[0] != 4 || state[1] != 1 || state[4] != 99 || first[0] != 42 || m.Total != 1 || m.Remaining != 2 || m.Status != 1 || m.Address != 8192 {
		return fmt.Errorf("refill bound/precise failure mismatch %d %d %d %d", provider.calls, m.Total, m.Remaining, m.Status)
	}
	if err = browserRefillForgedStackProbe(n); err != nil {
		return err
	}
	for _, width := range []int{1, 2, 4, 8, 16} {
		for _, permissions := range []uint64{3, 1, 7} {
			if err = browserRefillWriteCase(width, permissions); err != nil {
				return err
			}
		}
	}
	return nil
}

type browserRefillProbeProvider struct {
	context *MemoryContext
	second  *[4096]byte
	clock   *uint64
	calls   int
	fake    bool
}

func (p *browserRefillProbeProvider) refill(got *MemoryContext) bool {
	p.calls++
	if p.fake {
		return true
	}
	if got != p.context || got.Address != 8192 || BrowserRefill() {
		return false
	}
	got.Fill(2, p.second, 3, p.clock)
	return true
}

func browserRefillForgedStackProbe(n *Native) error {
	var b Builder
	b.Checkpoint(5, 0)
	b.Store(4, b.MemoryLoad(b.Load(2), 8))
	b.Checkpoint(5, 1)
	entry, err := n.CompileMemory(b.FinishMemory(5), 5)
	if err != nil {
		return err
	}
	forged := &MemoryContext{}
	forged.wasmServiceSP, forged.wasmServiceFP = 1, 1
	state := []uint64{0, 0, 8192, 0, 99}
	if err = n.CallMemory(entry, state, forged); err != nil {
		return err
	}
	if forged.Status != 1 || forged.Retired != 0 || state[4] != 99 || forged.wasmServiceSP != 0 || forged.wasmServiceFP != 0 {
		return fmt.Errorf("public context selected callback stack")
	}
	return nil
}

type browserRefillWriteProvider struct {
	page        *[4096]byte
	clock       *uint64
	permissions uint64
	calls       int
}

func (p *browserRefillWriteProvider) refill(m *MemoryContext) bool {
	p.calls++
	if !m.FastRefillWrite() || m.Address != 8192 {
		return false
	}
	m.Fill(2, p.page, p.permissions, p.clock)
	return true
}
func browserRefillWriteCase(width int, permissions uint64) error {
	n, err := NewNative(1 << 20)
	if err != nil {
		return err
	}
	defer n.Close()
	m := &MemoryContext{}
	clock := uint64(9)
	m.Clock = &clock
	m.ClaimLinks(n)
	if !n.AdmitCodeVersionedContext(m) {
		return fmt.Errorf("write refill owner failed")
	}
	if err = n.PrepareLinks(5, 0); err != nil {
		return err
	}
	var b Builder
	b.MemoryInstructionProgress = true
	b.Checkpoint(5, 0)
	b.Store(1, b.Binary(Add, b.Load(1), b.Constant(1)))
	b.Store(0, b.Constant(4))
	b.Checkpoint(5, 1)
	if width == 16 {
		b.MemoryPairStore(b.Load(2), b.Load(3), b.Constant(44))
	} else {
		b.MemoryStore(b.Load(2), b.Load(3), width)
	}
	b.Store(0, b.Constant(8))
	b.Checkpoint(5, 2)
	entry, err := n.CompileMemory(b.FinishMemory(5), 5)
	if err != nil {
		return err
	}
	if err = n.PrepareTargetLink(0, entry, 5, 2); err != nil {
		return err
	}
	m.PublishLink(0, entry, 2, [17]uint8{0, 0, 1})
	page := new([4096]byte)
	page[0] = 3
	provider := &browserRefillWriteProvider{page: page, clock: &clock, permissions: permissions}
	if !n.BeginFastRefill(m, provider.refill) {
		return fmt.Errorf("write refill begin failed")
	}
	state := []uint64{0, 0, 8192, 17, 99}
	err = n.CallLinked(state, m, 3)
	n.EndFastRefill()
	if err != nil {
		return err
	}
	if permissions == 3 {
		if state[0] != 8 || state[1] != 1 || page[0] != 17 || m.Total != 2 || m.Remaining != 1 || m.Status != 0 || provider.calls != 1 || width == 16 && page[8] != 44 {
			return fmt.Errorf("write refill success mismatch width=%d total=%d calls=%d", width, m.Total, provider.calls)
		}
	} else {
		if state[0] != 4 || state[1] != 1 || page[0] != 3 || page[8] != 0 || m.Total != 1 || m.Remaining != 2 || m.Status != 1 || provider.calls != 64 {
			return fmt.Errorf("write refill bypassed barrier/permission width=%d permission=%d calls=%d", width, permissions, provider.calls)
		}
	}
	return nil
}
