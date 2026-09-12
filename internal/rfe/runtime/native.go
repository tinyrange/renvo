//go:build !renvo

package runtime

import (
	"fmt"
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/runimage"
	"runtime"
)

type Native struct {
	arena *runimage.CodeArena
	Bytes int
	words map[int]int
}

func NewNative(capacity int) (*Native, error) {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		return nil, fmt.Errorf("RFE native blocks require a 64-bit host")
	}
	arena, err := runimage.NewCodeArena(capacity)
	if err != nil {
		return nil, err
	}
	return &Native{arena: arena, words: map[int]int{}}, nil
}
func (n *Native) Compile(ops []Op, words int) (int, error) {
	if err := Validate(ops, words); err != nil {
		return 0, err
	}
	records := make([]int, 0, len(ops)*4)
	for _, op := range ops {
		records = append(records, op.Kind, int(op.A), int(op.B), int(op.Imm))
	}
	code, ok := backendcompiled.RenvoEmitPureBlock(records, words, runtime.GOARCH == "arm64")
	if !ok {
		return 0, fmt.Errorf("Renvo could not emit native RFE block")
	}
	entry, err := n.arena.Install(code)
	if err == nil {
		n.Bytes += len(code)
		n.words[entry] = words
	}
	return entry, err
}
func (n *Native) Call(entry int, state []uint64) error {
	if n.words[entry] != len(state) || len(state) == 0 {
		return fmt.Errorf("native block state size mismatch")
	}
	return n.arena.Call(entry, state)
}
func (n *Native) Close() error { return n.arena.Close() }
