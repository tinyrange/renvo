package engine

import emu "renvo.dev/internal/rfe/runtime"

// CodeRange is a half-open execute-checked instruction extent. Regions may
// contain multiple noncontiguous extents; bytes between them are not code.
type CodeRange struct{ Start, End uint64 }

// CodeProofMemory replaces the coarse mutation epoch ONLY for owners which
// synchronously revoke every affected proof and native publication before a
// code/mapping/permission change. Full-context changes must revoke all proofs.
// No guest mutation may run concurrently with the engine or a native session.
type CodeProofMemory interface {
	CodeContextMemory
	CodeProofsEnabled() bool
	CodeMutation() uint64
	ProveCode([]CodeRange, uint64) CodeProof
}

// CodeProof is owned host metadata, never guest authority. Bind records the
// exact public descriptor to revoke before mutation; private native admissions
// cannot bypass that publication in a later execution session.
type CodeProof interface {
	Valid() bool
	Revalidate() bool
	Bind(*emu.NativeLink, int)
	Close()
}

func codeRanges(instructions []Instruction) []CodeRange {
	ranges := make([]CodeRange, 0, 1)
	for _, ins := range instructions {
		end := ins.PC + uint64(ins.Length)
		if n := len(ranges); n > 0 && ranges[n-1].End == ins.PC {
			ranges[n-1].End = end
		} else {
			ranges = append(ranges, CodeRange{ins.PC, end})
		}
	}
	return ranges
}
func (e *Engine) invalidateRegion(block *compiledBlock) {
	if !block.region {
		return
	}
	if block.regionProof != nil && block.regionProof != block.proof {
		block.regionProof.Close()
	}
	if e.config.DynamicCode && block.regionEntry != block.entry {
		e.pendingReleases = append(e.pendingReleases, block.regionEntry)
	}
	block.regionProof = nil
	block.region = false
	block.regionAttempted = false
	block.regionGeneration = 0
}
