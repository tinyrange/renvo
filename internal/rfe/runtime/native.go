//go:build !renvo

package runtime

import (
	"fmt"
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/runimage"
	"runtime"
	"sync"
)

type Native struct {
	arena                   *runimage.CodeArena
	Bytes                   int
	SymbolErrors            uint64
	linkEntry               int
	linkWords               int
	linkPC                  int
	linkCall                *runimage.LinkedCall
	profileMu               sync.Mutex
	profileCapture          func() error // opt-in cold snapshot, never used by native calls
	profileCaptureAttempted bool
}

func NewNative(capacity int) (*Native, error) {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		return nil, fmt.Errorf("RFE native blocks require a 64-bit host")
	}
	arena, err := runimage.NewCodeArena(capacity)
	if err != nil {
		return nil, err
	}
	return &Native{arena: arena}, nil
}
func (n *Native) Compile(ops []Op, words int) (int, error) {
	if err := Validate(ops, words); err != nil {
		return 0, err
	}
	records := make([]int, 0, len(ops)*4)
	for _, op := range ops {
		records = append(records, op.Kind, int(op.A), int(op.B), int(op.Imm))
	}
	code, body, ok := backendcompiled.RenvoEmitSharedBlock(records, words, false, runtime.GOARCH == "arm64")
	if !ok {
		return 0, fmt.Errorf("Renvo could not emit native RFE block")
	}
	entry, err := n.arena.InstallSharedBlock(code, words, false, body)
	if err == nil {
		n.Bytes += len(code)
	}
	return entry, err
}
func (n *Native) Call(entry int, state []uint64) error {
	return n.arena.Call(entry, state)
}

// CallBatch retains arena serialization while a bounded trusted dispatcher
// selects hot blocks. The selector must not compile, call or close this Native.
func (n *Native) CallBatch(state []uint64, limit int, next func(int) (int, bool, error)) (int, error) {
	return n.arena.CallBatch(state, limit, next)
}
func (n *Native) Close() error {
	n.profileMu.Lock()
	defer n.profileMu.Unlock()
	if n.profileCapture != nil {
		capture := n.profileCapture
		n.profileCapture = nil
		if err := capture(); err != nil {
			n.SymbolErrors++
		}
	}
	return n.arena.Close()
}
