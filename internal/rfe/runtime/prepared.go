package runtime

import "fmt"

// PreparedBlock owns an immutable copy of validated IR. It may be shared
// between machines; it never owns or caches architectural state.
type PreparedBlock struct {
	ops   []Op
	words int
}

func Prepare(ops []Op, words int) (*PreparedBlock, error) {
	if err := Validate(ops, words); err != nil {
		return nil, err
	}
	return &PreparedBlock{ops: append([]Op(nil), ops...), words: words}, nil
}

// IRMachine reuses bounded evaluator scratch. Like an emulator Engine, one
// machine is single-threaded/non-reentrant. Different machines may execute the
// same PreparedBlock concurrently with separate states.
type IRMachine struct{ values [2048]uint64 }

func (m *IRMachine) Run(block *PreparedBlock, state []uint64) error {
	if m == nil || block == nil || block.words < 1 || len(state) != block.words {
		return fmt.Errorf("prepared IR state size mismatch")
	}
	interpretValues(block.ops, state, m.values[:len(block.ops)])
	return nil
}
