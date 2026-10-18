//go:build browser

package engine

import emu "renvo.dev/internal/rfe/runtime"

func (e *Engine) rememberNativeRecycle(old *emu.Native) {
	for _, block := range e.blocks {
		block.recycleEntry, block.recycleRegionEntry = 0, 0
		if block.native {
			block.recycleEntry = block.entry
		}
		if block.native && block.region {
			block.recycleRegionEntry = block.regionEntry
		}
	}
}
func beginNativeRecycle(old *emu.Native) error { return old.SuspendForRecycle() }

// Inactive metadata remains only until the same quiescent validation/selection
// completes. rewatchDecoded validates the complete region dependency union.
func resetRegionForRecycle(block *compiledBlock) { block.regionEntry = 0 }

func (e *Engine) recycleNative(old *emu.Native, capacity int) (*emu.Native, error) {
	if len(e.recycleEntries) == 0 {
		e.recycleEntries = make([]int, emu.BrowserRecycleEntryLimit)
		e.recycleBlocks = make([]*compiledBlock, emu.BrowserRecycleEntryLimit)
		e.recycleRegions = make([]bool, emu.BrowserRecycleEntryLimit)
	}
	count, bytes := 0, 0
	thresholds := [3]uint64{256, 64, 16}
	for tier, minimum := range thresholds {
		for _, block := range e.blocks {
			if count == len(e.recycleEntries) {
				break
			}
			if block.recycleEntry == 0 || block.visits < minimum || tier != 0 && block.visits >= thresholds[tier-1] {
				continue
			}
			size := old.TransferEntryBytes(block.recycleEntry)
			if size == 0 || size > old.TransferByteBudget()-bytes {
				continue
			}
			e.recycleEntries[count] = block.recycleEntry
			e.recycleBlocks[count] = block
			e.recycleRegions[count] = false
			count++
			bytes += size
			regionSize := old.TransferEntryBytes(block.recycleRegionEntry)
			if count < len(e.recycleEntries) && regionSize != 0 && regionSize <= old.TransferByteBudget()-bytes {
				e.recycleEntries[count] = block.recycleRegionEntry
				e.recycleBlocks[count] = block
				e.recycleRegions[count] = true
				count++
				bytes += regionSize
			}
		}
	}
	for _, block := range e.blocks {
		block.recycleEntry, block.recycleRegionEntry = 0, 0
	}
	n, err := old.RecycleKeeping(e.recycleEntries[:count])
	if err != nil {
		old.Close()
		clear(e.recycleBlocks)
		return nil, err
	}
	for i, block := range e.recycleBlocks[:count] {
		if e.recycleRegions[i] {
			block.regionEntry = e.recycleEntries[i]
			block.region, block.regionAttempted = true, true
			block.regionGeneration = e.Stats.Promotions
		} else {
			block.entry = e.recycleEntries[i]
			block.native, block.attempted = true, true
		}
	}
	for _, block := range e.blocks {
		if !block.region {
			block.regionInstructions, block.regionPrefix = 0, [17]uint8{}
		}
	}
	clear(e.recycleBlocks)
	e.Stats.RecycledEntries += uint64(count)
	return n, nil
}
