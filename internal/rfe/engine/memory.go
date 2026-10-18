package engine

import (
	"fmt"
	emu "renvo.dev/internal/rfe/runtime"
)

func (e *Engine) stepMemory(c CPU, block *compiledBlock) error {
	memory, ok := c.MemoryBus().(NativeMemory)
	if !ok || e.native == nil || block.visits < e.nativeThreshold(block) {
		return e.interpret(c)
	}
	context := memory.NativeContext()
	if context == nil {
		return e.interpret(c)
	}
	if !e.prepareMemoryBlock(c, block) {
		return e.interpret(c)
	}
	if !block.native {
		return e.interpret(c)
	}
	err := e.native.CallMemory(block.entry, c.Registers(), context)
	if err != nil {
		return err
	}
	return e.accountMemory(c, block, memory, context)
}

// A slow exit has committed exactly the instructions preceding the access. The
// architecture performs the faulting/slow instruction once, never replays a
// prefix that may contain stores, and reports its original PC/opcode/cause.
func (e *Engine) accountMemory(c CPU, block *compiledBlock, memory NativeMemory, context *emu.MemoryContext) error {
	retired := context.Retired
	if retired > uint64(len(block.instructions)) || context.Status == 0 && retired != uint64(len(block.instructions)) {
		return fmt.Errorf("invalid native memory progress")
	}
	*c.Retirement() += retired
	e.Stats.Native += retired
	e.Stats.NativeMemory += uint64(block.memoryPrefix[retired])
	if context.Status != 0 {
		e.Stats.MemoryExits++
		if e.config.ProfileNative {
			e.recordMemoryExit(c.Registers()[e.arch.PC], context)
		}
		if context.Status == 1 {
			memory.NativeRefill(context.Address)
			if e.memoryRetryAllowed && !e.memoryRetry && !e.inNativeBatch && retired < uint64(len(block.instructions)) && refillReady(context) {
				return e.retryMemory(c, uint64(len(block.instructions))-retired)
			}
		}
		return e.interpret(c)
	}
	e.observeEdge(block.instructions[len(block.instructions)-1], c.Registers()[e.arch.PC])
	return nil
}

func (e *Engine) prepareMemoryBlock(c CPU, block *compiledBlock) bool {
	if block.attempted {
		return true
	}
	if clock := pauseTranslationClock(c); clock != nil {
		defer resumeTranslationClock(clock)
	}
	if !block.attempted {
		block.attempted = true
		pc := c.Registers()[e.arch.PC]
		if block.ops != nil {
			e.reuseTranslation(block)
		}
		if block.ops == nil {
			// Linked dispatch accounts guest memory instructions, not the number
			// of accesses (an x86 RMW or MOVS can issue several in one instruction).
			b, low, high := e.beginTranslation(true)
			b.Store(e.arch.PC, b.Constant(pc))
			for i, ins := range block.instructions {
				if !e.arch.Lower(b, ins, i) {
					e.endTranslation(low, high)
					e.Stats.CompileFailures++
					e.discard(pc)
					return false
				}
			}
			b.Checkpoint(e.arch.StateWords, len(block.instructions))
			block.ops = e.publishTranslation(block, e.finishTranslation(b, true, low, high))
		}
		entry, err := e.native.CompileMemory(block.ops, e.arch.StateWords)
		if err != nil {
			e.Stats.CompileFailures++
		} else {
			block.entry, block.native = entry, true
			e.nameEntry(entry, pc, len(block.ops), "memory")
			e.Stats.ProfileErrors = e.native.SymbolErrors
			e.Stats.Promotions++
		}
		e.Stats.NativeBytes = e.native.Bytes
		e.releaseTranslation(block)
	}
	return true
}
