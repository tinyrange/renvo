package engine

import (
	"fmt"
	emu "renvo.dev/internal/rfe/runtime"
)

type EngineConfig struct {
	Mode                   string // interpreter, ir, native; native includes both lower tiers
	RetainBrowserIR        bool   // opt-in bounded leaf-IR cache; no established whole-boot speedup
	DiscoveryThreshold     uint64 // optional bounded cold-PC observation before decoding
	RecycleNative          bool   // reclaim a full arena only at quiescent dispatch entry
	EvictCold              bool   // recycle decoded-cache slots; native arena stays bounded/append-only
	IRThreshold            uint64
	RegionThreshold        uint64 // optional visits before optimizing a compiled leaf into a region
	NativeThreshold        uint64
	MaxBlocks              int
	MaxInstructions        int // cold block discovery width
	MaxRegionInstructions  int // independently optimized region width
	MaxNativeInstructions  int // bounded uninterrupted scheduling slice
	NativeBytes            int
	ProfileNativeMonotonic bool // use CLOCK_MONOTONIC for sampling profiles
	ProfileNative          bool // emit Linux perf-map symbols; enabled by CLI -stats
}

func DefaultEngineConfig() EngineConfig {
	return EngineConfig{Mode: "native", IRThreshold: 8, NativeThreshold: 64, MaxBlocks: 4096, MaxInstructions: 16, MaxRegionInstructions: 256, MaxNativeInstructions: 65536, NativeBytes: 8 << 20}
}

type Stats struct {
	LeafTranslations, LeafIRReuses, IRCacheStores, IRCacheBypass, IRCacheDiscarded, IRCacheEvictions uint64
	IRCacheAllocated                                                                                 int
	RecycledEntries                                                                                  uint64
	ArenaResets                                                                                      uint64 // quiescent native arena replacements within the same byte cap
	Evictions                                                                                        uint64 // decoded cache replacements, not native arena reclamation
	LoopIterations                                                                                   uint64 // native loop iterations, without a Go transition
	NativeSessions, LinkMissStops, BudgetStops, OtherStops                                           uint64
	LinkContextChanges                                                                               uint64
	ProfileMissAbsent, ProfileMissInterpreted, ProfileMissCold, ProfileMissNative                    uint64

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
	cacheSlot                       int
	recent                          uint8
	regionAttempted, region         bool
	regionGeneration                uint64 // retry only after another native promotion
	prefillGeneration               uint64 // optional browser successor work, never execution authority
	prefillDirty                    bool
	browserDemand                   bool      // private preparation hotness, never execution authority
	linkSuccessors                  [4]uint64 // observed indirect/return targets; hints, not code proofs
	linkSuccessorCount, linkVictim  uint8
	regionEntry, regionInstructions int
	regionPrefix                    [17]uint8
	dependencies                    []codeDependency
	context                         CodeStamp
	instructions                    []Instruction
	visits                          uint64
	opStorage                       []emu.Op
	irUnits                         int
	irScore                         uint8 // observed compilation count, saturated; guest execution is not cache reuse
	irSlot                          int   // one-based cache owner slot; zero means borrowed or native-owned IR
	ops                             []emu.Op
	prepared                        *emu.PreparedBlock
	entry                           int
	recycleEntry                    int // browser-only candidate, revoked before rewatch
	recycleRegionEntry              int
	native, attempted               bool
	interpretOnly                   bool // cached instruction word, still architecturally executed
	effects                         bool
	memoryPrefix                    [17]uint8 // scalar accesses retired in each prefix of the block
}

type dispatchEntry struct {
	pc    uint64
	block *compiledBlock
}

type hotObservation struct{ pc, visits uint64 }

type Engine struct {
	fastRefillLow, fastRefillHigh int  // browser-only temporary setup scope
	memoryRetryAllowed            bool // only the explicitly batched RunQuanta API may retry
	inNativeBatch                 bool // host callbacks may hold the native arena lock
	memoryRetry                   bool // bound one fresh guarded dispatch per successfully refilled exit
	translationOps                [2048]emu.Op
	leafIR                        translationCache
	recycleEntries                []int
	recycleBlocks                 []*compiledBlock
	recycleRegions                []bool
	translation                   emu.Builder
	regionPartsScratch            [256]regionPart
	regionBestScratch             [256]regionPart
	regionCycleScratch            [256]regionPart
	regionResultScratch           [256]regionPart
	graph                         graphScratch
	hot                           [4096]hotObservation
	hotVictim                     uint64
	blockPool                     []*compiledBlock
	cacheOrder                    []dispatchEntry
	cacheNext                     int
	arch                          Architecture
	config                        EngineConfig
	blocks                        map[uint64]*compiledBlock
	dispatch                      [1024]dispatchEntry
	versionProofs                 [256]codePageProof  // bounded full-page/context proof reuse; never code ownership
	prefillLookup                 [1024]dispatchEntry // metadata only; revoked with the owning block
	ir                            emu.IRMachine
	native                        *emu.Native
	Stats                         Stats
	linkContext                   CodeStamp
	linkContexts                  map[*emu.MemoryContext]CodeStamp
	profileMisses                 *[1024]LinkMissSample
	profileMemory                 *[256]DispatchMemorySample
	profileTicks                  uint64
	profileLastNS                 uint64
	profileTimeline               []DispatchProfileSample
	edgeCounts                    map[uint64][2]uint64 // bounded by discovered conditional instructions
	linkReady                     bool                 // immutable state ABI, prepared once for this native engine
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
	if config.RecycleNative {
		e.blockPool = make([]*compiledBlock, 0, config.MaxBlocks)
	}
	if config.EvictCold {
		e.cacheOrder = make([]dispatchEntry, config.MaxBlocks)
	}
	if config.Mode == "native" {
		n, err := emu.NewNative(config.NativeBytes)
		if err != nil {
			e.Stats.CompileFailures++
		} else {
			e.native = n
			if config.ProfileNative {
				_ = n.EnablePerfMap(config.ProfileNativeMonotonic)
				e.Stats.ProfileErrors = n.SymbolErrors
			}
		}
	}
	return e, nil
}
func (e *Engine) Close() error {
	for _, block := range e.blocks {
		e.releaseInstructions(block)
	}
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
			// Complex owned decoder descriptors justify parsed-semantic reuse after
			// physical remapping, preserving discovery heat and any compiled leaf.
			// Cheap comparable instruction words retain the ordinary rediscovery path.
			if !valid && e.arch.EqualDecoded != nil {
				valid = e.revalidateIdenticalBlock(c, pc, old)
			}
		}
	} else {
		valid = e.validBytes(c, pc, old)
	}
	if !valid {
		e.discard(pc)
		e.Stats.Invalidations++
		return nil
	}
	old.recent = 1
	if old.native {
		old.recent = 2
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
	if clock := pauseTranslationClock(c); clock != nil {
		defer resumeTranslationClock(clock)
	}
	if e.config.DiscoveryThreshold > 1 {
		observation := e.discoveryObservation(pc, true)
		if observation.visits < e.config.DiscoveryThreshold {
			observation.visits++
		}
		if observation.visits < e.config.DiscoveryThreshold {
			return nil
		}
	}
	if len(e.blocks) >= e.config.MaxBlocks && !e.config.EvictCold {
		return nil
	}
	var block *compiledBlock
	if count := len(e.blockPool); count != 0 {
		block = e.blockPool[count-1]
		e.blockPool[count-1] = nil
		e.blockPool = e.blockPool[:count-1]
		instructions := block.instructions[:0]
		dependencies := block.dependencies[:0]
		ops := block.opStorage[:0]
		*block = compiledBlock{cacheSlot: -1, recent: 1, instructions: instructions, dependencies: dependencies, opStorage: ops}
	} else {
		block = &compiledBlock{cacheSlot: -1, recent: 1}
	}
	nativeMemory, capable := c.MemoryBus().(NativeMemory)
	allowMemory := capable && e.native != nil && nativeMemory.NativeContext() != nil
	addr := pc
	for i := 0; i < e.config.MaxInstructions; i++ {
		ins, err := e.arch.Fetch(c.MemoryBus(), addr)
		e.Stats.CodeReads++
		if err != nil || ins.PC != addr || ins.Length == 0 || ins.Length > 15 || uint64(ins.Length) > ^uint64(0)-addr || uint64(ins.Length)%e.arch.Alignment != 0 {
			e.releaseInstruction(ins)
			break
		}
		if ins.ScheduleBoundary && len(block.instructions) != 0 {
			e.releaseInstruction(ins)
			break // Clock/device scheduling reads own their host boundary.
		}
		if ins.InterpretOnly || ins.Memory && !allowMemory {
			if len(block.instructions) == 0 {
				block.instructions = append(block.instructions, ins)
				block.interpretOnly = true
			} else {
				e.releaseInstruction(ins)
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
		if ins.Terminal || ins.ScheduleBoundary {
			break
		}
		addr += uint64(ins.Length)
	}
	if len(block.instructions) == 0 {
		if len(e.blockPool) < cap(e.blockPool) {
			e.blockPool = append(e.blockPool, block)
		}
		return nil
	}
	e.dependencies(c, pc, block)
	if e.config.EvictCold {
		e.admit(pc, block)
	}
	e.blocks[pc] = block
	// An evicted PC must establish fresh heat, rather than retaining a
	// permanent discovery ticket and repeatedly decoding during cache churn.
	if e.config.DiscoveryThreshold > 1 {
		observation := e.discoveryObservation(pc, false)
		if observation != nil {
			observation.visits = 0
		}
	}
	*cache = dispatchEntry{pc, block}
	e.Stats.Blocks++
	return block
}

// Step retires at most remaining instructions. Code page stamps (or bytes for
// unversioned memories) and execute permission are checked on every block entry.
// Native arena exhaustion always falls back; invalid native entries are never
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
	if !e.preparePureBlock(c, block) {
		return e.interpret(c)
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

func (e *Engine) preparePureBlock(c CPU, block *compiledBlock) bool {
	if block.native && e.native != nil || block.prepared != nil && (e.native == nil || block.attempted || block.visits < e.nativeThreshold(block)) {
		return true
	}
	if block.ops != nil && (e.native == nil || block.attempted || block.visits < e.nativeThreshold(block)) {
		return true
	}
	if clock := pauseTranslationClock(c); clock != nil {
		defer resumeTranslationClock(clock)
	}
	if block.ops != nil {
		e.reuseTranslation(block)
	}
	if block.ops == nil {
		b, low, high := e.beginTranslation(false)
		// This translation is reachable only through its guarded PC key.
		// Specialize PC-relative values and sequential PC updates at build time.
		b.Store(e.arch.PC, b.Constant(c.Registers()[e.arch.PC]))
		for i, ins := range block.instructions {
			if !e.arch.Lower(b, ins, i) {
				e.endTranslation(low, high)
				e.Stats.CompileFailures++
				e.discard(c.Registers()[e.arch.PC])
				return false
			}
		}
		block.ops = e.publishTranslation(block, e.finishTranslation(b, false, low, high))
	}
	if e.native != nil && !block.attempted && block.visits >= e.nativeThreshold(block) {
		block.attempted = true
		entry, err := e.native.Compile(block.ops, e.arch.StateWords)
		if err != nil {
			e.Stats.CompileFailures++
		} else {
			block.entry = entry
			e.nameEntry(entry, c.Registers()[e.arch.PC], len(block.ops), "pure")
			e.Stats.ProfileErrors = e.native.SymbolErrors
			block.native = true
			e.Stats.Promotions++
		}
		e.Stats.NativeBytes = e.native.Bytes
	}
	if !block.native && block.prepared == nil {
		prepared, err := emu.Prepare(block.ops, e.arch.StateWords)
		if err != nil {
			e.Stats.CompileFailures++
			e.discard(c.Registers()[e.arch.PC])
			return false
		}
		block.prepared = prepared
	}
	e.releaseTranslation(block)
	return true
}
