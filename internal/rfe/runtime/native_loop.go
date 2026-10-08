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
	return n.compileLoop(ops, words, instructions, -1, nil)
}

// CompileLoopChained specializes up to eight direct exit PCs without enlarging
// the static region. Admission is still refreshed on every edge, and the exact
// native-call budget is never replenished by a continuation.
func (n *Native) CompileLoopChained(ops []Op, words, instructions, pcSlot int, exits []uint64) (int, error) {
	if n.linkWords != words || n.linkPC != pcSlot || pcSlot < 0 || pcSlot >= words || len(exits) < 1 || len(exits) > 8 {
		return 0, fmt.Errorf("invalid chained loop shape")
	}
	pcs := make([]int, len(exits))
	for i, pc := range exits {
		if pc&3 != 0 {
			return 0, fmt.Errorf("unaligned chained exit")
		}
		pcs[i] = int(pc)
	}
	return n.compileLoop(ops, words, instructions, pcSlot, pcs)
}

func (n *Native) compileLoop(ops []Op, words, instructions, pcSlot int, exits []int) (int, error) {
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
	records := make([]int, 0, len(ops)*4)
	for _, op := range ops {
		records = append(records, op.Kind, int(op.A), int(op.B), int(op.Imm))
	}
	transfer := len(exits) != 0 && pcSlot > 0 && runtime.GOARCH == "amd64"
	for _, op := range ops {
		if op.Kind == MemoryLoad || op.Kind == MemoryStore || op.Kind == MemoryPairStore || op.Kind == PairHigh {
			transfer = false
		}
	}
	code, ok := []byte(nil), false
	if transfer {
		code, ok = backendcompiled.RenvoEmitTransferLoopBlock(records, words, instructions, pcSlot, exits)
	} else if len(exits) == 0 {
		code, ok = backendcompiled.RenvoEmitSharedLoopBlock(records, words, instructions, runtime.GOARCH == "arm64")
	} else {
		code, ok = backendcompiled.RenvoEmitChainedLoopBlock(records, words, instructions, runtime.GOARCH == "arm64", pcSlot, exits)
	}
	if !ok {
		return 0, fmt.Errorf("Renvo could not emit native loop")
	}
	entry, err := 0, error(nil)
	if transfer {
		entry, err = n.arena.InstallTransferLoopBlock(code, words, instructions)
	} else {
		entry, err = n.arena.InstallLoopBlock(code, words, instructions)
	}
	if err == nil {
		n.Bytes += len(code)
	}
	return entry, err
}
