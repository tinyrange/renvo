//go:build !renvo

package runimage

import (
	"renvo.dev/internal/rfeabi"
	"testing"
	"unsafe"
)

func TestSelectedLinkTableAliasesRequireCompatibility(t *testing.T) {
	m := new(LinkedContextABI)
	m.Links = new(rfeabi.LinkTable)
	state := []uint64{0, 0}
	backing := make([]byte, 4096)
	if sessionLinksAlias(state, m, backing) {
		t.Fatal("disjoint code table rejected")
	}
	m.Clock = &m.Links.Blocks[0].Entry
	if !sessionLinksAlias(state, m, backing) {
		t.Fatal("native clock can mutate cached publication")
	}
	m.Clock = nil
	m.Stop = (*uint32)(unsafe.Pointer(&m.Links.Blocks[0].Instructions))
	if !sessionLinksAlias(state, m, backing) {
		t.Fatal("host notification aliases publication")
	}
	m.Stop = nil
	m.Pages[0].Epoch = &m.Links.Blocks[0].Instructions
	if !sessionLinksAlias(state, m, backing) {
		t.Fatal("guest epoch can mutate cached publication")
	}
	m.Pages[0].Epoch = nil
	alias := unsafe.Slice((*byte)(unsafe.Pointer(m.Links)), 4096)
	if !sessionLinksAlias(state, m, alias) {
		t.Fatal("guest RAM aliases public table")
	}
}
