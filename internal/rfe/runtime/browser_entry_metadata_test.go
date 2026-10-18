//go:build renvo && browser

package runtime

import (
	"strings"
	"testing"
)

func TestBrowserEntryMetadataRejectsForgedAndUninstalledEntries(t *testing.T) {
	n := &Native{capacity: 1 << 20}
	if err := n.recordWasm(16, 2, false, 100); err != nil {
		t.Fatal(err)
	}
	if err := n.recordWasm(48, 2, true, 200); err != nil {
		t.Fatal(err)
	}
	state := []uint64{11, 22}
	context := &MemoryContext{}
	for _, entry := range []int{-16, -1, 0, 1, 15, 17, 32, 64, (browserNativeEntryLimit + 1) * 16, int(^uint(0) >> 1)} {
		if n.entryMetadata(entry) != nil {
			t.Fatalf("entry %d acquired metadata", entry)
		}
		if n.Call(entry, state) == nil || n.CallMemory(entry, state, context) == nil || n.AdmitLink(entry, 2, 1) == nil || n.PrepareTargetLink(0, entry, 2, 1) == nil {
			t.Fatalf("forged entry %d admitted", entry)
		}
	}
	if state[0] != 11 || state[1] != 22 {
		t.Fatal("rejected entry changed state")
	}
	if err := n.Call(16, state); err == nil || !strings.Contains(err.Error(), "closed browser arena") {
		t.Fatal("valid pure metadata rejected before device", err)
	}
	if err := n.CallMemory(48, state, context); err == nil || !strings.Contains(err.Error(), "closed browser arena") {
		t.Fatal("valid memory metadata rejected before device", err)
	}
	if n.CallMemory(16, state, context) == nil || n.Call(48, state) == nil || n.Call(16, state[:1]) == nil {
		t.Fatal("entry dimensions/context checks lost")
	}
	n.entries[3].loopInstructions = 2
	if n.CallMemory(48, state, context) == nil || n.AdmitLink(48, 2, 1) == nil || n.AdmitLink(48, 2, 2) != nil {
		t.Fatal("loop instruction proof lost")
	}
	n.entries[3].loopInstructions = 0
	n.dispatchEntry = 48
	if n.CallMemory(48, state, context) == nil || n.AdmitLink(48, 2, 1) == nil {
		t.Fatal("public memory call admitted dispatcher")
	}
}

func TestBrowserEntryMetadataGrowthBoundsAndCloseRevocation(t *testing.T) {
	n := &Native{capacity: 1 << 20, ownedContexts: map[*MemoryContext]bool{}, codeVersionedContexts: map[*MemoryContext]bool{}}
	if err := n.recordWasm(16, 2, false, 10); err != nil {
		t.Fatal(err)
	}
	proof := browserProof{pc: 0xffff800000001000, instructions: 2}
	n.entries[1].proof = proof
	n.entries[1].loopInstructions = 2
	if err := n.recordWasm(browserNativeEntryLimit*16, 3, true, 20); err != nil {
		t.Fatal(err)
	}
	if len(n.entries) != browserNativeEntryLimit+1 || n.entryCount != 2 || n.Bytes != 30 || n.entries[1].proof != proof || n.entries[1].loopInstructions != 2 {
		t.Fatal("growth changed installed identity/accounting")
	}
	for _, entry := range []int{-16, 0, 17, 16, (browserNativeEntryLimit + 1) * 16} {
		if n.recordWasm(entry, 2, false, 100) == nil {
			t.Fatalf("invalid installation %d accepted", entry)
		}
	}
	if n.entryCount != 2 || n.Bytes != 30 || len(n.entries) != browserNativeEntryLimit+1 {
		t.Fatal("rejected installation changed bounded metadata")
	}
	owner := &MemoryContext{}
	owner.ClaimLinks(n)
	owner.PublishLink(0, 16, 2, [17]uint8{})
	n.ownedContexts[owner] = true
	n.codeVersionedContexts[owner] = true
	foreign := &MemoryContext{}
	foreign.ClaimLinks(&Native{})
	foreign.PublishLink(0, 16, 2, [17]uint8{})
	n.ownedContexts[foreign] = true
	n.dispatchEntry, n.dispatchWords, n.codeVersionedLoops = 48, 2, true
	if err := n.Close(); err != nil {
		t.Fatal(err)
	}
	if owner.LookupLink(0) != nil || owner.linkOwner != nil || foreign.LookupLink(0) == nil {
		t.Fatal("close revoked wrong owner's descriptors")
	}
	if n.entryCount != 0 || n.dispatchEntry != 0 || n.dispatchWords != 0 || n.codeVersionedLoops || len(n.ownedContexts) != 0 || len(n.codeVersionedContexts) != 0 {
		t.Fatal("closed owner retained admission")
	}
	for _, entry := range n.entries {
		if entry != (browserEntry{}) {
			t.Fatal("closed owner retained entry/proof metadata")
		}
	}
	if n.AdmitLink(16, 2, 2) == nil || n.Call(16, []uint64{0, 0}) == nil {
		t.Fatal("closed owner accepted stale integer entry")
	}
	// Reusing the same empty capacity cannot recover old proofs or loop status.
	if err := n.recordWasm(16, 3, false, 10); err != nil {
		t.Fatal(err)
	}
	if n.entries[1].proof != (browserProof{}) || n.entries[1].loopInstructions != 0 || n.entries[1].words != 3 {
		t.Fatal("empty metadata reused old proof")
	}
}
