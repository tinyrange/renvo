//go:build renvo && browser

package runtime

import (
	"encoding/binary"
	"fmt"
	"renvo.dev/internal/arena"
)

const BrowserRecycleEntryLimit = browserNativeEntryLimit / 2

// SuspendForRecycle revokes every public execution route before the engine
// reclaims/revalidates executable watches. Host storage stays charged to this
// same arena until a single bounded transfer or ordinary Close consumes it.
func (n *Native) SuspendForRecycle() error {
	if n == nil || n.device == nil || n.suspended {
		return fmt.Errorf("invalid browser recycle owner")
	}
	var packet [8]byte
	binary.LittleEndian.PutUint32(packet[:], 4)
	binary.LittleEndian.PutUint32(packet[4:], uint32(n.deviceOwner))
	if _, err := n.exchange(packet[:]); err != nil {
		return err
	}
	n.EndFastRefill()
	for context := range n.ownedContexts {
		if context.linkOwner == n {
			context.ClearLinks()
			context.linkOwner = nil
		}
	}
	clear(n.ownedContexts)
	clear(n.codeVersionedContexts)
	n.suspended, n.gateOwner = true, 0
	return nil
}

func (n *Native) transferMetadata(entry int) *browserEntry {
	slot := browserEntrySlot(entry)
	if n == nil || !n.suspended || slot == 0 || slot >= len(n.entries) || entry == n.dispatchEntry {
		return nil
	}
	m := &n.entries[slot]
	if m.words == 0 || m.bytes <= 0 {
		return nil
	}
	return m
}
func (n *Native) TransferEntryBytes(entry int) int {
	if m := n.transferMetadata(entry); m != nil {
		size := m.bytes
		if m.proof.instructions != 0 {
			size += len(n.proofRequest) - 24
		}
		return size
	}
	return 0
}
func (n *Native) TransferByteBudget() int { return n.capacity/2 - n.gateBytes }

// RecycleKeeping changes owner identity and compacts already-compiled ordinary
// entries. Dispatchers (which capture the mutable table) are never shared.
// An exact immutable admission tuple may survive only with the SAME
// entry/PC/dimensions and revalidated code. The new owner has a fresh prover;
// exported old-owner provers remain revoked. Every code/proof byte consumes the original cap; no
// executable watch or memory-context admission survives ownership revocation.
func (n *Native) RecycleKeeping(entries []int) (*Native, error) {
	if n == nil || n.device == nil || !n.suspended || len(entries) > BrowserRecycleEntryLimit {
		return nil, fmt.Errorf("invalid browser recycle selection")
	}
	next := &Native{layout: n.layout, device: n.device, capacity: n.capacity, entries: n.entries,
		ownedContexts: n.ownedContexts, codeVersionedContexts: n.codeVersionedContexts,
		refillStack: n.refillStack, proofRequest: n.proofRequest, gateBytes: n.gateBytes}
	low, high := arena.Mark(), arena.PersistMark()
	infos := make([]browserEntry, len(entries))
	var seen [512]uint32
	size := 0
	for i, entry := range entries {
		m := n.transferMetadata(entry)
		if m == nil {
			return nil, fmt.Errorf("nontransferable browser entry")
		}
		slot := entry >> 4
		mask := uint32(1) << uint(slot&31)
		if seen[slot>>5]&mask != 0 {
			return nil, fmt.Errorf("duplicate browser transfer entry")
		}
		seen[slot>>5] |= mask
		charge := n.TransferEntryBytes(entry)
		if charge > n.TransferByteBudget()-size {
			return nil, fmt.Errorf("browser retained byte budget exceeded")
		}
		size += charge
		infos[i] = *m
	}
	packet := make([]byte, 12+4*len(entries))
	binary.LittleEndian.PutUint32(packet, 5)
	binary.LittleEndian.PutUint32(packet[4:], uint32(n.deviceOwner))
	binary.LittleEndian.PutUint32(packet[8:], uint32(len(entries)))
	for i, entry := range entries {
		binary.LittleEndian.PutUint32(packet[12+4*i:], uint32(entry))
	}
	owner, err := n.exchange(packet)
	if err != nil {
		return nil, err
	}
	if owner <= 0 || owner == n.deviceOwner {
		return nil, fmt.Errorf("browser recycle did not change owner")
	}
	clear(next.entries)
	for i, info := range infos {
		next.entries[entries[i]>>4] = info
		if info.loopInstructions != 0 && n.codeVersionedLoops {
			next.codeVersionedLoops = true
		}
	}
	next.gateOwner, next.deviceOwner = owner, owner
	next.entryCount = len(entries)
	next.Bytes = size + n.gateBytes
	n.device = nil
	n.entries = nil
	n.ownedContexts = nil
	n.codeVersionedContexts = nil
	n.refillStack = nil
	n.proofRequest = nil
	n.deviceOwner = 0
	arena.Rewind(low)
	arena.PersistReset(high)
	return next, nil
}
