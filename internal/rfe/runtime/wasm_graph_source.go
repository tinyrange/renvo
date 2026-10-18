package runtime

// The optional hint must not change architectural progress for unusual layouts.
// Real browser contexts own a separate typed word; aliases simply omit the hint.
func wasmGraphLinkSourceSafe(l WasmMemoryLayout) bool {
	if l.LinkSource == 0 || uint64(l.LinkSource)+8 > uint64(1)<<32 {
		return false
	}
	for _, offset := range []uint32{l.Retired, l.MemoryRetired, l.Status, l.Address, l.Total, l.MemoryTotal, l.Remaining, l.Linked, l.LoopExits, l.LoopIterations, l.Clock, l.ServiceSP, l.ServiceFP, l.ServiceWrite} {
		if uint64(l.LinkSource) < uint64(offset)+8 && uint64(offset) < uint64(l.LinkSource)+8 {
			return false
		}
	}
	slots := l.PageSlots
	if slots == 0 {
		slots = 64
	}
	if uint64(l.LinkSource) < uint64(l.Pages)+uint64(slots)*uint64(l.PageSize) && uint64(l.Pages) < uint64(l.LinkSource)+8 {
		return false
	}
	blockBytes := uint64(12)
	if !l.BlocksIndirect {
		sets := l.BlockSets
		if sets == 0 {
			sets = 256
		}
		blockBytes = 4 * uint64(sets) * uint64(l.DescriptorSize)
	}
	if uint64(l.LinkSource) < uint64(l.Blocks)+blockBytes && uint64(l.Blocks) < uint64(l.LinkSource)+8 {
		return false
	}
	return true
}
