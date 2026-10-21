package engine

import (
	"fmt"
	emu "renvo.dev/internal/rfe/runtime"
	"time"
)

func (e *Engine) stepMemory(c CPU, block *compiledBlock) error {
	memory, ok := c.MemoryBus().(NativeMemory)
	if !ok || e.native == nil || !block.native && block.visits < e.nativeThreshold(block) {
		return e.executeCached(c, block.instructions[0])
	}
	context := memory.NativeContext()
	if context == nil {
		return e.executeCached(c, block.instructions[0])
	}
	if !e.prepareMemoryBlock(c, block) {
		return e.executeCached(c, block.instructions[0])
	}
	if !block.native {
		return e.executeCached(c, block.instructions[0])
	}
	if len(block.instructions) > 16 {
		start := time.Time{}
		if e.config.Adaptive {
			start = time.Now()
		}
		before := *c.Retirement()
		err := e.runLinked(c, uint64(len(block.instructions)), block, memory, context, block.context, 1)
		e.observeLeaf(block, start, *c.Retirement()-before)
		return err
	}
	e.publishLeaf(c, block, context)
	start := time.Time{}
	if e.config.Adaptive {
		start = time.Now()
	}
	err := e.native.CallMemory(block.entry, c.Registers(), context)
	if err != nil {
		return err
	}
	e.observeLeaf(block, start, context.Retired)
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
		}
		// A trusted code-versioned owner exits before executable/PTE stores.
		// A native prefix cannot have changed the remaining opcode. Recheck
		// after refill too, since architectural mapping metadata can alias code.
		if owner, ok := memory.(CodeVersionedNativeMemory); ok && owner.NativeCodeVersions() && owner.CodeContext() == block.context && (block.proof == nil || block.proof.Valid()) && retired < uint64(len(block.instructions)) {
			ins := block.instructions[retired]
			if c.Registers()[e.arch.PC] == ins.PC {
				return e.executeCached(c, ins)
			}
		}
		return e.interpret(c)
	}
	block.successor, block.successorSeen = c.Registers()[e.arch.PC], true
	e.observeEdge(block.instructions[len(block.instructions)-1], c.Registers()[e.arch.PC])
	return nil
}

func (e *Engine) prepareMemoryBlock(c CPU, block *compiledBlock) bool {
	if block.attempted {
		return true
	}
	defer pauseTranslationClock(c)()
	start := time.Time{}
	if e.config.Adaptive {
		start = time.Now()
	}
	if !block.attempted {
		block.attempted = true
		wide := len(block.instructions) > 16
		b := emu.Builder{MemoryInstructionProgress: wide}
		pc := c.Registers()[e.arch.PC]
		b.Store(e.arch.PC, b.Constant(pc))
		for i, ins := range block.instructions {
			if !e.arch.Lower(&b, ins, i) {
				e.Stats.CompileFailures++
				e.discard(pc)
				return false
			}
		}
		b.Checkpoint(e.arch.StateWords, len(block.instructions))
		if wide {
			b.LoopContinue(b.Constant(0))
		}
		block.ops = b.FinishMemory(e.arch.StateWords)
		var entry int
		var err error
		if wide {
			if owner, ok := c.MemoryBus().(CodeVersionedNativeMemory); ok && owner.NativeCodeVersions() {
				entry, err = e.native.CompileCodeVersionedLoop(block.ops, e.arch.StateWords, len(block.instructions), owner.NativeContext())
			} else {
				entry, err = e.native.CompileLoop(block.ops, e.arch.StateWords, len(block.instructions))
			}
		} else {
			entry, err = e.native.CompileMemory(block.ops, e.arch.StateWords)
		}
		e.observeCompilation(start, len(block.ops), len(block.instructions))
		if err != nil {
			e.Stats.CompileFailures++
		} else {
			block.entry, block.native = entry, true
			_ = e.native.NameEntry(entry, fmt.Sprintf("%s_%x_memory_%d_ops", e.arch.Name, pc, len(block.ops)))
			e.Stats.ProfileErrors = e.native.SymbolErrors
			e.Stats.Promotions++
			if e.config.DynamicCode {
				block.ops = nil
				block.prepared = nil
			}
			if wide {
				block.regionEntry, block.regionInstructions, block.region = entry, len(block.instructions), true
				block.regionProof = block.proof
				block.regionAttempted, block.regionGeneration = true, e.Stats.Promotions
			}
		}
		e.Stats.NativeBytes = e.native.Bytes
	}
	return true
}
