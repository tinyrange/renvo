package engine

import (
	"encoding/binary"
	"fmt"
	"testing"
)

type identicalMappingMemory struct {
	stubMemory
	pages       [2]*CodeIdentity
	context     CodeIdentity
	epoch       uint64
	onRead      func()
	denyVersion bool
}

func (m *identicalMappingMemory) CodeContext() CodeStamp { return CodeStamp{&m.context, m.epoch} }
func (m *identicalMappingMemory) CodeVersion(address uint64) (CodeStamp, error) {
	if m.denyVersion || address >= 8192 {
		return CodeStamp{}, fmt.Errorf("execute mapping denied")
	}
	return CodeStamp{m.pages[address>>12], 1}, nil
}
func (m *identicalMappingMemory) Read(address uint64, n int, x bool) (uint64, error) {
	if m.onRead != nil {
		f := m.onRead
		m.onRead = nil
		f()
	}
	return m.stubMemory.Read(address, n, x)
}
func TestIdenticalPhysicalRemapReusesCompiledLeafButNotOtherRegionParts(t *testing.T) {
	m := &identicalMappingMemory{pages: [2]*CodeIdentity{new(CodeIdentity), new(CodeIdentity)}, epoch: 1}
	binary.LittleEndian.PutUint16(m.bytes[4092:], 0x101)
	binary.LittleEndian.PutUint32(m.bytes[4094:], 0x00020002)
	cfg := DefaultEngineConfig()
	cfg.IRThreshold, cfg.NativeThreshold, cfg.MaxInstructions = 1, 1, 2
	arch := stubArchitecture()
	arch.EqualDecoded = func(a, b any) bool { return a == b }
	e, err := New(cfg, arch)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	c := stubCPU{memory: m, state: [2]uint64{0, 4092}}
	if err = e.Step(&c, 2); err != nil {
		t.Fatal(err)
	}
	original := e.blocks[4092]
	if original == nil || !original.native {
		t.Fatal("leaf did not compile")
	}
	entry, promotions := original.entry, e.Stats.Promotions
	// A region would additionally depend on instructions not present in the leaf.
	original.region, original.regionInstructions, original.regionEntry = true, 3, entry+16
	original.regionAttempted = true
	m.pages = [2]*CodeIdentity{new(CodeIdentity), new(CodeIdentity)}
	m.epoch++
	c.state, c.retired = [2]uint64{0, 4092}, 0
	if err = e.Step(&c, 2); err != nil {
		t.Fatal(err)
	}
	got := e.blocks[4092]
	if got != original || got.entry != entry || got.region || e.Stats.Promotions != promotions || c.state != [2]uint64{3, 4098} || c.retired != 2 {
		t.Fatal("identical remap did not preserve only compiled leaf", c.state, c.retired, e.Stats)
	}
	// Mutation in the second half of a straddling instruction is not equivalent.
	binary.LittleEndian.PutUint16(m.bytes[4096:], 9)
	m.pages[1] = new(CodeIdentity)
	m.epoch++
	c.state, c.retired = [2]uint64{0, 4092}, 0
	if e.block(&c) != nil || e.Stats.Invalidations == 0 {
		t.Fatal("changed straddling instruction reused old code")
	}
}
func TestIdenticalRemapRevalidationRejectsExecuteFaultAndResolverChanges(t *testing.T) {
	for _, kind := range []string{"execute-fault", "version-fault", "context-transition", "page-transition"} {
		t.Run(kind, func(t *testing.T) {
			m := &identicalMappingMemory{pages: [2]*CodeIdentity{new(CodeIdentity), new(CodeIdentity)}, epoch: 1}
			binary.LittleEndian.PutUint32(m.bytes[4094:], 0x00020002)
			cfg := DefaultEngineConfig()
			cfg.Mode = "ir"
			cfg.IRThreshold = 1
			cfg.MaxInstructions = 1
			arch := stubArchitecture()
			arch.EqualDecoded = func(a, b any) bool { return a == b }
			e, err := New(cfg, arch)
			if err != nil {
				t.Fatal(err)
			}
			defer e.Close()
			c := stubCPU{memory: m, state: [2]uint64{0, 4094}}
			block := e.block(&c)
			if block == nil {
				t.Fatal("missing initial code")
			}
			prior := block.context
			m.pages[0], m.pages[1] = new(CodeIdentity), new(CodeIdentity)
			m.epoch++
			switch kind {
			case "execute-fault":
				m.denySecond = true
			case "version-fault":
				m.denyVersion = true
			case "context-transition":
				m.onRead = func() { m.epoch++ }
			case "page-transition":
				m.onRead = func() { m.pages[1] = new(CodeIdentity) }
			}
			if e.revalidateIdenticalBlock(&c, 4094, block) || block.context != prior || c.retired != 0 || c.state != [2]uint64{0, 4094} {
				t.Fatal("unstable/denied revalidation admitted code", kind)
			}
		})
	}
}
