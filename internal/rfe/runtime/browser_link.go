//go:build renvo && browser

package runtime

import (
	"encoding/binary"
	"fmt"
	"renvo.dev/internal/arena"
)

func (n *Native) PrepareLinks(words, pc int) error {
	if n.dispatchEntry != 0 {
		if n.dispatchWords != words || n.dispatchPC != pc {
			return fmt.Errorf("browser dispatcher dimensions changed")
		}
		return nil
	}
	code, err := EmitWasmDispatcher(words, pc, wasmMemoryLayout())
	if err != nil {
		return err
	}
	entry, err := n.installWasm(code, words, true)
	if err != nil {
		return err
	}
	n.dispatchEntry, n.dispatchWords, n.dispatchPC = entry, words, pc
	return nil
}
func (n *Native) AdmitLink(entry, words, instructions int) error {
	metadata := n.entryMetadata(entry)
	if metadata == nil || metadata.words != words || entry == n.dispatchEntry || instructions < 1 || instructions > 256 {
		return fmt.Errorf("invalid browser link admission")
	}
	if count := metadata.loopInstructions; count != 0 && count != instructions {
		return fmt.Errorf("browser region dimensions changed")
	}
	return nil
}
func (n *Native) PrepareTargetLink(pc uint64, entry, words, instructions int) error {
	// Validate once against private immutable metadata. The public AdmitLink
	// API retains the same checks, but need not cause a second entry lookup.
	metadata := n.entryMetadata(entry)
	if metadata == nil || metadata.words != words || entry == n.dispatchEntry || instructions < 1 || instructions > 256 {
		return fmt.Errorf("invalid browser link admission")
	}
	if count := metadata.loopInstructions; count != 0 && count != instructions {
		return fmt.Errorf("browser region dimensions changed")
	}
	proof := browserProof{pc, instructions}
	if metadata.proof == proof {
		return nil
	}
	size := len(n.proofRequest) - 24
	if size > n.capacity-n.Bytes {
		return fmt.Errorf("browser proof arena full")
	}
	low, high := arena.Mark(), arena.PersistMark()
	packet := n.proofRequest
	binary.LittleEndian.PutUint32(packet[4:], uint32(entry))
	binary.LittleEndian.PutUint64(packet[8:], pc)
	binary.LittleEndian.PutUint32(packet[16:], uint32(words))
	binary.LittleEndian.PutUint32(packet[20:], uint32(instructions))
	// exchange consumes this fixed packet synchronously. All proof bytes are
	// still charged per installation, preserving the original arena bound.
	if _, err := n.exchange(packet); err != nil {
		return err
	}
	arena.Rewind(low)
	arena.PersistReset(high)
	n.Bytes += size
	metadata.proof = proof
	return nil
}
func (n *Native) CanLinkMemory(m *MemoryContext) bool {
	return m != nil && n.dispatchEntry != 0 && (!n.codeVersionedLoops || n.codeVersionedContexts[m])
}

// This is explicit host-owner authority, separate from ownedContexts, which
// only tracks generic link lifetimes. Once an optimized loop is installed,
// every linked call must carry the same disjoint-RAM/code-version contract.
func (n *Native) AdmitCodeVersionedContext(m *MemoryContext) bool {
	if m == nil {
		return false
	}
	if !n.codeVersionedContexts[m] && len(n.codeVersionedContexts) >= 64 {
		return false
	}
	n.codeVersionedContexts[m] = true
	n.ownedContexts[m] = true
	return true
}
func (n *Native) CallLinked(state []uint64, m *MemoryContext, budget uint64) error {
	if !n.CanLinkMemory(m) || len(state) != n.dispatchWords || budget < 1 || budget > NativeSessionLimit {
		return fmt.Errorf("invalid browser linked call")
	}
	n.ownedContexts[m] = true
	m.Remaining, m.Total, m.MemoryTotal = budget, 0, 0
	m.Status, m.Retired, m.LoopExits, m.LoopIterations = 0, 0, 0, 0
	m.wasmLinked = 1
	err := n.callWasm(n.dispatchEntry, state, m)
	m.wasmLinked = 0
	if err != nil {
		return err
	}
	if m.Status == 3 || m.Total > budget || m.Remaining != budget-m.Total || m.MemoryTotal > m.Total {
		return fmt.Errorf("invalid browser linked progress")
	}
	return nil
}
func (n *Native) RunLinkedSession(state []uint64, m *MemoryContext, budget uint64) error {
	return n.CallLinked(state, m, budget)
}
func (n *Native) RunLinkedBackingSession(state []uint64, m *MemoryContext, budget uint64, backing []byte) error {
	return n.CallLinked(state, m, budget)
}
func (n *Native) RunLinkedQuanta(state []uint64, m *MemoryContext, limit int, remaining, generation uint64) (int, error) {
	if limit < 1 || limit > 16 || remaining == 0 {
		return 0, fmt.Errorf("invalid browser linked quantum")
	}
	// A single bounded compiled session includes all transitions. No Go/JS
	// guest-instruction loop stands in for the WASM dispatcher.
	budget := remaining
	if budget > uint64(limit*64) {
		budget = uint64(limit * 64)
	}
	return 1, n.CallLinked(state, m, budget)
}
func (n *Native) CompileLoop(ops []Op, words, instructions int) (int, error) {
	if instructions < 1 || instructions > 256 {
		return 0, fmt.Errorf("invalid browser loop dimensions")
	}
	entry, err := n.compileWasmBody(ops, words, &n.layout, instructions)
	if err == nil {
		n.entryMetadata(entry).loopInstructions = instructions
	}
	return entry, err
}
func (n *Native) CompileLoopMode(ops []Op, words, instructions int, direct bool) (int, error) {
	return n.CompileLoop(ops, words, instructions)
}
func (n *Native) CompileCodeVersionedLoop(ops []Op, words, instructions int, m *MemoryContext) (int, error) {
	if instructions < 1 || instructions > 256 {
		return 0, fmt.Errorf("invalid browser loop dimensions")
	}
	if !n.AdmitCodeVersionedContext(m) {
		return 0, fmt.Errorf("missing browser memory owner")
	}
	entry, err := n.compileWasmMode(ops, words, &n.layout, instructions, true)
	if err == nil {
		n.entryMetadata(entry).loopInstructions = instructions
		n.codeVersionedLoops = true
	}
	return entry, err
}
