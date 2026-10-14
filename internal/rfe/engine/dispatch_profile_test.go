package engine

import (
	"encoding/json"
	"testing"
)

type profileOpcode string

func (p profileOpcode) ProfileOpcode() string { return string(p) }

func TestDispatchProfilePreservesInteriorMissesAcrossReset(t *testing.T) {
	if profileMonotonicNS() == 0 {
		t.Skip("host monotonic profiling unavailable")
	}
	e := &Engine{config: EngineConfig{ProfileNative: true}, blocks: map[uint64]*compiledBlock{}}
	c := &stubCPU{memory: new(stubMemory)}
	// A real diagnostic record carries an unsupported decoder classification.
	e.blocks[0x1234] = &compiledBlock{interpretOnly: true, instructions: []Instruction{{Bits: 0xd50342df, Decoded: profileOpcode("control-write")}}}
	for i := 0; i < 7; i++ {
		e.recordLinkMiss(0x1234)
	}
	e.profileTicks = 4095
	e.recordDispatchProfile(c)
	if len(e.LinkMissProfile()) != 0 {
		t.Fatal("live interval not cleared")
	}
	// Shutdown replaces the live table, not the retained interior snapshot.
	e.recordLinkMiss(0x5678)
	e.profileTicks = 8191
	e.profileLastNS = 0
	e.recordDispatchProfile(c)
	var samples []DispatchProfileSample
	if err := json.Unmarshal([]byte(e.DispatchProfileJSON()), &samples); err != nil {
		t.Fatal(err)
	}
	if len(samples) != 2 || len(samples[0].Misses) != 1 || len(samples[1].Misses) != 1 {
		t.Fatal("interval evidence lost", samples)
	}
	interior := samples[0].Misses[0]
	if interior.PC != 0x1234 || interior.Count != 7 || interior.Kind != 1 || interior.Bits != 0xd50342df || interior.Opcode != "control-write" || samples[1].Misses[0].PC != 0x5678 {
		t.Fatal("interior evidence overwritten by shutdown", samples)
	}
	if e.Stats.ProfileMissInterpreted != 7 || e.Stats.ProfileMissAbsent != 1 {
		t.Fatal("diagnostic snapshot changed cumulative counters")
	}
}
