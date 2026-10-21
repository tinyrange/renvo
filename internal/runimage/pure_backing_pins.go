//go:build !renvo

package runimage

import (
	"renvo.dev/internal/rfeabi"
	"runtime"
)

// Physical RAM and its byte watches are shared by privilege/root views. Pin
// that graph once, independently of a particular context's descriptor/table
// lease; a CR3/CPL switch must not unpin and repin thousands of unchanged masks.
type ownedBackingPins struct {
	pinner  runtime.Pinner
	backing *byte
	watches *rfeabi.WriteWatches
	count   int
	epochs  *uint64
}

func (a *CodeArena) releaseBackingPinsLocked() {
	if a.backingPins != nil {
		a.backingPins.pinner.Unpin()
		a.backingPins = nil
	}
}
func (a *CodeArena) pinBackingLocked(memory *rfeabi.OwnedMemory) {
	p := a.backingPins
	if p == nil || p.backing != memory.Backing || p.watches != memory.Watches || p.epochs != memory.DecodeEpochs {
		a.releaseBackingPinsLocked()
		p = &ownedBackingPins{backing: memory.Backing, watches: memory.Watches}
		p.epochs = memory.DecodeEpochs
		if p.epochs != nil {
			p.pinner.Pin(p.epochs)
		}
		p.pinner.Pin(memory.Backing)
		p.pinner.Pin(memory.Watches)
		p.pinner.Pin(memory.Watches.Base)
		a.backingPins = p
	}
	masks := memory.Watches.AllocatedMasks()
	for _, mask := range masks[p.count:] {
		p.pinner.Pin(mask)
	}
	p.count = len(masks)
}
