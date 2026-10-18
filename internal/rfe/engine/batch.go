package engine

import emu "renvo.dev/internal/rfe/runtime"

// Run executes one bounded dispatch quantum. Step remains the single-block API.
// Only already-native blocks and cached architectural instructions participate;
// discovery requiring promotion returns to Step outside the arena lock. Every
// block entry still validates executable code, including after guest stores.
func (e *Engine) Run(c CPU, remaining uint64) error {
	return e.run(c, remaining, 1)
}

// RunQuanta selects the configured native scheduling slice when supported.
// Conservative hosts retain separate short calls; Step is still single-block.
func (e *Engine) RunQuanta(c CPU, remaining uint64) error {
	e.memoryRetryAllowed = true
	err := e.run(c, remaining, 16)
	e.memoryRetryAllowed = false
	return err
}

func (e *Engine) run(c CPU, remaining uint64, quanta int) error {
	if e.config.ProfileNative {
		e.recordDispatchProfile(c)
	}
	if err := e.recycle(c); err != nil {
		return err
	}
	if err := e.validateCPU(c); err != nil {
		return err
	}
	if e.native == nil || remaining == 0 {
		return e.Step(c, remaining)
	}
	if first, memory, context := e.linkedSnapshot(c, remaining); first != nil {
		return e.runLinked(c, remaining, first, memory, context, first.context, quanta)
	}
	first := e.block(c)
	if first == nil || !first.native || first.instructions[0].ScheduleBoundary || uint64(len(first.instructions)) > remaining {
		return e.stepBlock(c, remaining, first)
	}
	if memory, ok := c.MemoryBus().(NativeMemory); ok {
		if context := memory.NativeContext(); context != nil {
			stamp := first.context // block() just validated this context in the same thread
			if stamp.Identity != nil {
				return e.runLinked(c, remaining, first, memory, context, stamp, quanta)
			}
		}
	}
	return e.runUnlinked(c, remaining, first)
}

// Isolate callback captures from the normal linked dispatcher path. Renvo's
// escape-preserving closure cells are needed only by this fallback batch.
func (e *Engine) runUnlinked(c CPU, remaining uint64, first *compiledBlock) error {
	budget := remaining
	if budget > 64 {
		budget = 64
	}
	var pending *compiledBlock
	memory, capable := c.MemoryBus().(NativeMemory)
	var context *emu.MemoryContext
	if capable {
		context = memory.NativeContext()
	}
	selectBlock := func(completed int) (int, bool, error) {
		// This callback runs after each successful native call (also after the
		// final call), never after a failed native entry check.
		if pending != nil {
			if pending.effects {
				before, status := *c.Retirement(), context.Status
				err := e.accountMemory(c, pending, memory, context)
				budget -= *c.Retirement() - before
				pending = nil
				if status != 0 || err != nil {
					return 0, false, err
				}
			} else {
				retired := uint64(len(pending.instructions))
				*c.Retirement() += retired
				e.Stats.Native += retired
				budget -= retired
				pending = nil
			}
		}
		for budget != 0 {
			block := first
			first = nil
			if block == nil {
				block = e.block(c)
			}
			if block == nil || block.instructions[0].ScheduleBoundary || uint64(len(block.instructions)) > budget {
				return 0, false, nil
			}
			if block.interpretOnly {
				before := *c.Retirement()
				err := c.ExecuteInstruction(block.instructions[0])
				retired := *c.Retirement() - before
				e.Stats.Interpreted += retired
				budget -= retired
				if err != nil {
					return 0, false, err
				}
				continue
			}
			if !block.native || block.effects && context == nil {
				return 0, false, nil
			}
			block.visits++
			pending = block
			return block.entry, true, nil
		}
		return 0, false, nil
	}
	var err error
	e.inNativeBatch = true // accountMemory callback must not recursively enter this arena
	if context == nil {
		_, err = e.native.CallBatch(c.Registers(), 64, selectBlock)
	} else {
		_, err = e.native.CallBatchMemory(c.Registers(), context, 64, selectBlock)
	}
	e.inNativeBatch = false
	return err
}
