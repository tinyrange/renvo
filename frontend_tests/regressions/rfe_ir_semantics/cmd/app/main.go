package main

import (
	"renvo.dev/internal/rfe/engine"
	"renvo.dev/internal/rfe/linuxuser"
	emu "renvo.dev/internal/rfe/runtime"
)

func main() {
	portableEngine()
	var div emu.Builder
	a, rhs := div.Load(0), div.Load(1)
	for k := emu.UnsignedDivide; k <= emu.SignedRemainder; k++ {
		div.Store(k-emu.UnsignedDivide+2, div.Divide(k, a, rhs, 64))
	}
	divops := div.Finish(6)
	zero := []uint64{7, 0, 0, 0, 0, 0}
	overflow := []uint64{1 << 63, ^uint64(0), 0, 0, 0, 0}
	if emu.Interpret(divops, zero) != nil || zero[2] != ^uint64(0) || zero[3] != ^uint64(0) || zero[4] != 7 || zero[5] != 7 {
		panic("total division zero")
	}
	if emu.Interpret(divops, overflow) != nil || overflow[2] != 0 || overflow[3] != 1<<63 || overflow[4] != 1<<63 || overflow[5] != 0 {
		panic("total division overflow")
	}

	// Renvo-built runners retain the complete native API for portable fallback.
	// A missing loop method used to make importing the AArch64 engine fail.
	if native, err := emu.NewNative(4096); err != nil {
		var unavailable emu.Native
		if _, err := unavailable.CompileLoop(nil, 1, 1); err == nil {
			panic("unavailable native loop accepted")
		}
	} else {
		native.Close()
	}

	page := new([4096]byte)
	epoch := uint64(7)
	memory := emu.MemoryContext{NativeContext: emu.NativeContext{Clock: &epoch, Remaining: 9}}
	memory.Fill(65, page, 3, &epoch)
	if memory.Pages[1].Data != page || memory.Pages[1].Epoch != memory.Clock || memory.Remaining != 9 {
		panic("shared native ABI prefix")
	}
	memory.NativeContext.Total = 4
	memory.Forget(65)
	if memory.Total != 4 || memory.Pages[1].Data != nil {
		panic("shared native ABI mutation")
	}

	var b emu.Builder
	x := b.Load(0)
	r := b.Binary(emu.And, b.Binary(emu.Add, x, b.Constant(1)), b.Constant(65535))
	b.Store(0, r)
	b.Store(1, b.Choose(b.Binary(emu.Equal, r, b.Constant(0)), b.Constant(4), b.Constant(0)))
	b.Store(2, b.Shift(emu.Shr, x, 64))
	state := []uint64{65535, 0, 1}
	if emu.Interpret(b.Finish(3), state) != nil || state[0] != 0 || state[1] != 4 || state[2] != 0 {
		panic("RFE IR semantics")
	}
	var q emu.Queue
	q.Schedule(0, 2, 1, 2)
	q.Schedule(0, 1, 1, 1)
	e, ok := q.Pop(2)
	if !ok || e.Value != 1 {
		panic("RFE event order")
	}
	println("PASS")
}

// A second guest shape verifies the shared controller also compiles through
// Renvo's portable frontend, independently of the AArch64 archive/host runner.
type tinyCPU struct {
	state   [2]uint64
	memory  *linuxuser.Memory
	retired uint64
}

func (c *tinyCPU) Registers() []uint64      { return c.state[:] }
func (c *tinyCPU) MemoryBus() engine.Memory { return c.memory }
func (c *tinyCPU) Retirement() *uint64      { return &c.retired }
func (c *tinyCPU) Step() error {
	bits, err := c.memory.Read(c.state[1], 2, true)
	if err != nil {
		return err
	}
	return c.ExecuteInstruction(engine.Instruction{PC: c.state[1], Bits: bits, Length: 2})
}
func (c *tinyCPU) ExecuteInstruction(ins engine.Instruction) error {
	c.state[0] += ins.Bits
	c.state[1] += 2
	c.retired++
	return nil
}
func portableEngine() {
	m := linuxuser.NewMemory()
	if m.Map(4096, 4096, 7) != nil || m.Write(4096, 2, 1) != nil || m.Write(4098, 2, 2) != nil {
		panic("shared memory")
	}
	arch := engine.Architecture{Name: "tiny", StateWords: 2, PC: 1, Alignment: 2,
		Fetch: func(m engine.Memory, pc uint64) (engine.Instruction, error) {
			bits, err := m.Read(pc, 2, true)
			return engine.Instruction{PC: pc, Bits: bits, Length: 2}, err
		},
		Lower: func(b *emu.Builder, ins engine.Instruction, completed int) bool {
			b.Store(0, b.Binary(emu.Add, b.Load(0), b.Constant(ins.Bits)))
			b.Store(1, b.Constant(ins.PC+2))
			return true
		},
	}
	config := engine.DefaultEngineConfig()
	config.Mode = "ir"
	config.IRThreshold = 1
	config.MaxInstructions = 2
	e, err := engine.New(config, arch)
	if err != nil {
		panic(err)
	}
	c := tinyCPU{memory: m}
	c.state[1] = 4096
	if e.Step(&c, 2) != nil || c.state[0] != 3 || c.state[1] != 4100 || c.retired != 2 {
		panic("shared variable-width engine")
	}
	e.Close()
	p := linuxuser.Process{Memory: m}
	regs := []uint64{99, 0, 0, 0, 0, 0, 0, 172}
	abi := linuxuser.SyscallABI{Number: 7, Result: 0, Arguments: [6]int{6, 5, 4, 3, 2, 1}, Trap: func(err error) (bool, error) { return false, err }}
	exited, _, err := p.ApplySyscall(regs, abi, nil, nil)
	if err != nil || exited || regs[0] != 1 || regs[7] != 172 {
		panic("shared syscall adapter")
	}
}
