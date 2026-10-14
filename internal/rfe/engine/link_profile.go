package engine

// LinkMissSample is bounded, approximate hot-PC diagnostic data, not exact
// counts for a time interval. Full tags prevent collisions conferring authority.
type LinkMissSample struct {
	PC, Count uint64
	Bits      uint64
	Kind      uint8 // 0 absent, 1 interpreted-only, 2 not native yet, 3 native
	Decoded   any   // immutable decoder classification; diagnostics, not authority
}

func (e *Engine) recordLinkMiss(pc uint64) {
	if e.profileMisses == nil {
		e.profileMisses = new([1024]LinkMissSample)
	}
	// Four candidates share each bounded set. Full PC tags remain diagnostic
	// only; evict the smallest retained count, not a colliding hot PC blindly.
	first := int(((pc>>1)^(pc>>12))&255) * 4
	slot := first
	for i := first; i < first+4; i++ {
		if e.profileMisses[i].PC == pc && e.profileMisses[i].Count != 0 {
			slot = i
			break
		}
		if e.profileMisses[i].Count < e.profileMisses[slot].Count {
			slot = i
		}
	}
	sample := &e.profileMisses[slot]
	if sample.PC != pc || sample.Count == 0 {
		*sample = LinkMissSample{PC: pc}
	}
	sample.Kind, sample.Decoded, sample.Bits = 0, nil, 0
	block := e.blocks[pc]
	if block == nil {
		e.Stats.ProfileMissAbsent++
	} else {
		if len(block.instructions) != 0 {
			sample.Decoded = block.instructions[0].Decoded
			sample.Bits = block.instructions[0].Bits
		}
		if block.interpretOnly {
			sample.Kind = 1
			e.Stats.ProfileMissInterpreted++
		} else if !block.native {
			sample.Kind = 2
			e.Stats.ProfileMissCold++
		} else {
			sample.Kind = 3
			e.Stats.ProfileMissNative++
		}
	}
	sample.Count++
}

// LinkMissProfile returns at most sixteen hottest slots since the latest
// generic host-time snapshot (approximately two seconds), not whole-run counts.
// Disabled profiling performs no per-PC recording and returns no samples.
func (e *Engine) LinkMissProfile() []LinkMissSample {
	if e.profileMisses == nil {
		return nil
	}
	top := make([]LinkMissSample, 0, 16)
	for _, sample := range e.profileMisses {
		if sample.Count == 0 {
			continue
		}
		at := 0
		for at < len(top) && top[at].Count >= sample.Count {
			at++
		}
		if at >= 16 {
			continue
		}
		if len(top) < 16 {
			top = append(top, LinkMissSample{})
		}
		for i := len(top) - 1; i > at; i-- {
			top[i] = top[i-1]
		}
		top[at] = sample
	}
	return top
}
