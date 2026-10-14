package engine

import (
	"fmt"
	"testing"
)

type discoveryCollisionMemory struct {
	words map[uint64]uint64
	deny  uint64
}

func (m *discoveryCollisionMemory) Read(pc uint64, n int, execute bool) (uint64, error) {
	v, ok := m.words[pc]
	if !ok || n != 2 || execute && pc == m.deny {
		return 0, fmt.Errorf("execute denied")
	}
	return v, nil
}
func (*discoveryCollisionMemory) Write(uint64, int, uint64) error { return fmt.Errorf("read-only") }

func TestCollidingDiscoveryWorkingSetWarmsWithoutGrantingPermission(t *testing.T) {
	pcs := []uint64{0, 8196, 16392, 24588} // collide even in the former direct 4096 table
	m := &discoveryCollisionMemory{words: map[uint64]uint64{}, deny: ^uint64(0)}
	for _, pc := range pcs {
		m.words[pc] = 0x101
	}
	cfg := DefaultEngineConfig()
	cfg.Mode = "ir"
	cfg.MaxBlocks, cfg.MaxInstructions = 4, 1
	cfg.DiscoveryThreshold = 4
	cfg.IRThreshold, cfg.NativeThreshold = 1, 1
	e, err := New(cfg, stubArchitecture())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	c := &stubCPU{memory: m}
	for round := 0; round < 4; round++ {
		for _, pc := range pcs {
			c.state[1] = pc
			if err := e.Step(c, 1); err != nil {
				t.Fatal(err)
			}
		}
		if round < 3 && e.CachedBlocks() != 0 {
			t.Fatal("collision bypassed discovery threshold")
		}
	}
	if e.CachedBlocks() != 4 || c.state[0] != 16 || c.retired != 16 {
		t.Fatal("colliding hot PCs stayed cold", e.Stats, c.state)
	}
	m.deny = pcs[1]
	c.state[1] = pcs[1]
	before, retired := c.state, c.retired
	if err := e.Step(c, 1); err == nil || c.state != before || c.retired != retired {
		t.Fatal("retained heat granted execute permission")
	}
}
