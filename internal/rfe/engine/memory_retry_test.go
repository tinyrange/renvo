//go:build !renvo

package engine

import (
	"encoding/binary"
	"fmt"
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

type retryRAM struct {
	data         [3][4096]byte
	context      emu.MemoryContext
	clock, epoch uint64
	identity     *CodeIdentity
	refills      int
	afterRefill  func()
	deny         bool
}

func (m *retryRAM) Read(a uint64, n int, x bool) (uint64, error) {
	if a+uint64(n) > 12288 || x && m.deny {
		return 0, fmt.Errorf("access denied")
	}
	v := uint64(0)
	for i := 0; i < n; i++ {
		at := a + uint64(i)
		v |= uint64(m.data[at>>12][at&4095]) << uint(8*i)
	}
	return v, nil
}
func (m *retryRAM) Write(a uint64, n int, v uint64) error {
	if a+uint64(n) > 12288 {
		return fmt.Errorf("access denied")
	}
	for i := 0; i < n; i++ {
		at := a + uint64(i)
		m.data[at>>12][at&4095] = byte(v >> uint(8*i))
	}
	return nil
}
func (m *retryRAM) CodeContext() CodeStamp { return CodeStamp{m.identity, m.epoch} }
func (m *retryRAM) CodeVersion(a uint64) (CodeStamp, error) {
	if m.deny || a >= 4096 {
		return CodeStamp{}, fmt.Errorf("execute denied")
	}
	return m.CodeContext(), nil
}
func (m *retryRAM) NativeContext() *emu.MemoryContext { return &m.context }
func (m *retryRAM) NativeRefill(a uint64) {
	m.refills++
	if p := a >> 12; p > 0 && p < 3 {
		m.context.Fill(p, &m.data[p], 3, &m.clock)
	}
	if m.afterRefill != nil {
		f := m.afterRefill
		m.afterRefill = nil
		f()
	}
}

type retryCPU struct {
	state   [2]uint64
	memory  *retryRAM
	retired uint64
	slow    int
}

func (c *retryCPU) Registers() []uint64 { return c.state[:] }
func (c *retryCPU) MemoryBus() Memory   { return c.memory }
func (c *retryCPU) Retirement() *uint64 { return &c.retired }
func retryFetch(m Memory, pc uint64) (Instruction, error) {
	v, err := m.Read(pc, 2, true)
	return Instruction{PC: pc, Bits: v, Length: 2, Terminal: true, Memory: v != 4, Flow: Successors{Known: true, Next: pc + 2}}, err
}
func (c *retryCPU) Step() error {
	ins, err := retryFetch(c.memory, c.state[1])
	if err != nil {
		return err
	}
	return c.ExecuteInstruction(ins)
}
func (c *retryCPU) ExecuteInstruction(ins Instruction) error {
	c.slow++
	a := uint64(4096)
	if ins.Bits == 2 {
		a = 8192
	}
	v, err := c.memory.Read(a, 8, false)
	if err != nil {
		return err
	}
	switch ins.Bits {
	case 3:
		if err = c.memory.Write(a, 8, v+1); err != nil {
			return err
		}
	case 4:
		c.state[0] = 99
	case 5:
		w, err := c.memory.Read(8192, 8, false)
		if err != nil {
			return err
		}
		c.state[0] = v + w
	default:
		c.state[0] = v
	}
	c.state[1] = ins.PC + 2
	c.retired++
	return nil
}
func newRetryEngine(t *testing.T, ops ...uint16) (*Engine, *retryCPU) {
	t.Helper()
	m := &retryRAM{clock: 1, epoch: 1, identity: new(CodeIdentity)}
	m.context.Clock = &m.clock
	m.data[1][0], m.data[2][0] = 42, 17
	for i, op := range ops {
		binary.LittleEndian.PutUint16(m.data[0][i*2:], op)
	}
	arch := Architecture{Name: "retry", StateWords: 2, PC: 1, Alignment: 2, Fetch: retryFetch, Lower: func(b *emu.Builder, ins Instruction, completed int) bool {
		b.Store(1, b.Constant(ins.PC))
		b.Checkpoint(2, completed)
		if ins.Bits == 4 {
			b.Store(0, b.Constant(99))
		} else {
			a := uint64(4096)
			if ins.Bits == 2 {
				a = 8192
			}
			v := b.MemoryLoad(b.Constant(a), 8)
			if ins.Bits == 3 {
				b.MemoryStore(b.Constant(a), b.Binary(emu.Add, v, b.Constant(1)), 8)
			} else {
				if ins.Bits == 5 {
					v = b.Binary(emu.Add, v, b.MemoryLoad(b.Constant(8192), 8))
				}
				b.Store(0, v)
			}
		}
		b.Store(1, b.Constant(ins.PC+2))
		b.Checkpoint(2, completed+1)
		return true
	}}
	cfg := DefaultEngineConfig()
	cfg.IRThreshold, cfg.NativeThreshold, cfg.MaxInstructions, cfg.MaxRegionInstructions = 1, 1, 1, 1
	e, err := New(cfg, arch)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { e.Close() })
	if !e.NativeAvailable() {
		t.Skip("native unavailable")
	}
	return e, &retryCPU{memory: m}
}
func TestMemoryRetryOrdinaryRefillAndTwoMissBound(t *testing.T) {
	for _, op := range []uint16{1, 5} {
		e, c := newRetryEngine(t, op)
		warmRetry(t, e, c)
		if err := e.RunQuanta(c, 1); err != nil {
			t.Fatal(err)
		}
		slow, want, refills := 0, uint64(42), 1
		if op == 5 {
			slow, want, refills = 1, 59, 2
		}
		if c.retired != 1 || c.state != [2]uint64{want, 2} || c.slow != slow || c.memory.refills != refills || e.memoryRetry || e.inNativeBatch {
			t.Fatal("retry progress/bound", c.state, c.retired, c.slow, c.memory.refills)
		}
	}
}
func TestMemoryRetryLinkedPrefixNotReplayedOrBudgetExtended(t *testing.T) {
	e, c := newRetryEngine(t, 3, 2, 4)
	for i := 0; i < 2; i++ {
		if err := e.Step(c, 1); err != nil {
			t.Fatal(err)
		}
	}
	// Publish the already compiled successor before entering the linked prefix.
	c.state[1] = 2
	if err := e.Run(c, 1); err != nil {
		t.Fatal(err)
	}
	c.state, c.retired, c.slow = [2]uint64{}, 0, 0
	c.memory.context.ClearPages()
	c.memory.context.Fill(1, &c.memory.data[1], 3, &c.memory.clock)
	c.memory.data[1][0] = 42
	if err := e.RunQuanta(c, 2); err != nil {
		t.Fatal(err)
	}
	if c.retired != 2 || c.state != [2]uint64{17, 4} || c.memory.data[1][0] != 43 || c.slow != 0 {
		t.Fatal("replayed prefix or exceeded budget", c.state, c.retired, c.slow, c.memory.data[1][0])
	}
}
func TestMemoryRetryRevalidatesChangedCodeAndExecutePermission(t *testing.T) {
	for _, denied := range []bool{false, true} {
		e, c := newRetryEngine(t, 1)
		warmRetry(t, e, c)
		c.memory.afterRefill = func() {
			c.memory.epoch++
			binary.LittleEndian.PutUint16(c.memory.data[0][:], 4)
			c.memory.deny = denied
		}
		err := e.RunQuanta(c, 1)
		if denied {
			if err == nil || c.retired != 0 || c.state != [2]uint64{} {
				t.Fatal("stale execute grant", err, c.state, c.retired)
			}
		} else if err != nil || c.retired != 1 || c.state != [2]uint64{99, 2} {
			t.Fatal("stale code after refill", err, c.state, c.retired)
		}
	}
}
func TestMemoryRetryDoesNotReenterUnlinkedNativeCallback(t *testing.T) {
	e, c := newRetryEngine(t, 1)
	c.memory.identity = nil
	if err := e.Step(c, 1); err != nil {
		t.Fatal(err)
	}
	c.state, c.retired, c.slow = [2]uint64{}, 0, 0
	c.memory.context.ClearPages()
	if err := e.RunQuanta(c, 1); err != nil {
		t.Fatal(err)
	}
	if c.retired != 1 || c.slow != 1 || c.state != [2]uint64{42, 2} || e.inNativeBatch || e.memoryRetry {
		t.Fatal("unlinked callback retry", c.state, c.retired, c.slow)
	}
}

// Install the code before exercising a refilled batched session. Plain Step
// retains its original single-block/architectural-miss behavior and counters.
func warmRetry(t *testing.T, e *Engine, c *retryCPU) {
	t.Helper()
	for p := uint64(1); p < 3; p++ {
		c.memory.context.Fill(p, &c.memory.data[p], 3, &c.memory.clock)
	}
	if err := e.Step(c, 1); err != nil {
		t.Fatal(err)
	}
	c.memory.context.ClearPages()
	c.state, c.retired, c.slow = [2]uint64{}, 0, 0
	c.memory.refills = 0
}
