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
	code, ok := backendcompiled.RenvoEmitSharedLoopBlock(records, words, instructions, runtime.GOARCH == "arm64")
	if !ok {
		return 0, fmt.Errorf("Renvo could not emit native loop")
	}
	entry, err := n.arena.InstallLoopBlock(code, words, instructions)
	if err == nil {
		n.Bytes += len(code)
	}
	return entry, err
}
