package engine

import (
	"fmt"
	emu "renvo.dev/internal/rfe/runtime"
	"time"
)

type EngineConfig struct {
	Adaptive               bool   // measured compilation/transition payback, not fixed heat quotas
	NaturalBlocks          bool   // end blocks at control flow, scheduling, or a code-page boundary; no artificial width
	DynamicCode            bool   // retain live blocks and grow native storage; retire invalidated fragments individually
	Mode                   string // interpreter, ir, native; native includes both lower tiers
	DiscoveryThreshold     uint64 // optional bounded cold-PC observation before decoding
	RecycleNative          bool   // reclaim a full arena only at quiescent dispatch entry
	CacheContexts          bool   // keep distinct CodeContext identities within the same global block cap
	EvictCold              bool   // recycle decoded-cache slots; native arena stays bounded/append-only
	IRThreshold            uint64
	LoopRegionsOnly        bool   // specialize/session only proven cyclic traces, not one-shot leaves
	RegionThreshold        uint64 // additional observed block visits before region specialization
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
	ArenaResets                                                                   uint64 // quiescent native arena replacements within the same byte cap
	Evictions                                                                     uint64 // decoded cache replacements, not native arena reclamation
	LoopIterations                                                                uint64 // native loop iterations, without a Go transition
	NativeSessions, LinkMissStops, BudgetStops, OtherStops                        uint64
	LinkContextChanges                                                            uint64
	ProfileMissAbsent, ProfileMissInterpreted, ProfileMissCold, ProfileMissNative uint64

	Regions, RegionExits                                     uint64 // bounded cyclic regions and successful side exits
	ProfileErrors                                            uint64
	Linked                                                   uint64 // instructions retired inside the native dispatcher
	Interpreted, IR, Native                                  uint64 // retired instructions in each tier
	Blocks, Promotions, Invalidations, CompileFailures       uint64
	NativeBytes                                              int
	NativeMemory, MemoryExits                                uint64 // native scalar instructions and precise slow exits
	CodeReads, VersionChecks, ContextChecks                  uint64 // translation discovery/revalidation only
	VersionIdentityMisses, VersionEpochMisses, VersionFaults uint64 // code-proof failure reasons
}

type compiledBlock struct {
	proof, regionProof              CodeProof
	sessionDebt                     uint64
	sessionLoss                     uint64
	successor                       uint64
	successorSeen                   bool
	cacheContext                    *CodeIdentity // immutable translation-domain key, not a code version
	cacheSlot                       int
	recent                          uint8
	regionAttempted, region         bool
	regionGeneration                uint64 // retry only after another native promotion
	regionEntry, regionInstructions int
	regionPrefix                    [17]uint8
	dependencies                    []codeDependency
	context                         CodeStamp
	instructions                    []Instruction
	visits                          uint64
	sessionRetry                    uint64 // observed short guarded sessions temporarily use ordinary native leaves
	sessionShort                    uint8
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

type hotObservation struct{ pc, visits uint64 }

type contextPC struct {
	pc       uint64
	identity *CodeIdentity
}

type Engine struct {
	dispatchShift uint

	codeLinks       map[codeLinkKey]*codeLinkState
	payback         paybackModel
	reused          map[contextPC]bool
	pendingReleases []int
	freeCacheSlots  []int
	cacheContext    *CodeIdentity
	variants        map[contextPC]*compiledBlock // optional variants share one global replacement clock
	hot             [4096]hotObservation
	hotVictim       uint64
	cacheOrder      []dispatchEntry
	cacheNext       int
	arch            Architecture
	config          EngineConfig
	blocks          map[uint64]*compiledBlock
	dispatch        [1024]dispatchEntry
	ir              emu.IRMachine
	native          *emu.Native
	Stats           Stats
	linkContext     CodeStamp
	linkContexts    map[*emu.MemoryContext]CodeStamp
	profileMisses   *[1024]LinkMissSample
	profileMemory   *[256]DispatchMemorySample
	profileTicks    uint64
	profileLastNS   uint64
	profileTimeline []DispatchProfileSample
	edgeCounts      map[uint64][2]uint64 // bounded by discovered conditional instructions
	linkReady       bool                 // immutable state ABI, prepared once for this native engine
}

func New(config EngineConfig, arch Architecture) (*Engine, error) {
	if arch.Name == "" || arch.StateWords < 1 || arch.StateWords > 256 || arch.PC < 0 || arch.PC >= arch.StateWords || arch.Alignment == 0 || arch.Alignment > 8 || arch.Alignment&(arch.Alignment-1) != 0 || arch.Fetch == nil || arch.Lower == nil {
		return nil, fmt.Errorf("invalid guest architecture")
	}
	if config.Mode != "interpreter" && config.Mode != "ir" && config.Mode != "native" {
		return nil, fmt.Errorf("invalid engine %q", config.Mode)
	}
	if config.MaxRegionInstructions == 0 && !config.NaturalBlocks {
		config.MaxRegionInstructions = config.MaxInstructions
	}
	if config.MaxNativeInstructions == 0 {
		config.MaxNativeInstructions = 64
	}
	if (!config.NaturalBlocks && (config.MaxRegionInstructions < 1 || config.MaxRegionInstructions > 256)) || config.MaxNativeInstructions < 1 || config.MaxNativeInstructions > emu.NativeSessionLimit {
		return nil, fmt.Errorf("invalid execution slice configuration")
	}
	if !config.Adaptive && (config.IRThreshold == 0 || config.NativeThreshold < config.IRThreshold) || !config.DynamicCode && (config.MaxBlocks < 1 || config.MaxBlocks > 65536) || !config.NaturalBlocks && (config.MaxInstructions < 1 || config.MaxInstructions > 16) || config.NativeBytes < 4096 || config.NativeBytes > 64<<20 {
		return nil, fmt.Errorf("invalid engine budget")
	}
	e := &Engine{arch: arch, config: config, blocks: map[uint64]*compiledBlock{}}
	for alignment := arch.Alignment; alignment > 1; alignment >>= 1 {
		e.dispatchShift++
	}
	if config.CacheContexts {
		e.variants = make(map[contextPC]*compiledBlock)
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
			if config.DynamicCode {
				n.AllowGrowth()
			}
			if config.ProfileNative {
				_ = n.EnablePerfMap(config.ProfileNativeMonotonic)
				e.Stats.ProfileErrors = n.SymbolErrors
			}
		}
	}
	return e, nil
}
func (e *Engine) Close() error {
	if e.variants != nil {
		for _, b := range e.variants {
			if b.proof != nil {
				b.proof.Close()
			}
			if b.regionProof != nil {
				b.regionProof.Close()
			}
		}
	} else {
		for _, b := range e.blocks {
			if b.proof != nil {
				b.proof.Close()
			}
			if b.regionProof != nil {
				b.regionProof.Close()
			}
		}
	}
	e.codeLinks = nil
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
	var context CodeStamp
	if e.config.CacheContexts {
		if memory, ok := c.MemoryBus().(CodeContextMemory); ok {
			context = memory.CodeContext()
			e.Stats.ContextChecks++
		}
		e.cacheContext = context.Identity
	}
	cache := &e.dispatch[e.dispatchSlot(pc)]
	old := cache.block
	direct := old != nil && cache.pc == pc && old.cacheContext == e.cacheContext
	if !direct {
		old = e.cachedBlock(pc)
	}
	if old == nil {
		return e.discoverBlock(c, pc, cache)
	}
	valid := false
	if old.proof != nil {
		valid = old.proof.Valid()
		if !valid {
			valid = old.proof.Revalidate()
		}
		if valid {
			if !e.config.CacheContexts {
				context = c.MemoryBus().(CodeContextMemory).CodeContext()
			}
			old.context = context
		}
	} else if len(old.dependencies) != 0 {
		if !e.config.CacheContexts {
			if memory, ok := c.MemoryBus().(CodeContextMemory); ok {
				context = memory.CodeContext()
				e.Stats.ContextChecks++
			}
		}
		valid = context.Identity != nil && context == old.context
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
	if old.regionProof != nil && !old.regionProof.Valid() && !old.regionProof.Revalidate() {
		e.invalidateRegion(old)
	}
	old.recent = 1
	if old.native {
		old.recent = 2
	}
	// A direct hit already contains exactly this pointer; do not rewrite its
	// GC-tracked slot on every executed block. Map hits still repair collisions.
	if !direct {
		*cache = dispatchEntry{pc, old}
		e.blocks[pc] = old
	}
	return old
}

// Keep decoding, allocation and the large instruction descriptor out of the
// guarded lookup's frame; none of these are needed for a hot block entry.
func (e *Engine) discoverBlock(c CPU, pc uint64, cache *dispatchEntry) *compiledBlock {
	defer pauseTranslationClock(c)()
	if e.config.Adaptive {
		key := contextPC{pc, e.cacheContext}
		if e.reused == nil {
			e.reused = make(map[contextPC]bool)
		}
		if !e.reused[key] {
			e.reused[key] = true
			return nil
		}
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
	if !e.config.DynamicCode && e.CachedBlocks() >= e.config.MaxBlocks && !e.config.EvictCold {
		return nil
	}
	owner, local := c.MemoryBus().(CodeProofMemory)
	if local {
		local = owner.CodeProofsEnabled()
	}
	mutation := uint64(0)
	if local {
		mutation = owner.CodeMutation()
	}
	block := &compiledBlock{cacheSlot: -1, recent: 1, cacheContext: e.cacheContext}
	nativeMemory, capable := c.MemoryBus().(NativeMemory)
	allowMemory := capable && e.native != nil && nativeMemory.NativeContext() != nil
	addr := pc
	for i := 0; e.config.NaturalBlocks || i < e.config.MaxInstructions; i++ {
		if i > 0 && e.config.NaturalBlocks && addr>>12 != pc>>12 {
			break
		}
		ins, err := e.arch.Fetch(c.MemoryBus(), addr)
		e.Stats.CodeReads++
		if err != nil || ins.PC != addr || ins.Length == 0 || ins.Length > 15 || uint64(ins.Length) > ^uint64(0)-addr || uint64(ins.Length)%e.arch.Alignment != 0 {
			break
		}
		if ins.ScheduleBoundary && len(block.instructions) != 0 {
			break // Clock/device scheduling reads own their host boundary.
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
		if count < len(block.memoryPrefix) {
			block.memoryPrefix[count] = block.memoryPrefix[count-1]
		}
		if ins.Memory {
			block.effects = true
			if count < len(block.memoryPrefix) {
				block.memoryPrefix[count]++
			}
		}
		if ins.Terminal || ins.ScheduleBoundary {
			break
		}
		addr += uint64(ins.Length)
	}
	if len(block.instructions) == 0 {
		return nil
	}
	// Wide bodies use the loop ABI even for one-shot code: retirement and
	// memory progress live in checked native exits, never in a 17-byte prefix.
	if len(block.instructions) > 16 {
		block.effects = true
	}
	e.dependencies(c, pc, block)
	if local {
		block.proof = owner.ProveCode(codeRanges(block.instructions), mutation)
		if block.proof == nil {
			return nil
		}
	}
	if e.config.EvictCold {
		e.admit(pc, block)
	}
	e.blocks[pc] = block
	if e.variants != nil {
		e.variants[contextPC{pc, e.cacheContext}] = block
	}
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
	// Step is also a quiescent dispatch entry. Callers owning precise device
	// boundaries need not use RunQuanta merely to keep the native arena alive.
	if err := e.recycle(c); err != nil {
		return err
	}
	if e.config.Mode == "interpreter" {
		return e.interpret(c)
	}
	return e.stepBlock(c, remaining, e.block(c))
}

// stepBlock consumes a lookup already validated in this single-threaded engine.
// This lets Run fall back without a second lookup/context query at the same PC.
func (e *Engine) stepBlock(c CPU, remaining uint64, block *compiledBlock) error {
	if block == nil {
		return e.interpret(c)
	}
	if uint64(len(block.instructions)) > remaining {
		return e.executeCached(c, block.instructions[0])
	}
	if block.interpretOnly {
		// block() has validated this instruction's execute permission and
		// code version (or bytes). Avoid fetching it a second time. Memory,
		// traps and unsupported instructions keep CPU semantics.
		return e.executeCached(c, block.instructions[0])
	}
	block.visits++
	if block.effects {
		return e.stepMemory(c, block)
	}
	if !block.native && (!e.config.Adaptive && block.visits < e.config.IRThreshold || e.config.Adaptive && block.visits < e.nativeThreshold(block)) {
		return e.executeCached(c, block.instructions[0])
	}
	if !e.preparePureBlock(c, block) {
		return e.interpret(c)
	}
	var err error
	start := time.Time{}
	if e.config.Adaptive {
		start = time.Now()
	}
	if block.native && e.native != nil {
		var context *emu.MemoryContext
		if memory, ok := c.MemoryBus().(NativeMemory); ok {
			context = memory.NativeContext()
		}
		if context != nil && e.config.Adaptive {
			e.publishLeaf(c, block, context)
			// Pure leaves do not use the context, but keep the same admitted
			// owner lease alive instead of unpinning its RAM graph on each ALU leaf.
			err = e.native.CallMemory(block.entry, c.Registers(), context)
		} else {
			err = e.native.Call(block.entry, c.Registers())
		}
	} else {
		err = e.ir.Run(block.prepared, c.Registers())
	}
	if err != nil {
		return err
	}
	block.successor, block.successorSeen = c.Registers()[e.arch.PC], true
	e.observeEdge(block.instructions[len(block.instructions)-1], c.Registers()[e.arch.PC])
	retired := uint64(len(block.instructions))
	if block.native {
		e.observeLeaf(block, start, retired)
	}
	*c.Retirement() += retired
	if block.native && e.native != nil {
		e.Stats.Native += retired
	} else {
		e.Stats.IR += retired
	}
	return nil
}

// executeCached is reached only after this instruction's code/mode/execute
// proof was validated in the current dispatch. It skips fetching, not any
// architectural memory, fault, device or retirement behavior.
func (e *Engine) executeCached(c CPU, ins Instruction) error {
	before := *c.Retirement()
	sampled := e.config.Adaptive && (e.Stats.Interpreted == 0 || e.Stats.Interpreted&(e.Stats.Interpreted-1) == 0)
	start := time.Time{}
	if sampled {
		start = time.Now()
	}
	err := c.ExecuteInstruction(ins)
	if sampled && *c.Retirement() > before {
		e.payback.interpretedNS += uint64(time.Since(start))
		e.payback.interpreted += *c.Retirement() - before
	}
	e.Stats.Interpreted += *c.Retirement() - before
	if err == nil && *c.Retirement() == before+1 {
		e.observeEdge(ins, c.Registers()[e.arch.PC])
	}
	return err
}
func (e *Engine) interpret(c CPU) error {
	before := *c.Retirement()
	sampled := e.config.Adaptive && (e.Stats.Interpreted == 0 || e.Stats.Interpreted&(e.Stats.Interpreted-1) == 0)
	start := time.Time{}
	if sampled {
		start = time.Now()
	}
	block := e.cachedBlock(c.Registers()[e.arch.PC])
	err := c.Step()
	if sampled && *c.Retirement() > before {
		e.payback.interpretedNS += uint64(time.Since(start))
		e.payback.interpreted += *c.Retirement() - before
	}
	e.Stats.Interpreted += *c.Retirement() - before
	if err == nil && *c.Retirement() == before+1 && block != nil && len(block.instructions) != 0 {
		e.observeEdge(block.instructions[0], c.Registers()[e.arch.PC])
	}
	return err
}

// NativeAvailable and CachedBlocks expose diagnostics without mutable caches.
func (e *Engine) NativeAvailable() bool { return e.native != nil }
func (e *Engine) CachedBlocks() int {
	if e.variants != nil {
		return len(e.variants)
	}
	return len(e.blocks)
}

func (e *Engine) validateCPU(c CPU) error {
	if e == nil || c == nil || len(c.Registers()) != e.arch.StateWords || c.Retirement() == nil {
		return fmt.Errorf("guest state does not match architecture")
	}
	return nil
}

func (e *Engine) preparePureBlock(c CPU, block *compiledBlock) bool {
	if block.native && e.native != nil {
		return true
	}
	if block.ops != nil && (e.native == nil || block.attempted || block.visits < e.nativeThreshold(block)) {
		return true
	}
	defer pauseTranslationClock(c)()
	start := time.Time{}
	if e.config.Adaptive {
		start = time.Now()
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
				return false
			}
		}
		block.ops = b.Finish(e.arch.StateWords)
		prepared, err := emu.Prepare(block.ops, e.arch.StateWords)
		if err != nil {
			e.Stats.CompileFailures++
			e.discard(c.Registers()[e.arch.PC])
			return false
		}
		block.prepared = prepared
	}
	if e.native != nil && !block.attempted && block.visits >= e.nativeThreshold(block) {
		block.attempted = true
		entry, err := e.native.Compile(block.ops, e.arch.StateWords)
		e.observeCompilation(start, len(block.ops), len(block.instructions))
		if err != nil {
			e.Stats.CompileFailures++
		} else {
			block.entry = entry
			_ = e.native.NameEntry(entry, fmt.Sprintf("%s_%x_pure_%d_ops", e.arch.Name, c.Registers()[e.arch.PC], len(block.ops)))
			e.Stats.ProfileErrors = e.native.SymbolErrors
			block.native = true
			e.Stats.Promotions++
			if e.config.DynamicCode {
				block.ops = nil
				block.prepared = nil
			}
		}
		e.Stats.NativeBytes = e.native.Bytes
	}
	return true
}
