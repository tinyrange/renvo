package engine

import (
	"encoding/binary"
	"testing"
)

func TestColdDiscoveryObservationCannotGrantCodePermission(t *testing.T) {
	memory := new(stubMemory)
	binary.LittleEndian.PutUint16(memory.bytes[:], 0x101)
	config := DefaultEngineConfig()
	config.Mode = "ir"
	config.DiscoveryThreshold = 4
	config.IRThreshold, config.NativeThreshold = 1, 1
	config.MaxInstructions = 1
	e, err := New(config, stubArchitecture())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	c := stubCPU{memory: memory}
	for i := 0; i < 3; i++ {
		c.state[1] = 0
		if err = e.Step(&c, 1); err != nil || e.CachedBlocks() != 0 {
			t.Fatal("cold observation decoded prematurely", err)
		}
	}
	c.state[1] = 0
	if err = e.Step(&c, 1); err != nil || e.CachedBlocks() != 1 || c.state[0] != 4 {
		t.Fatal("hot source was not admitted", err)
	}
	memory.denySecond = true
	c.state[1] = 4096
	for i := 0; i < 8; i++ {
		before := c.state
		retired := c.retired
		if err = e.Step(&c, 1); err == nil || c.state != before || c.retired != retired {
			t.Fatal("hot observation bypassed execute permission")
		}
	}
}
