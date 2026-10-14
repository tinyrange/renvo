package engine

import "encoding/json"

// DispatchProfileSample is a cumulative host-monotonic snapshot. Adjacent
// samples can isolate phases without recognizing benchmark markers or guest PCs.
type DispatchProfileSample struct {
	MonotonicNS uint64
	Stats       Stats
	Mapping     [4]uint64              // table writes, executable-page writes, root selections, all invalidations
	Misses      []DispatchMissSample   `json:",omitempty"`
	Memory      []DispatchMemorySample `json:",omitempty"`
}

// DispatchMissSample deliberately omits decoder-owned objects: bounded numeric
// evidence is sufficient to recover the opcode without retaining object graphs.
type DispatchMissSample struct {
	PC     uint64 `json:"p"`
	Count  uint64 `json:"n"`
	Bits   uint64 `json:"b"`
	Kind   uint8  `json:"k"`
	Opcode string `json:"o,omitempty"`
}

func (e *Engine) recordDispatchProfile(c CPU) {
	e.profileTicks++
	if e.profileTicks&4095 != 0 || len(e.profileTimeline) >= 512 {
		return
	}
	now := profileMonotonicNS()
	if now == 0 || e.profileLastNS != 0 && now-e.profileLastNS < 2000000000 {
		return
	}
	sample := DispatchProfileSample{MonotonicNS: now, Stats: e.Stats}
	if memory, ok := c.MemoryBus().(interface{ ProfileMappingCounters() [4]uint64 }); ok {
		sample.Mapping = memory.ProfileMappingCounters()
	}
	// Save the interval before clearing it. Previously only the last (often
	// shutdown) window survived; cumulative counters could not identify its
	// benchmark-interior unsupported instructions. Eight hottest PCs bound
	// wire size and remain approximate diagnostic counts, never authority.
	for i, miss := range e.LinkMissProfile() {
		if i == 8 {
			break
		}
		value := DispatchMissSample{PC: miss.PC, Count: miss.Count, Bits: miss.Bits, Kind: miss.Kind}
		if decoder, ok := miss.Decoded.(interface{ ProfileOpcode() string }); ok {
			value.Opcode = decoder.ProfileOpcode()
			if len(value.Opcode) > 32 {
				value.Opcode = value.Opcode[:32]
			}
		}
		sample.Misses = append(sample.Misses, value)
	}
	sample.Memory = e.memoryExitProfile()
	e.profileTimeline = append(e.profileTimeline, sample)
	e.profileLastNS = now
	// Keep hot-PC diagnostics recent rather than dominated by cold boot. The
	// cumulative class counters above remain untouched. This is the same
	// generic host cadence, never a guest marker/PC-dependent reset.
	if e.profileMisses != nil {
		clear(e.profileMisses[:])
	}
	if e.profileMemory != nil {
		clear(e.profileMemory[:])
	}
}

// DispatchProfileJSON is diagnostic-only, bounded by 512 snapshots per engine.
func (e *Engine) DispatchProfileJSON() string {
	if !e.config.ProfileNative {
		return "[]"
	}
	data, err := json.Marshal(e.profileTimeline)
	if err != nil {
		return "[]"
	}
	return string(data)
}
