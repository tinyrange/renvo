//go:build !renvo

package engine

import (
	"fmt"
	emu "renvo.dev/internal/rfe/runtime"
	"testing"
)

type prefillMemory struct {
	stubMemory
	context    emu.MemoryContext
	stamp      CodeStamp
	versions   map[uint64]CodeStamp
	reads      int
	transition bool
	swapOnRead bool
	alternate  *emu.MemoryContext
}

func (m *prefillMemory) NativeContext() *emu.MemoryContext {
	if m.alternate != nil {
		return m.alternate
	}
	return &m.context
}
func (m *prefillMemory) NativeRefill(uint64)    {}
func (m *prefillMemory) CodeContext() CodeStamp { return m.stamp }
func (m *prefillMemory) CodeVersion(page uint64) (CodeStamp, error) {
	m.reads++
	if m.swapOnRead {
		m.alternate = new(emu.MemoryContext)
	}
	if m.transition {
		m.stamp.Epoch++
	}
	s, ok := m.versions[page]
	if !ok {
		return CodeStamp{}, fmt.Errorf("execute permission revoked")
	}
	return s, nil
}
func prefillFixture(t *testing.T) (*Engine, *stubCPU, *prefillMemory) {
	t.Helper()
	return prefillSpacingFixture(t, 4096)
}
func prefillSpacingFixture(t *testing.T, spacing int) (*Engine, *stubCPU, *prefillMemory) {
	t.Helper()
	n, err := emu.NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { n.Close() })
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	dependency := CodeStamp{Identity: new(CodeIdentity), Epoch: 1}
	m := &prefillMemory{stamp: CodeStamp{Identity: new(CodeIdentity), Epoch: 2}, versions: map[uint64]CodeStamp{0: dependency, 4096: dependency, 8192: dependency}}
	m.context.ClaimLinks(n)
	e := &Engine{arch: stubArchitecture(), native: n, blocks: map[uint64]*compiledBlock{}}
	for i := 0; i < 3; i++ {
		pc, next := uint64(i*spacing), uint64((i+1)*spacing)
		var b emu.Builder
		b.Store(0, b.Binary(emu.Add, b.Load(0), b.Constant(uint64(i+1))))
		b.Store(1, b.Constant(next))
		entry, err := n.Compile(b.Finish(2), 2)
		if err != nil {
			t.Fatal(err)
		}
		block := &compiledBlock{entry: entry, native: true, instructions: []Instruction{{PC: pc, Length: 2, Flow: Successors{Known: true, Next: next}}}, dependencies: []codeDependency{{pc &^ 4095, dependency}}}
		e.blocks[pc] = block
	}
	first := e.blocks[0]
	if err = n.PrepareTargetLink(0, first.entry, 2, 1); err != nil {
		t.Fatal(err)
	}
	m.context.PublishLink(0, first.entry, 1, [17]uint8{})
	return e, &stubCPU{memory: m}, m
}
func TestPrefillChecksEveryDependencyWithoutExecutingGuest(t *testing.T) {
	e, c, m := prefillFixture(t)
	before := c.state
	if !e.prefillLinks(c, e.blocks[0], m, &m.context, m.stamp) {
		t.Fatal("stable mapping rejected")
	}
	if c.state != before || c.retired != 0 || m.reads != 2 || m.context.LookupLink(4096) == nil || m.context.LookupLink(8192) == nil {
		t.Fatal("prefill effects or missing successors", c.state, c.retired, m.reads)
	}
	state := []uint64{5, 0}
	if err := e.native.CallLinked(state, &m.context, 3); err != nil {
		t.Fatal(err)
	}
	if state[0] != 11 || state[1] != 12288 || m.context.Total != 3 {
		t.Fatal("compiled chain", state, m.context.Total)
	}
}
func TestPrefillRejectsRevokedCodeAndSchedulingBoundaries(t *testing.T) {
	for _, mode := range []string{"permission", "version", "boundary", "cold", "unknown"} {
		t.Run(mode, func(t *testing.T) {
			e, c, m := prefillFixture(t)
			switch mode {
			case "permission":
				delete(m.versions, 4096)
			case "version":
				m.versions[4096] = CodeStamp{Identity: new(CodeIdentity), Epoch: 1}
			case "boundary":
				e.blocks[4096].instructions[0].ScheduleBoundary = true
			case "cold":
				e.blocks[4096].native = false
			case "unknown":
				e.blocks[0].instructions[0].Flow.Known = false
			}
			if !e.prefillLinks(c, e.blocks[0], m, &m.context, m.stamp) || m.context.LookupLink(4096) != nil || m.context.LookupLink(8192) != nil || c.retired != 0 {
				t.Fatal("unvalidated successor admitted")
			}
		})
	}
}
func TestPrefillRevokesGroupIfResolverChangesContext(t *testing.T) {
	for _, failed := range []bool{false, true} {
		e, c, m := prefillFixture(t)
		m.transition = true
		if failed {
			delete(m.versions, 4096)
		}
		if e.prefillLinks(c, e.blocks[0], m, &m.context, m.stamp) || m.context.LookupLink(0) != nil || m.context.LookupLink(4096) != nil || c.retired != 0 {
			t.Fatal("context transition retained stale authority", failed)
		}
	}
}

func TestPrefillSharesOnlyCurrentPageProofs(t *testing.T) {
	for _, mismatch := range []bool{false, true} {
		e, c, m := prefillSpacingFixture(t, 16)
		first := e.blocks[0]
		first.context = m.stamp // the normal caller validated the root first
		if mismatch {
			e.blocks[16].dependencies[0].stamp.Epoch++
		}
		if !e.prefillLinks(c, first, m, &m.context, m.stamp) || m.reads != 0 {
			t.Fatal("same-page proof was not shared", mismatch, m.reads)
		}
		if mismatch {
			if m.context.LookupLink(16) != nil || m.context.LookupLink(32) != nil {
				t.Fatal("mismatching dependency reused another block's proof")
			}
			continue
		}
		state := []uint64{5, 0}
		if err := e.native.CallLinked(state, &m.context, 3); err != nil || state[0] != 11 || state[1] != 48 || m.context.Total != 3 {
			t.Fatal("shared-proof compiled chain", state, m.context.Total, err)
		}
		// No proof survives another prefill call under a new context. A
		// removed executable page must invalidate both warm successors.
		m.context.ClearLinks()
		m.stamp.Epoch++
		delete(m.versions, 0)
		if !e.prefillLinks(c, first, m, &m.context, m.stamp) || m.reads == 0 || m.context.LookupLink(16) != nil || m.context.LookupLink(32) != nil {
			t.Fatal("page proof survived its original context")
		}
	}
}

func TestObservedIndirectLinksRepopulateCompiledChain(t *testing.T) {
	e, c, m := prefillFixture(t)
	for _, b := range e.blocks {
		b.instructions[0].Flow = Successors{} // no statically known call/return target
	}
	e.observeLink(0, 4096)
	e.observeLink(4096, 8192)
	for round := 0; round < 2; round++ {
		m.context.ClearLinks()
		m.stamp.Epoch++
		m.context.PublishLink(0, e.blocks[0].entry, 1, [17]uint8{})
		before := c.state
		if !e.prefillLinks(c, e.blocks[0], m, &m.context, m.stamp) || c.state != before || c.retired != 0 {
			t.Fatal("observed-link prefill executed guest or rejected stable mapping")
		}
		state := []uint64{5, 0}
		if err := e.native.CallLinked(state, &m.context, 32); err != nil || state[0] != 11 || state[1] != 12288 || m.context.Total != 3 || m.context.Remaining != 29 {
			t.Fatal("indirect compiled chain did not stay linked", round, state, m.context.Total, err)
		}
	}
}

func TestObservedLinksRemainHintsAcrossRevocation(t *testing.T) {
	for _, mode := range []string{"permission", "version", "boundary", "cold", "context"} {
		t.Run(mode, func(t *testing.T) {
			e, c, m := prefillFixture(t)
			e.blocks[0].instructions[0].Flow = Successors{}
			e.observeLink(0, 4096)
			switch mode {
			case "permission":
				delete(m.versions, 4096)
			case "version":
				m.versions[4096] = CodeStamp{Identity: new(CodeIdentity)}
			case "boundary":
				e.blocks[4096].instructions[0].ScheduleBoundary = true
			case "cold":
				e.blocks[4096].native = false
			case "context":
				m.transition = true
			}
			stable := e.prefillLinks(c, e.blocks[0], m, &m.context, m.stamp)
			if stable == (mode == "context") || m.context.LookupLink(4096) != nil || c.retired != 0 {
				t.Fatal("observed hint bypassed current admission", mode, stable)
			}
		})
	}
}

func TestObservedLinksUseFullPCsAndBoundedReplacement(t *testing.T) {
	e, _, _ := prefillFixture(t)
	b := e.blocks[0]
	b.instructions[0].Flow = Successors{}
	for _, pc := range []uint64{0, 1 << 32, 1 << 48, 1 << 63} {
		e.observeLink(0, pc)
	}
	b.prefillDirty = false
	e.observeLink(0, 1<<32)
	if b.prefillDirty || b.linkSuccessorCount != 4 {
		t.Fatal("duplicate or truncated PC hint")
	}
	e.observeLink(0, ^uint64(3))
	if !b.prefillDirty || b.linkSuccessorCount != 4 || b.linkSuccessors[0] != ^uint64(3) || b.linkSuccessors[1] != 1<<32 {
		t.Fatal("replacement exceeded bounded full-PC history", b.linkSuccessors)
	}
	e.observeLink(12345, 0) // a dispatcher root evicted from metadata is not reconstructed
	if len(e.blocks) != 3 {
		t.Fatal("hint caused speculative discovery")
	}
}

func TestObservedLinksRejectNativeContextSwapDuringResolution(t *testing.T) {
	for _, failed := range []bool{false, true} {
		e, c, m := prefillFixture(t)
		e.blocks[0].instructions[0].Flow = Successors{}
		e.observeLink(0, 4096)
		m.swapOnRead = true
		if failed {
			delete(m.versions, 4096)
		}
		if e.prefillLinks(c, e.blocks[0], m, &m.context, m.stamp) || m.context.LookupLink(0) != nil || m.context.LookupLink(4096) != nil || c.retired != 0 {
			t.Fatal("native context swap retained publication", failed)
		}
	}
}

func TestPrefillChecksInitialContextEvenForCurrentBlocks(t *testing.T) {
	e, c, m := prefillFixture(t)
	old := m.stamp
	for _, b := range e.blocks {
		b.context = old
	}
	m.stamp.Epoch++
	if e.prefillLinks(c, e.blocks[0], m, &m.context, old) || m.context.LookupLink(0) != nil || m.context.LookupLink(4096) != nil || m.reads != 0 {
		t.Fatal("already-current block bypassed initial context check")
	}
}

func TestPrefillMetadataCacheCannotOutliveDiscard(t *testing.T) {
	e, c, m := prefillFixture(t)
	if !e.prefillLinks(c, e.blocks[0], m, &m.context, m.stamp) {
		t.Fatal("initial chain rejected")
	}
	old := e.blocks[4096]
	e.discard(4096)
	m.context.ClearLinks()
	// Simulate retaining a detached metadata reference: it is no longer owned
	// by the engine, even if its old executable entry remains in the arena.
	old.instructions = []Instruction{{PC: 4096, Length: 2}}
	old.dependencies = e.blocks[8192].dependencies
	old.context = m.stamp
	if got := e.prefillBlock(4096); got != nil {
		t.Fatal("discarded metadata survived in successor cache")
	}
	if !e.prefillLinks(c, e.blocks[0], m, &m.context, m.stamp) || m.context.LookupLink(4096) != nil {
		t.Fatal("detached entry was republished")
	}
	// A colliding full-PC tag must not be confused with the original PC or
	// cleared when a different colliding PC is discarded.
	high := uint64(1)<<48 | 8192
	e.blocks[high] = &compiledBlock{native: true, instructions: []Instruction{{PC: high, Length: 2}}}
	if e.prefillBlock(high) != e.blocks[high] {
		t.Fatal("high PC missing")
	}
	e.discard(8192)
	if e.prefillBlock(high) != e.blocks[high] || e.prefillBlock(8192) != nil {
		t.Fatal("colliding discard changed a different full-PC entry")
	}
}

type snapshotPrefillMemory struct {
	*prefillMemory
	queries int
}

func (m *snapshotPrefillMemory) NativeCodeContext() (CodeStamp, *emu.MemoryContext) {
	m.queries++
	return m.CodeContext(), m.NativeContext()
}
func TestPrefillSnapshotStillRechecksAfterEveryResolver(t *testing.T) {
	for _, mode := range []string{"stable", "stamp", "context", "failed-stamp", "failed-context"} {
		t.Run(mode, func(t *testing.T) {
			e, c, m := prefillFixture(t)
			memory := &snapshotPrefillMemory{prefillMemory: m}
			c.memory = memory
			if mode == "stamp" || mode == "failed-stamp" {
				m.transition = true
			}
			if mode == "context" || mode == "failed-context" {
				m.swapOnRead = true
			}
			if mode == "failed-stamp" || mode == "failed-context" {
				delete(m.versions, 4096)
			}
			stable := e.prefillLinks(c, e.blocks[0], memory, &m.context, m.stamp)
			if stable != (mode == "stable") || memory.queries < 2 {
				t.Fatal("snapshot not refreshed after resolver", mode, stable, memory.queries)
			}
			if !stable && (m.context.LookupLink(0) != nil || m.context.LookupLink(4096) != nil || c.retired != 0) {
				t.Fatal("snapshot retained stale authority", mode)
			}
		})
	}
}
