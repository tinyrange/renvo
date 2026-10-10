//go:build !renvo

package runtime

import (
	"fmt"
	"renvo.dev/internal/backendcompiled"
	"runtime"
)

// CompileLoop compiles one immutable iteration and its native loop-carried
// state/exit maps. The static body is bounded by 256 guest instructions; native
// execution cannot exceed the independently bounded caller's session budget.
// Loop entries can only be entered by that checked dispatcher, not CallMemory.
func (n *Native) CompileLoop(ops []Op, words, instructions int) (int, error) {
	return n.CompileLoopMode(ops, words, instructions, false)
}

// CompileLoopMode retains a checked entry body even when direct memory is
// requested. Only an admitted foreign session with a direct window selects it.
func (n *Native) CompileLoopMode(ops []Op, words, instructions int, direct bool) (int, error) {
	return n.compileLoopMode(ops, words, instructions, direct, nil)
}

// CompileCodeVersionedLoop admits an owned address space whose generations are
// code stamps, not observable data-write counters. The owner must serialize
// mapping/execution, keep guest RAM disjoint from host metadata, invalidate
// descriptors on mapping changes, and assign a fresh generation before making
// data executable. Executable stores still exit to the architecture.
// After installation, linked calls accept only contexts explicitly registered
// through this method; generic contexts retain their original store semantics.
func (n *Native) CompileCodeVersionedLoop(ops []Op, words, instructions int, memory *MemoryContext) (int, error) {
	if memory == nil {
		return 0, fmt.Errorf("missing code-versioned memory owner")
	}
	return n.compileLoopMode(ops, words, instructions, false, memory)
}

func (n *Native) compileLoopMode(ops []Op, words, instructions int, direct bool, codeVersions *MemoryContext) (int, error) {
	if !linkLayoutOK() || instructions < 1 || instructions > 256 {
		return 0, fmt.Errorf("invalid native loop dimensions")
	}
	if err := ValidateMemory(ops, words); err != nil {
		return 0, err
	}
	seen := make([]bool, words)
	progress, accesses := 0, 0
	for i, op := range ops {
		switch op.Kind {
		case LoadState:
			if seen[op.Imm] {
				return 0, fmt.Errorf("duplicate loop input")
			}
			seen[op.Imm] = true
		case Progress:
			if op.Imm < uint64(progress) || op.Imm > uint64(instructions) {
				return 0, fmt.Errorf("invalid loop checkpoint")
			}
			progress = int(op.Imm)
		case MemoryLoad, MemoryStore, MemoryPairStore:
			accesses++
			if accesses > instructions || progress >= instructions {
				return 0, fmt.Errorf("invalid loop access progress")
			}
		case RegionGuard, Guard:
			if progress >= instructions || op.Kind == RegionGuard && progress == 0 {
				return 0, fmt.Errorf("invalid loop guard progress")
			}
		case LoopContinue:
			if i != len(ops)-1 || progress != instructions {
				return 0, fmt.Errorf("invalid loop terminator")
			}
		}
	}
	if ops[len(ops)-1].Kind != LoopContinue {
		return 0, fmt.Errorf("missing loop terminator")
	}
	records := nativeRecords(ops)
	var code []byte
	var faults []int
	var ok bool
	if codeVersions != nil {
		code, ok = backendcompiled.RenvoEmitCodeVersionedLoopBlock(records, words, instructions, runtime.GOARCH == "arm64")
	} else if direct && runtime.GOOS == "linux" && runtime.GOARCH == "amd64" && NativeSessionsAvailable() {
		code, faults, ok = backendcompiled.RenvoEmitDirectLoopBlock(records, words, instructions)
	} else {
		code, ok = backendcompiled.RenvoEmitSharedLoopBlock(records, words, instructions, runtime.GOARCH == "arm64")
	}
	if !ok {
		return 0, fmt.Errorf("Renvo could not emit native loop")
	}
	var entry int
	var err error
	if len(faults) != 0 {
		entry, err = n.arena.InstallFaultLoopBlock(code, words, instructions, faults)
	} else {
		entry, err = n.arena.InstallLoopBlock(code, words, instructions)
	}
	if err == nil {
		if codeVersions != nil {
			if n.codeVersionedContexts == nil {
				n.codeVersionedContexts = make(map[*MemoryContext]bool)
			}
			n.codeVersionedContexts[codeVersions] = true
		}
		n.Bytes += len(code)
	}
	return entry, err
}
