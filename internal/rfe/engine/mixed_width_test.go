package engine

import (
	"encoding/binary"
	"fmt"
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

// A deliberately non-A64 guest: two-byte adds, four-byte adds and two-byte
// direct branches. The extended instruction can straddle executable pages.
type stubMemory struct {
	bytes      [8192]byte
	denySecond bool
}

func (m *stubMemory) Read(at uint64, size int, execute bool) (uint64, error) {
	if at >= 8192 || uint64(size) > 8192-at || execute && m.denySecond && at+uint64(size) > 4096 {
		return 0, fmt.Errorf("fetch fault")
	}
	var v uint64
	for i := 0; i < size; i++ {
		v |= uint64(m.bytes[at+uint64(i)]) << uint(i*8)
	}
	return v, nil
}
func (m *stubMemory) Write(at uint64, size int, v uint64) error {
	if at >= 8192 || uint64(size) > 8192-at {
		return fmt.Errorf("write fault")
	}
	for i := 0; i < size; i++ {
		m.bytes[at+uint64(i)] = byte(v >> uint(i*8))
	}
	return nil
}

type stubCPU struct {
	state   [2]uint64
	memory  Memory
	retired uint64
}

func (c *stubCPU) Registers() []uint64 { return c.state[:] }
func (c *stubCPU) MemoryBus() Memory   { return c.memory }
func (c *stubCPU) Retirement() *uint64 { return &c.retired }
func (c *stubCPU) Step() error {
	if c.state[1]&1 != 0 {
		return fmt.Errorf("unaligned PC")
	}
	ins, err := stubFetch(c.memory, c.state[1])
	if err != nil {
		return err
	}
	return c.ExecuteInstruction(ins)
}
func (c *stubCPU) ExecuteInstruction(ins Instruction) error {
	if ins.Bits&255 == 3 {
		c.state[1] = ins.Bits >> 8
	} else {
		value := ins.Bits >> 8
		if ins.Length == 4 {
			value = ins.Bits >> 16
		}
		c.state[0] += value
		c.state[1] += uint64(ins.Length)
	}
	c.retired++
	return nil
}
func stubFetch(m Memory, pc uint64) (Instruction, error) {
	bits, err := m.Read(pc, 2, true)
	if err != nil {
		return Instruction{}, err
	}
	n := uint8(2)
	if bits&255 == 2 {
		n = 4
		bits, err = m.Read(pc, 4, true)
		if err != nil {
			return Instruction{}, err
		}
	}
	ins := Instruction{PC: pc, Bits: bits, Length: n, Terminal: bits&255 == 3, Flow: Successors{Known: true, Next: pc + uint64(n)}}
	if ins.Terminal {
		ins.Flow.Next = bits >> 8
	}
	return ins, nil
}
func stubArchitecture() Architecture {
	return Architecture{Name: "mixed", StateWords: 2, PC: 1, Alignment: 2, Fetch: stubFetch, Lower: func(b *emu.Builder, ins Instruction, completed int) bool {
		if !ins.Terminal {
			value := ins.Bits >> 8
			if ins.Length == 4 {
				value = ins.Bits >> 16
			}
			b.Store(0, b.Binary(emu.Add, b.Load(0), b.Constant(value)))
		}
		b.Store(1, b.Constant(ins.Flow.Next))
		return true
	}}
}
func TestMixedWidthExecutionAndStraddlingInvalidation(t *testing.T) {
	for _, mode := range []string{"interpreter", "ir", "native"} {
		m := new(stubMemory)
		binary.LittleEndian.PutUint16(m.bytes[4092:], 0x101)
		binary.LittleEndian.PutUint32(m.bytes[4094:], 0x00020002)
		binary.LittleEndian.PutUint16(m.bytes[4098:], 0x301)
		cfg := DefaultEngineConfig()
		cfg.Mode = mode
		cfg.IRThreshold = 1
		cfg.NativeThreshold = 1
		cfg.MaxInstructions = 2
		e, err := New(cfg, stubArchitecture())
		if err != nil {
			t.Fatal(err)
		}
		c := stubCPU{memory: m}
		c.state[1] = 4092
		for c.retired < 2 {
			if err = e.Step(&c, 2-c.retired); err != nil {
				t.Fatal(err)
			}
		}
		if c.state != [2]uint64{3, 4098} || c.retired != 2 {
			t.Fatal(mode, c.state, c.retired)
		}
		// Invalidate the second half of the final instruction, not its start.
		binary.LittleEndian.PutUint16(m.bytes[4096:], 7)
		c.state = [2]uint64{0, 4092}
		c.retired = 0
		for c.retired < 2 {
			if err = e.Step(&c, 2-c.retired); err != nil {
				t.Fatal(err)
			}
		}
		if c.state != [2]uint64{8, 4098} {
			t.Fatal("stale instruction extent", mode, c.state)
		}
		c.state = [2]uint64{0, 4092}
		c.retired = 0
		m.denySecond = true
		if err = e.Step(&c, 1); err != nil {
			t.Fatal(err)
		}
		if err = e.Step(&c, 1); err == nil || c.retired != 1 || c.state != [2]uint64{1, 4094} {
			t.Fatal("fault prefix", mode, c.state, c.retired, err)
		}
		e.Close()
	}
}

// Both page-level and whole-code generations belong to memory, not the ISA.
type versionedStub struct {
	stubMemory
	identity CodeIdentity
	epoch    uint64
	pages    [2]uint64
	context  emu.MemoryContext
}

func (m *versionedStub) CodeContext() CodeStamp {
	return CodeStamp{Identity: &m.identity, Epoch: m.epoch}
}
func (m *versionedStub) CodeVersion(at uint64) (CodeStamp, error) {
	if at >= 8192 || m.denySecond && at >= 4096 {
		return CodeStamp{}, fmt.Errorf("execute permission")
	}
	return CodeStamp{Identity: &m.identity, Epoch: m.pages[at/4096]}, nil
}
func (m *versionedStub) Write(at uint64, n int, v uint64) error {
	if err := m.stubMemory.Write(at, n, v); err != nil {
		return err
	}
	m.epoch++
	m.pages[at/4096] = m.epoch
	m.pages[(at+uint64(n)-1)/4096] = m.epoch
	return nil
}
func (m *versionedStub) NativeContext() *emu.MemoryContext { return &m.context }
func (m *versionedStub) NativeRefill(uint64)               {}

func TestMixedWidthVersionedRegionsAndBudgets(t *testing.T) {
	for _, maximum := range []int{64, 65536} {
		m := new(versionedStub)
		m.Write(2, 2, 0x101)
		m.Write(4, 4, 0x00020002)
		m.Write(8, 2, 0x203)
		cfg := DefaultEngineConfig()
		cfg.IRThreshold = 1
		cfg.NativeThreshold = 1
		cfg.MaxInstructions = 2
		cfg.MaxNativeInstructions = maximum
		e, err := New(cfg, stubArchitecture())
		if err != nil {
			t.Fatal(err)
		}
		if !e.NativeAvailable() {
			e.Close()
			t.Skip("native host unavailable")
		}
		c := stubCPU{memory: m}
		c.state[1] = 2
		if err = e.Step(&c, 2); err != nil {
			t.Fatal(err)
		}
		if err = e.Step(&c, 1); err != nil {
			t.Fatal(err)
		}
		for _, budget := range []uint64{1, 2, 3, 63, 64, 65, 1024, 65537} {
			want := c
			for i := uint64(0); i < budget; i++ {
				if err = want.Step(); err != nil {
					t.Fatal(err)
				}
			}
			before := c.retired
			for c.retired-before < budget {
				if err = e.RunQuanta(&c, budget-(c.retired-before)); err != nil {
					t.Fatal(err)
				}
				if c.retired-before > budget {
					t.Fatal("instruction budget counted bytes")
				}
			}
			if c.state != want.state {
				t.Fatal("mixed native state", maximum, budget, c.state, want.state)
			}
		}
		if e.Stats.Linked == 0 || e.Stats.Regions == 0 {
			t.Fatal("did not exercise mixed-width native regions", e.Stats)
		}
		m.Write(6, 2, 9) // mutate only the latter half of the 32-bit instruction
		want := c
		want.Step()
		want.Step()
		want.Step()
		before := c.retired
		for c.retired-before < 3 {
			if err = e.RunQuanta(&c, 3-(c.retired-before)); err != nil {
				t.Fatal(err)
			}
		}
		if c.state != want.state {
			t.Fatal("stale native region survived code mutation")
		}
		e.Close()
	}
}
func TestVersionedInstructionIncludesFinalPage(t *testing.T) {
	m := new(versionedStub)
	m.Write(4094, 4, 0x00020002)
	cfg := DefaultEngineConfig()
	cfg.Mode = "ir"
	cfg.IRThreshold = 1
	cfg.MaxInstructions = 1
	e, err := New(cfg, stubArchitecture())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	c := stubCPU{memory: m}
	c.state[1] = 4094
	if err = e.Step(&c, 1); err != nil {
		t.Fatal(err)
	}
	block := e.blocks[4094]
	if len(block.dependencies) != 2 {
		t.Fatal("last instruction's second page omitted")
	}
	m.Write(4096, 2, 7)
	c.state = [2]uint64{0, 4094}
	if err = e.Step(&c, 1); err != nil || c.state[0] != 7 {
		t.Fatal("stale second page", err, c.state)
	}
	c.state = [2]uint64{0, 4094}
	m.denySecond = true
	m.epoch++
	before := c.retired
	if err = e.Step(&c, 1); err == nil || c.retired != before || c.state[1] != 4094 {
		t.Fatal("cross-page execute permission bypass")
	}
}
