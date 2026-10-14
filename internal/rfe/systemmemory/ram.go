// Package systemmemory supplies a checked native RAM view for full-system CPUs.
// MMU policy and architectural faults remain owned by the CPU/machine.
package systemmemory

import (
	"fmt"
	"renvo.dev/internal/rfe/engine"
	emu "renvo.dev/internal/rfe/runtime"
)

// Resolve checks the current privilege-domain mapping and returns a RAM-relative offset.
// writable must include any architectural dirty-bit requirement. It must not
// deliver a fault, modify fault registers, or admit devices as ordinary RAM.
type Resolve func(address uint64, execute bool) (offset uint64, writable bool, err error)

type RAM struct {
	direct           directMemory
	Data             []byte
	Profile          bool // bounded host diagnostics only; no guest semantic effects
	profileMapping   [4]uint64
	MultipleArenas   bool // per-domain executable watches have distinct owners
	codeOwners       []byte
	Active           func() bool
	AddressSpace     func() uint64       // exact host-checked MMU root tag; root switches are not PTE mutations
	Domain           func() int          // 0 user, 1 mapped kernel, 2 physical kernel
	NativeAddress    func(uint64) uint64 // reconstruct architectural VA from checked key
	Resolve          Resolve
	ReadGuest        func(uint64, int, bool) (uint64, error)
	WriteGuest       func(uint64, int, uint64) error
	identity         [3]engine.CodeIdentity
	clock, codeEpoch uint64
	pageIdentity     [3][]engine.CodeIdentity
	pageEpoch        []uint64
	watched          []byte // physical-page barriers: code=1, page tables=2
	tableListed      []bool
	tableOwners      []uint16 // 12 bounded root slots plus an untagged-owner bit
	tablePages       []int    // bounded by RAM pages; lifetime ends at a full architectural TLB flush
	native           [3][4]emu.MemoryContext
	disabled         bool
	filled           bool
	lastDomain       int
	lastSpace        uint64
	spaceValid       bool
	spaces           [3][4]spaceIdentity
	spaceNext        [3]uint8
}

func New(data []byte, active func() bool, resolve Resolve) *RAM {
	count := (len(data) + 4095) / 4096
	m := &RAM{Data: data, Active: active, Resolve: resolve, clock: 1, codeEpoch: 1,
		watched: make([]byte, count), tableListed: make([]bool, count), tableOwners: make([]uint16, count), codeOwners: make([]byte, count), pageEpoch: make([]uint64, count)}
	for i := range m.pageIdentity {
		m.pageIdentity[i] = make([]engine.CodeIdentity, count)
	}
	return m
}
func (m *RAM) domain() int {
	if m.Domain != nil {
		return m.Domain()
	}
	return 0
}
func (m *RAM) checkDomain() {
	domain := m.domain()
	space := uint64(0)
	if m.AddressSpace != nil {
		space = m.AddressSpace()
	}
	if domain != m.lastDomain || !m.spaceValid || space != m.lastSpace {
		m.ForgetData()
		m.lastDomain = domain
		m.lastSpace, m.spaceValid = space, true
	}
}
func (m *RAM) usable() bool { return !m.disabled && m.Active() }
func (m *RAM) Read(a uint64, n int, x bool) (uint64, error) {
	if !m.usable() {
		return 0, fmt.Errorf("native user mapping inactive")
	}
	return m.ReadGuest(a, n, x)
}
func (m *RAM) Write(a uint64, n int, v uint64) error { return m.WriteGuest(a, n, v) }
func (m *RAM) ForgetData() {
	if m.direct != nil {
		for i := range m.native {
			for j := range m.native[i] {
				m.native[i][j].DirectGuest, m.native[i][j].DirectHost, m.native[i][j].DirectSize = 0, 0, 0
			}
		}
		m.direct.forget()
	}
	if m.filled {
		for i := range m.native {
			for j := range m.native[i] {
				clear(m.native[i][j].Pages[:])
			}
		}
		m.filled = false
	}
	// Linked code is invalidated by Engine's CodeContext guard, not by data
	// cache misses. Preserve links across unchanged exception/return mappings.
}
func (m *RAM) InvalidateMappings() {
	m.profileMapping[3]++
	if m.clock == ^uint64(0) {
		m.disabled = true
	} else {
		m.clock++
		m.codeEpoch = m.clock
	}
	m.ForgetData()
}

// MarkPageTable protects dirty/access bits and permission/remapping updates
// from native stores, including through a different physical alias.
func (m *RAM) MarkPageTable(offset uint64) {
	if offset >= uint64(len(m.Data)) {
		return
	}
	page := int(offset >> 12)
	if m.AddressSpace != nil {
		domain, slot := m.domain(), m.contextSlot()
		mask := uint16(1) << uint(domain*4+slot)
		if m.tableOwners[page]&mask == 0 {
			m.tableOwners[page] |= mask
			m.spaces[domain][slot].tables = append(m.spaces[domain][slot].tables, page)
		}
	} else {
		m.tableOwners[page] |= 1 << 15
	}
	if !m.tableListed[page] {
		m.tableListed[page] = true
		m.tablePages = append(m.tablePages, page)
	}
	m.watch(offset, 2)
}

// ClearPageTableWatches may be called ONLY after the architecture has revoked
// all software TLB entries. Revoke every native descriptor/code-context stamp
// before releasing old table barriers. Subsequent walks rewatch active tables;
// surviving executable-page barriers and their generation identities remain.
func (m *RAM) ClearPageTableWatches() {
	m.InvalidateMappings()
	for _, page := range m.tablePages {
		m.watched[page] &^= 2
		m.tableListed[page], m.tableOwners[page] = false, 0
	}
	m.tablePages = m.tablePages[:0]
	for domain := range m.spaces {
		for slot := range m.spaces[domain] {
			m.spaces[domain][slot].tables = m.spaces[domain][slot].tables[:0]
		}
	}
}
func (m *RAM) watch(offset uint64, bit byte) {
	if offset >= uint64(len(m.Data)) {
		return
	}
	p := offset >> 12
	if m.watched[p]&bit == 0 {
		m.watched[p] |= bit
		// Revoke already published aliases before the next native entry.
		m.ForgetData()
	}
}

// BeforeWrite is called only after the architectural access was checked, and
// before mutating physical RAM. Ordinary data writes preserve code proofs.
func (m *RAM) BeforeWrite(offset uint64, n int) {
	if n < 1 || offset >= uint64(len(m.Data)) || uint64(n) > uint64(len(m.Data))-offset {
		return
	}
	first, last := offset>>12, (offset+uint64(n)-1)>>12
	changed := false
	for p := first; p <= last; p++ {
		if m.watched[p] != 0 {
			changed = true
		}
	}
	if changed {
		barriers := m.watched[first] | m.watched[last]
		if barriers&2 != 0 {
			m.profileMapping[0]++
		}
		if barriers&1 != 0 {
			m.profileMapping[1]++
		}
		m.InvalidateMappings()
		for p := first; p <= last; p++ {
			if m.watched[p]&1 != 0 {
				m.pageEpoch[p] = m.clock
			}
		}
	}
}
func (m *RAM) CodeContext() engine.CodeStamp {
	m.checkDomain()
	if !m.usable() {
		return engine.CodeStamp{}
	}
	identity := m.contextIdentity()
	return engine.CodeStamp{Identity: identity, Epoch: m.codeEpoch}
}
func (m *RAM) CodeVersion(address uint64) (engine.CodeStamp, error) {
	if !m.usable() {
		return engine.CodeStamp{}, fmt.Errorf("native user mapping inactive")
	}
	offset, _, err := m.Resolve(address, true)
	if err != nil || offset >= uint64(len(m.Data)) {
		if err == nil {
			err = fmt.Errorf("instruction is not in RAM")
		}
		return engine.CodeStamp{}, err
	}
	m.codeOwners[offset>>12] |= 1 << uint(m.domain())
	m.watch(offset, 1)
	// Table/ASID changes force execute-checked mapping revalidation, but do
	// not change the identity/version of untouched physical instruction RAM.
	page := offset >> 12
	return engine.CodeStamp{Identity: &m.pageIdentity[m.domain()][page], Epoch: m.pageEpoch[page]}, nil
}
func (m *RAM) NativeContext() *emu.MemoryContext {
	m.checkDomain()
	if !m.usable() {
		return nil
	}
	context := &m.native[m.domain()][m.contextSlot()]
	context.Clock = &m.clock
	if m.direct != nil && m.domain() == 0 {
		context.DirectGuest, context.DirectHost, context.DirectSize = m.direct.window()
	}
	return context
}
func (m *RAM) NativeCodeVersions() bool { return m.usable() }
func (m *RAM) NativeRefill(address uint64) {
	m.checkDomain()
	if !m.usable() {
		return
	}
	va := address
	if m.NativeAddress != nil {
		va = m.NativeAddress(address)
	}
	offset, writable, err := m.Resolve(va, false)
	number := address >> 12
	if err != nil || offset >= uint64(len(m.Data)) || offset&4095 != address&4095 {
		m.native[m.domain()][m.contextSlot()].Forget(number)
		if m.direct != nil {
			m.direct.forgetPage(address)
		}
		return
	}
	base := offset &^ uint64(4095)
	if uint64(len(m.Data))-base < 4096 {
		m.native[m.domain()][m.contextSlot()].Forget(number)
		if m.direct != nil {
			m.direct.forgetPage(address)
		}
		return
	}
	permissions := uint64(1)
	if writable {
		permissions |= 2
	}
	// This bit is a host write barrier, not an architectural execute grant.
	// CodeVersion independently checks guest execute permissions.
	if m.watched[base>>12] != 0 {
		permissions |= 4
	}
	if m.direct != nil && m.domain() == 0 {
		m.direct.refill(address, offset, permissions)
	}
	m.filled = true
	m.native[m.domain()][m.contextSlot()].Fill(number, (*[4096]byte)(m.Data[base:base+4096]), permissions, &m.clock)
}

func (m *RAM) NativeBacking() []byte { return m.Data }
