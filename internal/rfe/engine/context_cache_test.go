package engine

import (
	"encoding/binary"
	"fmt"
	"testing"
)

type variantMemory struct {
	stubMemory
	identities   [3]CodeIdentity
	epochs       [3]uint64
	active       int
	denied       bool
	executeReads uint64
}

func (m *variantMemory) Read(at uint64, size int, execute bool) (uint64, error) {
	if execute {
		m.executeReads++
	}
	if execute && m.denied {
		return 0, fmt.Errorf("execute permission revoked")
	}
	return m.stubMemory.Read(at, size, execute)
}
func (m *variantMemory) CodeContext() CodeStamp {
	return CodeStamp{Identity: &m.identities[m.active], Epoch: m.epochs[m.active]}
}
func (m *variantMemory) CodeVersion(uint64) (CodeStamp, error) {
	if m.denied {
		return CodeStamp{}, fmt.Errorf("execute permission revoked")
	}
	return m.CodeContext(), nil
}
func (m *variantMemory) selectVariant(variant int, add uint16) {
	m.active = variant
	binary.LittleEndian.PutUint16(m.bytes[:], add<<8|1)
}

func TestContextVariantsReuseNativeCodeAndKeepGlobalBound(t *testing.T) {
	memory := new(variantMemory)
	config := DefaultEngineConfig()
	config.CacheContexts, config.EvictCold = true, true
	config.MaxBlocks, config.MaxInstructions = 2, 1
	config.IRThreshold, config.NativeThreshold = 1, 1
	e, err := New(config, stubArchitecture())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	if !e.NativeAvailable() {
		t.Skip("native host unavailable")
	}
	c := stubCPU{memory: memory}
	run := func(variant int, add uint16) {
		t.Helper()
		memory.selectVariant(variant, add)
		c.state[1] = 0
		before := c.state[0]
		if err = e.Step(&c, 1); err != nil {
			t.Fatal(err)
		}
		if c.state[0] != before+uint64(add) || c.state[1] != 2 {
			t.Fatal("cross-context executable alias", c.state)
		}
		if e.CachedBlocks() > config.MaxBlocks || e.Stats.NativeBytes > config.NativeBytes {
			t.Fatal("expanded context cache budgets")
		}
	}
	for i := 0; i < 40; i++ {
		run(i%2, uint16(i%2+1))
	}
	if e.Stats.Promotions != 2 || e.Stats.Invalidations != 0 || e.Stats.Native != 40 || e.CachedBlocks() != 2 {
		t.Fatalf("valid contexts discarded hot code: %+v cached=%d", e.Stats, e.CachedBlocks())
	}
	// Mutating inactive address-space bytes must revoke only that variant.
	memory.epochs[0]++
	run(0, 7) // architectural fallback on an actual code-proof failure
	run(0, 7) // publish replacement, then reuse the untouched second variant
	run(1, 2)
	if e.Stats.Promotions != 3 || e.Stats.Invalidations != 1 {
		t.Fatal("incorrect variant invalidation", e.Stats)
	}
	// Execute revocation also forbids a warm native entry, without retirement.
	memory.denied = true
	memory.epochs[memory.active]++
	c.state[1] = 0
	retired := c.retired
	before := c.state
	if err = e.Step(&c, 1); err == nil || c.retired != retired || c.state != before {
		t.Fatal("revoked variant executed", err)
	}
	memory.denied = false
	memory.epochs[0]++ // restore the first context's original opcode below
	// A third context must compete within, not multiply, the global budget.
	for i := 0; i < 24; i++ {
		run(i%3, uint16(i%3+1))
	}
	if e.Stats.Evictions == 0 || e.CachedBlocks() > 2 {
		t.Fatal("context variants escaped global replacement")
	}
}

func TestContextVariantRecyclingRevokesEveryLookupOwner(t *testing.T) {
	memory := new(variantMemory)
	for i := 0; i < 200; i++ {
		binary.LittleEndian.PutUint16(memory.bytes[i*2:], uint16(i%7+1)<<8|1)
	}
	config := DefaultEngineConfig()
	config.CacheContexts, config.EvictCold, config.RecycleNative = true, true, true
	config.IRThreshold, config.NativeThreshold = 1, 1
	config.MaxBlocks, config.MaxInstructions, config.NativeBytes = 256, 1, 4096
	e, err := New(config, stubArchitecture())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	if !e.NativeAvailable() {
		t.Skip("native host unavailable")
	}
	c := stubCPU{memory: memory}
	expected := uint64(0)
	resets := uint64(0)
	for i := 0; i < 200; i++ {
		memory.active = i % 3
		c.state[1] = uint64(i * 2)
		if err = e.Step(&c, 1); err != nil {
			t.Fatal(err)
		}
		expected += uint64(i%7 + 1)
		if c.state[0] != expected || c.retired != uint64(i+1) || e.Stats.NativeBytes > config.NativeBytes {
			t.Fatal("recycling changed semantics or arena cap")
		}
		if e.Stats.ArenaResets != resets {
			if e.CachedBlocks() != 1 {
				t.Fatal("recycled arena retained a context variant", e.CachedBlocks())
			}
			resets = e.Stats.ArenaResets
		}
	}
	memory.active = 0
	c.state[1] = 0
	if err = e.Step(&c, 1); err != nil || c.state[0] != expected+1 || c.state[1] != 2 {
		t.Fatal("stale entry survived arena replacement", err, c.state)
	}
	if resets == 0 || e.Stats.Native != 201 {
		t.Fatal("recycling disabled native tier", e.Stats)
	}
}

func TestColdValidatedInstructionAvoidsDuplicateFetch(t *testing.T) {
	memory := new(variantMemory)
	memory.selectVariant(0, 7)
	config := DefaultEngineConfig()
	config.Mode = "ir"
	config.MaxInstructions = 1
	e, err := New(config, stubArchitecture())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	c := stubCPU{memory: memory}
	if err = e.Step(&c, 1); err != nil || c.state != [2]uint64{7, 2} || c.retired != 1 {
		t.Fatal("cold execution", err, c.state)
	}
	if memory.executeReads != 1 {
		t.Fatal("validated cold instruction was decoded twice", memory.executeReads)
	}
	memory.denied = true
	memory.epochs[0]++
	c.state[1] = 0
	if err = e.Step(&c, 1); err == nil || c.retired != 1 || c.state != [2]uint64{7, 0} {
		t.Fatal("cold cache bypassed revocation", err, c.state)
	}
}
