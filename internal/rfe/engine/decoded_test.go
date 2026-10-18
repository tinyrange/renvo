package engine

import (
	"encoding/binary"
	"testing"
)

func TestOwnedDecodedValidationEvictionAndClose(t *testing.T) {
	type cell struct{ bits uint64 }
	var free []*cell
	live := map[*cell]bool{}
	allocated := 0
	memory := new(stubMemory)
	for i := 0; i < 16; i++ {
		binary.LittleEndian.PutUint16(memory.bytes[i*2:], 0x101)
	}
	arch := stubArchitecture()
	fetch := arch.Fetch
	arch.Fetch = func(m Memory, pc uint64) (Instruction, error) {
		ins, err := fetch(m, pc)
		if err != nil {
			return ins, err
		}
		var p *cell
		if len(free) > 0 {
			p = free[len(free)-1]
			free = free[:len(free)-1]
		} else {
			p = &cell{}
			allocated++
		}
		if live[p] {
			t.Fatal("reused live decoder descriptor")
		}
		live[p] = true
		p.bits = ins.Bits
		ins.Decoded = p
		return ins, nil
	}
	arch.Release = func(ins Instruction) {
		p := ins.Decoded.(*cell)
		if !live[p] {
			t.Fatal("double decoder release")
		}
		delete(live, p)
		p.bits = 0
		free = append(free, p)
	}
	arch.EqualDecoded = func(a, b any) bool { return a.(*cell).bits == b.(*cell).bits }
	cfg := DefaultEngineConfig()
	cfg.MaxBlocks, cfg.MaxInstructions = 2, 1
	cfg.IRThreshold, cfg.NativeThreshold = 1, 1
	cfg.EvictCold, cfg.RecycleNative = true, true
	e, err := New(cfg, arch)
	if err != nil {
		t.Fatal(err)
	}
	c := stubCPU{memory: memory}
	for round := 0; round < 20; round++ {
		for pc := uint64(0); pc < 32; pc += 2 {
			c.state[1] = pc
			if err = e.Step(&c, 1); err != nil {
				t.Fatal(err)
			}
			// A second entry validates bytes using a distinct temporary descriptor.
			c.state[1] = pc
			if err = e.Step(&c, 1); err != nil {
				t.Fatal(err)
			}
			if len(live) > 2 || e.CachedBlocks() > 2 {
				t.Fatal("unbounded decoded ownership", len(live))
			}
		}
	}
	before := c.state[0]
	binary.LittleEndian.PutUint16(memory.bytes[30:], 0x201)
	c.state[1] = 30
	if err = e.Step(&c, 1); err != nil {
		t.Fatal(err)
	}
	if e.Stats.Invalidations == 0 {
		t.Fatal("changed code accepted through pointer equality")
	}
	// The invalidated lookup falls back once; rediscovery must see the new value.
	c.state[1] = 30
	if err = e.Step(&c, 1); err != nil {
		t.Fatal(err)
	}
	if c.state[0] <= before {
		t.Fatal("no execution after descriptor reuse")
	}
	if err = e.Close(); err != nil {
		t.Fatal(err)
	}
	if err = e.Close(); err != nil {
		t.Fatal(err)
	}
	if len(live) != 0 || allocated > 4 {
		t.Fatal("decoder lifetime leak", len(live), allocated)
	}
}
