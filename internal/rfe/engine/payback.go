package engine

import (
	emu "renvo.dev/internal/rfe/runtime"
	"sync/atomic"
	"time"
)

// paybackModel learns from real interpreter, compiler and native work. No
// calibration workload is run and cold one-pass code never requests compilation.
type paybackModel struct{ interpretedNS, interpreted, compileNS, compileOps, operations, instructions, leafNS, leafInstructions uint64 }

func (e *Engine) interpretedCost() uint64 {
	if e.payback.interpreted == 0 {
		return 0
	}
	return e.payback.interpretedNS / e.payback.interpreted
}
func (e *Engine) compilationCost(instructions int) uint64 {
	m := &e.payback
	if m.compileOps == 0 || m.instructions == 0 {
		return 0
	} // bootstrap from first actual reuse
	return uint64(instructions) * m.operations / m.instructions * m.compileNS / m.compileOps
}
func (e *Engine) observeCompilation(start time.Time, ops, instructions int) {
	if !e.config.Adaptive {
		return
	}
	m := &e.payback
	m.compileNS += uint64(time.Since(start))
	m.compileOps += uint64(ops)
	m.operations += uint64(ops)
	m.instructions += uint64(instructions)
}
func (e *Engine) observeLeaf(block *compiledBlock, start time.Time, retired uint64) {
	if !e.config.Adaptive {
		return
	}
	ns := uint64(time.Since(start))
	e.payback.leafNS += ns
	e.payback.leafInstructions += retired
	if ns >= block.sessionDebt {
		block.sessionDebt = 0
	} else {
		block.sessionDebt -= ns
	}
}
func (e *Engine) regionPays(block *compiledBlock, count int) bool {
	if !e.config.Adaptive {
		return true
	}
	// Region fusion saves host dispatches. Use measured leaf costs rather than
	// assuming that a loop or a particular instruction count is profitable.
	if e.payback.leafInstructions == 0 {
		return false
	}
	saving := e.payback.leafNS / e.payback.leafInstructions
	return block.visits*uint64(count)*saving >= e.compilationCost(count)
}
func (e *Engine) sessionResult(block *compiledBlock, start time.Time, context *emu.MemoryContext, budget uint64) {
	if !e.config.Adaptive || budget <= uint64(len(block.instructions)) {
		return
	}
	// True asynchronous scheduling events and successful long runs are not a
	// failed optimization. A missing mapping is retried immediately after refill.
	ns := uint64(time.Since(start))
	if context.Stop != nil && atomic.LoadUint32(context.Stop) != 0 {
		return
	}
	baseline := uint64(0)
	if e.payback.leafInstructions != 0 {
		baseline = context.Total * e.payback.leafNS / e.payback.leafInstructions
	}
	if ns < baseline {
		block.sessionDebt, block.sessionLoss = 0, 0
		return
	}
	// Failed experiments accrue their measured loss, not a fixed instruction
	// threshold/retry count. Ordinary leaves pay it down with real host work.
	block.sessionLoss += ns - baseline
	block.sessionDebt = block.sessionLoss

}
