package engine

import (
	"fmt"
	emu "renvo.dev/internal/rfe/runtime"
)

func (e *Engine) stepMemory(c CPU, block *compiledBlock) error {
	memory, ok := c.MemoryBus().(NativeMemory)
	if !ok || e.native == nil || block.visits < e.config.NativeThreshold {
		return e.interpret(c)
	}
	context := memory.NativeContext()
	if context == nil {
		return e.interpret(c)
	}
	if !block.attempted {
		block.attempted = true
		var b emu.Builder
		pc := c.Registers()[e.arch.PC]
		b.Store(e.arch.PC, b.Constant(pc))
		for i, ins := range block.instructions {
			if !e.arch.Lower(&b, ins, i) {
				e.Stats.CompileFailures++
				e.discard(pc)
				return e.interpret(c)
			}
		}
		b.Checkpoint(e.arch.StateWords, len(block.instructions))
		block.ops = b.FinishMemory(e.arch.StateWords)
		entry, err := e.native.CompileMemory(block.ops, e.arch.StateWords)
		if err != nil {
			e.Stats.CompileFailures++
		} else {
			block.entry, block.native = entry, true
			_ = e.native.NameEntry(entry, fmt.Sprintf("%s_%x_memory_%d_ops", e.arch.Name, pc, len(block.ops)))
			e.Stats.ProfileErrors = e.native.SymbolErrors
			e.Stats.Promotions++
		}
		e.Stats.NativeBytes = e.native.Bytes
	}
	if !block.native {
		return e.interpret(c)
	}
	if err := e.native.CallMemory(block.entry, c.Registers(), context); err != nil {
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
		if context.Status == 1 {
			memory.NativeRefill(context.Address)
		}
		return e.interpret(c)
	}
	return nil
}
