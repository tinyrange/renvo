package linuxuser

import (
	"fmt"
	"renvo.dev/internal/rfe/engine"
	emu "renvo.dev/internal/rfe/runtime"
)

// Copies of a Memory share both the page map and version clock, so they cannot
// accidentally reuse a generation when updating aliased pages.
type memoryVersions struct {
	direct    directMemory
	closed    bool
	identity  *engine.CodeIdentity
	epoch     uint64
	codeEpoch uint64             // last mutation affecting executable pages
	pageCache [64]pageCacheEntry // shared by copies along with the mapping table
	native    emu.MemoryContext  // host-owned, rooted native data descriptors
}

func (m *Memory) nextVersion() (uint64, error) {
	if m.versions.epoch == ^uint64(0) {
		return 0, fmt.Errorf("code-page version space exhausted")
	}
	m.versions.epoch++
	return m.versions.epoch, nil
}
func (m *Memory) CodeContext() engine.CodeStamp {
	return engine.CodeStamp{Identity: m.versions.identity, Epoch: m.versions.codeEpoch}
}
func (m *Memory) CodeVersion(address uint64) (engine.CodeStamp, error) {
	if !validRange(address, 1) {
		return engine.CodeStamp{}, fmt.Errorf("invalid access at %#x", address)
	}
	p := m.lookupPage(address / PageSize)
	if p == nil || p.permissions&ExecutePermission == 0 {
		return engine.CodeStamp{}, fmt.Errorf("unmapped or forbidden access at %#x", address)
	}
	return engine.CodeStamp{Identity: m.versions.identity, Epoch: p.epoch}, nil
}

// markWritten is called after full validation and successful version allocation.
func (m *Memory) markWritten(p *page, version uint64) {
	p.epoch = version
	if p.permissions&ExecutePermission != 0 {
		m.versions.codeEpoch = version
	}
}

// commit is used only after a full-range write check. Every touched page gets
// a fresh generation. Owned native non-executable stores need no generation:
// they cannot change a valid code stamp, and Protect assigns a fresh generation
// before such a page becomes executable. Executable stores always exit native
// code before mutation, then use this checked invalidation path.
func (m *Memory) commit(address uint64, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	version, err := m.nextVersion()
	if err != nil {
		return err
	}
	first, last := address/PageSize, (address+uint64(len(data))-1)/PageSize
	for at := first; at <= last; at++ {
		m.markWritten(m.pages[at], version)
	}
	m.copyIn(address, data)
	return nil
}
