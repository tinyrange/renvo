// Package engine owns bounded tiering, code validation and trace discovery.
// Guest decoders supply instruction boundaries and semantics, not host pointers.
package engine

import emu "renvo.dev/internal/rfe/runtime"

// Memory checks the complete access before committing writes. Fetches require
// execute permission. Pair accesses and architecture-specific alignment belong
// to the guest adapter, not the tier controller.
type Memory interface {
	Read(address uint64, size int, execute bool) (uint64, error)
	Write(address uint64, size int, value uint64) error
}

// Instruction describes one architectural instruction, independently of its
// byte length. PC/Length cover every fetched byte, including a straddling page.
// Bits is decoder-owned. Class is an optional cached-interpreter classification.
type Instruction struct {
	PC                              uint64
	Bits                            uint64
	Length                          uint8
	Class                           uint8
	InterpretOnly, Memory, Terminal bool
	Flow                            Successors
}

type Successors struct {
	Next, Alternate    uint64
	Conditional, Known bool
}

// Architecture is immutable after New. Fetch must use execute-checked reads.
// Lower must leave the builder unchanged on rejection. completed is an
// instruction count, never a byte offset. Unsupported operations use the CPU.
type Architecture struct {
	Name           string
	StateWords, PC int
	Alignment      uint64
	Fetch          func(Memory, uint64) (Instruction, error)
	Lower          func(*emu.Builder, Instruction, int) bool
}

// CPU retains ownership of its register array, memory and retirement counter.
// These views must remain stable during each single-threaded engine invocation.
// ExecuteInstruction skips fetching only; it must preserve Step's semantics.
type CPU interface {
	Registers() []uint64
	MemoryBus() Memory
	Retirement() *uint64
	Step() error
	ExecuteInstruction(Instruction) error
}

// CodeIdentity distinguishes address spaces even when generation values match.
type CodeIdentity struct{ marker byte }
type CodeStamp struct {
	Identity *CodeIdentity
	Epoch    uint64
}

// VersionedMemory stamps cover 4 KiB engine code pages. Each query must check
// execute permission. Stamps change for every content, mapping or permission
// change and must never be reused, including after remapping an address.
// Implementations and the engine are single-threaded.
type VersionedMemory interface {
	CodeVersion(uint64) (CodeStamp, error)
}

// CodeContextMemory changes its identity/epoch before any executable byte,
// execute permission or executable mapping changes. Copies of an address space
// share that clock. A stable context only reuses already-checked dependencies;
// it never validates a newly discovered block. Epochs must not be reused.
type CodeContextMemory interface {
	VersionedMemory
	CodeContext() CodeStamp
}

// NativeMemory is trusted and single-threaded. Aliases share clocks, mappings
// and rooted descriptors. Refill updates metadata, never performs guest access.
type NativeMemory interface {
	CodeContextMemory
	NativeContext() *emu.MemoryContext
	NativeRefill(uint64)
}
