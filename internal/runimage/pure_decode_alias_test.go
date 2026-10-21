//go:build !renvo

package runimage

import (
	"renvo.dev/internal/rfeabi"
	"testing"
	"unsafe"
)

func TestDecodeVersionMetadataMustBeDisjointFromGuestAndExecutionState(t *testing.T) {
	ram := make([]byte, 4096)
	clock := uint64(1)
	m := &LinkedContextABI{Clock: &clock}
	m.Memory = rfeabi.NewOwnedMemory(ram, &clock, rfeabi.NewWriteWatches(len(ram)))
	state := make([]uint64, 2)
	m.Memory.SetDecodeEpochs(make([]uint64, 1))
	if sessionDecodeVersionsAlias(state, m) {
		t.Fatal("disjoint versions rejected")
	}
	for _, epochs := range [][]uint64{state[:1], unsafe.Slice(m.Clock, 1), unsafe.Slice((*uint64)(unsafe.Pointer(&ram[128])), 1), unsafe.Slice(&m.Total, 1)} {
		m.Memory.SetDecodeEpochs(epochs)
		if !sessionDecodeVersionsAlias(state, m) {
			t.Fatal("aliased decode metadata accepted")
		}
	}
}
