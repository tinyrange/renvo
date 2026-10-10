package engine

import (
	"fmt"
	emu "renvo.dev/internal/rfe/runtime"
)

type EngineConfig struct {
	Mode                  string // interpreter, ir, native; native includes both lower tiers
	IRThreshold           uint64
	NativeThreshold       uint64
	MaxBlocks             int
	MaxInstructions       int // cold block discovery width
	MaxRegionInstructions int // independently optimized region width
	MaxNativeInstructions int // bounded uninterrupted scheduling slice
	NativeBytes           int
	ProfileNative         bool // emit Linux perf-map symbols; enabled by CLI -stats
}

func DefaultEngineConfig() EngineConfig {
	return EngineConfig{Mode: "native", IRThreshold: 8, NativeThreshold: 64, MaxBlocks: 4096, MaxInstructions: 16, MaxRegionInstructions: 256, MaxNativeInstructions: 65536, NativeBytes: 8 << 20}
}

type Stats struct {
	LoopIterations                                         uint64 // native loop iterations, without a Go transition
	NativeSessions, LinkMissStops, BudgetStops, OtherStops uint64

	Regions, RegionExits                               uint64 // bounded cyclic regions and successful side exits
	ProfileErrors                                      uint64
	Linked                                             uint64 // instructions retired inside the native dispatcher
	Interpreted, IR, Native                            uint64 // retired instructions in each tier
	Blocks, Promotions, Invalidations, CompileFailures uint64
	NativeBytes                                        int
	NativeMemory, MemoryExits                          uint64 // native scalar instructions and precise slow exits
	CodeReads, VersionChecks, ContextChecks            uint64 // translation discovery/revalidation only
}

type compiledBlock struct {
	regionAttempted, region         bool
	regionGeneration                uint64 // retry only after another native promotion
	regionEntry, regionInstructions int
	regionPrefix                    [17]uint8
	dependencies                    []codeDependency
	context                         CodeStamp
	instructions                    []Instruction
	visits                          uint64
	ops                             []emu.Op
	prepared                        *emu.PreparedBlock
	entry                           int
	native, attempted               bool
	interpretOnly                   bool // cached instruction word, still architecturally executed
	effects                         bool
	memoryPrefix                    [17]uint8 // scalar accesses retired in each prefix of the block
}

type dispatchEntry struct {
	pc    uint64
	block *compiledBlock
}

type Engine struct {
	arch        Architecture
	config      EngineConfig
	blocks      map[uint64]*compiledBlock
	dispatch    [1024]dispatchEntry
	ir          emu.IRMachine
	native      *emu.Native
	Stats       Stats
	linkContext CodeStamp
	edgeCounts  map[uint64][2]uint64 // bounded by discovered conditional instructions
	linkReady   bool                 // immutable state ABI, prepared once for this native engine
}

func New(config EngineConfig, arch Architecture) (*Engine, error) {
	if arch.Name == "" || arch.StateWords < 1 || arch.StateWords > 256 || arch.PC < 0 || arch.PC >= arch.StateWords || arch.Alignment == 0 || arch.Alignment > 8 || arch.Alignment&(arch.Alignment-1) != 0 || arch.Fetch == nil || arch.Lower == nil {
		return nil, fmt.Errorf("invalid guest architecture")
	}
	if config.Mode != "interpreter" && config.Mode != "ir" && config.Mode != "native" {
		return nil, fmt.Errorf("invalid engine %q", config.Mode)
	}
	if config.MaxRegionInstructions == 0 {
		config.MaxRegionInstructions = config.MaxInstructions
	}
	if config.MaxNativeInstructions == 0 {
		config.MaxNativeInstructions = 64
	}
	if config.MaxRegionInstructions < 1 || config.MaxRegionInstructions > 256 || config.MaxNativeInstructions < 1 || config.MaxNativeInstructions > emu.NativeSessionLimit {
		return nil, fmt.Errorf("invalid execution slice configuration")
	}
	if config.IRThreshold == 0 || config.NativeThreshold < config.IRThreshold || config.MaxBlocks < 1 || config.MaxBlocks > 65536 || config.MaxInstructions < 1 || config.MaxInstructions > 16 || config.NativeBytes < 4096 || config.NativeBytes > 64<<20 {
		return nil, fmt.Errorf("invalid engine budget")
	}
	e := &Engine{arch: arch, config: config, blocks: map[uint64]*compiledBlock{}}
	if config.Mode == "native" {
		n, err := emu.NewNative(config.NativeBytes)
		if err != nil {
			e.Stats.CompileFailures++
		} else {
			e.native = n
			if config.ProfileNative {
				_ = n.EnablePerfMap()
				e.Stats.ProfileErrors = n.SymbolErrors
			}
		}
	}
	return e, nil
}
func (e *Engine) Close() error {
	if e.native != nil {
		err := e.native.Close()
		e.native = nil
		return err
	}
	return nil
}
func (e *Engine) block(c CPU) *compiledBlock {
	pc := c.Registers()[e.arch.PC]
	if pc&(e.arch.Alignment-1) != 0 || c.MemoryBus() == nil {
		return nil
	}
	cache := &e.dispatch[(pc>>2)&1023]
	old := cache.block
	direct := old != nil && cache.pc == pc
	if !direct {
		old = e.blocks[pc]
	}
	if old == nil {
		return e.discoverBlock(c, pc, cache)
	}
	valid := false
	if len(old.dependencies) != 0 {
		var context CodeStamp
		if memory, ok := c.MemoryBus().(CodeContextMemory); ok {
			context = memory.CodeContext()
			e.Stats.ContextChecks++
			valid = context.Identity != nil && context == old.context
		}
		if !valid {
			valid = e.validVersions(c, old, context)
		}
	} else {
		valid = e.validBytes(c, pc, old)
	}
	if !valid {
		e.discard(pc)
		e.Stats.Invalidations++
		return nil
	}
	// A direct hit already contains exactly this pointer; do not rewrite its
	// GC-tracked slot on every executed block. Map hits still repair collisions.
	if !direct {
		*cache = dispatchEntry{pc, old}
	}
	return old
}

// Keep decoding, allocation and the large instruction descriptor out of the
// guarded lookup's frame; none of these are needed for a hot block entry.
func (e *Engine) discoverBlock(c CPU, pc uint64, cache *dispatchEntry) *compiledBlock {
	if len(e.blocks) >= e.config.MaxBlocks {
		return nil
	}
	block := &compiledBlock{}
	nativeMemory, capable := c.MemoryBus().(NativeMemory)
	allowMemory := capable && e.native != nil && nativeMemory.NativeContext() != nil
	addr := pc
	for i := 0; i < e.config.MaxInstructions; i++ {
		ins, err := e.arch.Fetch(c.MemoryBus(), addr)
		e.Stats.CodeReads++
		if err != nil || ins.PC != addr || ins.Length == 0 || ins.Length > 15 || uint64(ins.Length) > ^uint64(0)-addr || uint64(ins.Length)%e.arch.Alignment != 0 {
			break
		}
		if ins.InterpretOnly || ins.Memory && !allowMemory {
			if len(block.instructions) == 0 {
				block.instructions = append(block.instructions, ins)
				block.interpretOnly = true
			}
			break
		}
		block.instructions = append(block.instructions, ins)
		count := len(block.instructions)
		block.memoryPrefix[count] = block.memoryPrefix[count-1]
		if ins.Memory {
			block.effects = true
			block.memoryPrefix[count]++
		}
		if ins.Terminal {
			break
		}
		addr += uint64(ins.Length)
	}
	if len(block.instructions) == 0 {
		return nil
	}
	e.dependencies(c, pc, block)
	e.blocks[pc] = block
	*cache = dispatchEntry{pc, block}
	e.Stats.Blocks++
	return block
}

// Step retires at most remaining instructions. Code page stamps (or bytes for
// unversioned memories) and execute permission are checked on every block entry.
// Code cache exhaustion always falls back; invalid native entries are never
// reused or reclaimed underneath a call.
func (e *Engine) Step(c CPU, remaining uint64) error {
	if err := e.validateCPU(c); err != nil {
		return err
	}
	if remaining == 0 {
		return fmt.Errorf("instruction budget exhausted")
	}
	if e.config.Mode == "interpreter" {
		return e.interpret(c)
	}
	return e.stepBlock(c, remaining, e.block(c))
}

// stepBlock consumes a lookup already validated in this single-threaded engine.
// This lets Run fall back without a second lookup/context query at the same PC.
func (e *Engine) stepBlock(c CPU, remaining uint64, block *compiledBlock) error {
	if block == nil || uint64(len(block.instructions)) > remaining {
		return e.interpret(c)
	}
	if block.interpretOnly {
		// block() has validated this instruction's execute permission and
		// code version (or bytes). Avoid fetching it a second time. Memory,
		// traps and unsupported instructions keep CPU semantics.
		before := *c.Retirement()
		err := c.ExecuteInstruction(block.instructions[0])
		e.Stats.Interpreted += *c.Retirement() - before
		return err
	}
	block.visits++
	if block.effects {
		return e.stepMemory(c, block)
	}
	if block.visits < e.config.IRThreshold {
		return e.interpret(c)
	}
	if block.ops == nil {
		var b emu.Builder
		// This translation is reachable only through its guarded PC key.
		// Specialize PC-relative values and sequential PC updates at build time.
		b.Store(e.arch.PC, b.Constant(c.Registers()[e.arch.PC]))
		for i, ins := range block.instructions {
			if !e.arch.Lower(&b, ins, i) {
				e.Stats.CompileFailures++
				e.discard(c.Registers()[e.arch.PC])
				return e.interpret(c)
			}
		}
		block.ops = b.Finish(e.arch.StateWords)
		prepared, err := emu.Prepare(block.ops, e.arch.StateWords)
		if err != nil {
			e.Stats.CompileFailures++
			e.discard(c.Registers()[e.arch.PC])
			return e.interpret(c)
		}
		block.prepared = prepared
	}
	if e.native != nil && !block.attempted && block.visits >= e.config.NativeThreshold {
		block.attempted = true
		entry, err := e.native.Compile(block.ops, e.arch.StateWords)
		if err != nil {
			e.Stats.CompileFailures++
		} else {
			block.entry = entry
			_ = e.native.NameEntry(entry, fmt.Sprintf("%s_%x_pure_%d_ops", e.arch.Name, c.Registers()[e.arch.PC], len(block.ops)))
			e.Stats.ProfileErrors = e.native.SymbolErrors
			block.native = true
			e.Stats.Promotions++
		}
		e.Stats.NativeBytes = e.native.Bytes
	}
	var err error
	if block.native && e.native != nil {
		err = e.native.Call(block.entry, c.Registers())
	} else {
		err = e.ir.Run(block.prepared, c.Registers())
	}
	if err != nil {
		return err
	}
	e.observeEdge(block.instructions[len(block.instructions)-1], c.Registers()[e.arch.PC])
	retired := uint64(len(block.instructions))
	*c.Retirement() += retired
	if block.native && e.native != nil {
		e.Stats.Native += retired
	} else {
		e.Stats.IR += retired
	}
	return nil
}
func (e *Engine) interpret(c CPU) error {
	before := *c.Retirement()
	block := e.blocks[c.Registers()[e.arch.PC]]
	err := c.Step()
	e.Stats.Interpreted += *c.Retirement() - before
	if err == nil && *c.Retirement() == before+1 && block != nil && len(block.instructions) != 0 {
		e.observeEdge(block.instructions[0], c.Registers()[e.arch.PC])
	}
	return err
}

// NativeAvailable and CachedBlocks expose diagnostics without mutable caches.
func (e *Engine) NativeAvailable() bool { return e.native != nil }
func (e *Engine) CachedBlocks() int     { return len(e.blocks) }

func (e *Engine) validateCPU(c CPU) error {
	if e == nil || c == nil || len(c.Registers()) != e.arch.StateWords || c.Retirement() == nil {
		return fmt.Errorf("guest state does not match architecture")
	}
	return nil
}
