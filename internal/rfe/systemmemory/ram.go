// Package systemmemory supplies a checked native RAM view for full-system CPUs.
// MMU policy and architectural faults remain owned by the CPU/machine.
package systemmemory

import (
	"fmt"
	"renvo.dev/internal/rfe/engine"
	emu "renvo.dev/internal/rfe/runtime"
)

// Resolve checks the current privilege-domain mapping and returns a RAM-relative offset.
// For data queries, writable must include any architectural dirty-bit requirement.
// Execute queries use only offset; writable is ignored. Resolve must not
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
	AddressSpace     func() uint64                           // exact host-checked MMU root tag; root switches are not PTE mutations
	AddressSpacePair func() (uint64, uint64)                 // exact ordered root pair; takes precedence over AddressSpace
	Domain           func() int                              // 0 user, 1 mapped kernel, 2 physical kernel
	NativeAddress    func(uint64) uint64                     // reconstruct architectural VA from checked key
	CachedMapping    func(uint64, bool) (uint64, bool, bool) // pure cached translation: offset, writable, present; no walks or retained allocations
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
	pageInvalidation bool   // enabled with a revocation barrier before dependency tracking
	filled           uint16 // one bit for each domain/root with published data pages
	lastDomain       int
	lastSpace        uint64
	lastSpaceHigh    uint64
	spaceValid       bool
	spaces           [3][4]spaceIdentity
	spaceNext        [3]uint8
	CodeWindow       func(uint64) []byte // optional instruction-local, execute-checked ordinary RAM view
	rootedData       bool                // explicit complete-walker/root-isolated owner contract; appended for layout stability
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

// Capture one coherent domain/root observation per query. These callbacks
// inspect architectural control state; no guest operation runs during selection.
func (m *RAM) checkDomain() int {
	domain := m.domain()
	space, spaceHigh := m.addressSpace()
	if domain != m.lastDomain || !m.spaceValid || space != m.lastSpace || spaceHigh != m.lastSpaceHigh {
		m.SelectPrivilege()
		m.lastDomain = domain
		m.lastSpace, m.lastSpaceHigh, m.spaceValid = space, spaceHigh, true
	}
	return domain
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
	if m.filled != 0 {
		for i := range m.native {
			for j := range m.native[i] {
				if m.filled&(1<<uint(i*4+j)) != 0 {
					m.native[i][j].ClearPages()
				}
			}
		}
		m.filled = 0
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
		for domain := range m.spaces {
			for slot := range m.spaces[domain] {
				m.spaces[domain][slot].epoch = m.clock
			}
		}
	}
	m.ForgetData()
}

// MarkPageTable protects dirty/access bits and permission/remapping updates
// from native stores, including through a different physical alias.
func (m *RAM) MarkPageTable(offset uint64) { m.markPageTable(offset, false) }

// MarkPageTableEntry is an explicit aligned eight-byte walker dependency.
// Native stores still trap on the entire physical table page. The bounded
// filter can only over-invalidate: every overlapping observed entry hits.
func (m *RAM) MarkPageTableEntry(offset uint64) { m.markPageTable(offset, offset&7 == 0) }

func (m *RAM) markPageTable(offset uint64, precise bool) {
	if offset >= uint64(len(m.Data)) {
		return
	}
	page := int(offset >> 12)
	if m.AddressSpace != nil || m.AddressSpacePair != nil {
		domain, slot := m.domain(), m.contextSlot()
		space := &m.spaces[domain][slot]
		// An architectural TLB may share a translation between privilege levels.
		// All cached views of this exact root pair depend on the observed walk,
		// even if only one domain actually walked the table.
		for d := range m.spaces {
			for s := range m.spaces[d] {
				other := &m.spaces[d][s]
				if other.valid && other.tag == space.tag && other.tagHigh == space.tagHigh {
					m.addTableOwner(d, s, page)
					if !precise {
						other.coarseTables = true
					} else {
						bit := tableEntryBit(offset >> 3)
						other.tableEntries[bit>>6] |= uint64(1) << uint(bit&63)
					}
				}
			}
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
			m.spaces[domain][slot].coarseTables = false
			clear(m.spaces[domain][slot].tableEntries[:])
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
	barriers := byte(0)
	owners := uint16(0)
	for p := first; p <= last; p++ {
		barriers |= m.watched[p]
		if m.watched[p]&2 != 0 {
			pageFirst, pageLast := offset, (offset + uint64(n) - 1)
			if pageFirst < p<<12 {
				pageFirst = p << 12
			}
			if pageLast > p<<12|4095 {
				pageLast = p<<12 | 4095
			}
			owners |= m.tableEntryOwners(m.tableOwners[p], pageFirst>>3, pageLast>>3)
		}
	}
	if barriers != 0 {
		if barriers&2 != 0 {
			m.profileMapping[0]++
		}
		if barriers&1 != 0 {
			m.profileMapping[1]++
		}
		if barriers&1 != 0 || owners&(1<<15) != 0 {
			// Executable aliases and unscoped walkers keep the global guard.
			m.InvalidateMappings()
		} else if owners != 0 {
			m.invalidateTableOwners(owners)
		} else {
			m.ForgetData()
		}
		for p := first; p <= last; p++ {
			if m.watched[p]&1 != 0 {
				m.pageEpoch[p] = m.clock
			}
		}
	}
}
func (m *RAM) CodeContext() engine.CodeStamp {
	domain := m.checkDomain()
	if !m.usable() {
		return engine.CodeStamp{}
	}
	return m.contextStamp(domain)
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
	// Every successful execute check records its full virtual page under the
	// exact root/domain lifetime. The filter survives epoch changes: old code
	// may later revalidate through a surviving architectural TLB entry.
	if m.pageInvalidation && (m.AddressSpace != nil || m.AddressSpacePair != nil) {
		domain, slot := m.domain(), m.contextSlot()
		bit := executePageBit(address >> 12)
		m.spaces[domain][slot].codePages[bit>>6] |= uint64(1) << uint(bit&63)
	}
	m.codeOwners[offset>>12] |= 1 << uint(m.domain())
	m.watch(offset, 1)
	// Table/ASID changes force execute-checked mapping revalidation, but do
	// not change the identity/version of untouched physical instruction RAM.
	page := offset >> 12
	return engine.CodeStamp{Identity: &m.pageIdentity[m.domain()][page], Epoch: m.pageEpoch[page]}, nil
}
func (m *RAM) NativeContext() *emu.MemoryContext {
	domain := m.checkDomain()
	if !m.usable() {
		return nil
	}
	slot := m.contextSlotFor(domain, m.lastSpace, m.lastSpaceHigh)
	if m.disabled {
		return nil
	}
	context := &m.native[domain][slot]
	context.Clock = &m.clock
	if m.direct != nil && domain == 0 {
		context.DirectGuest, context.DirectHost, context.DirectSize = m.direct.window()
	}
	return context
}

// NativeCodeContext returns the same stamp and refreshed native context as the
// separate queries, but observes architecture callbacks and selects the bounded
// root slot only once. A fifth-root replacement advances that slot's epoch, so form
// the returned stamp only AFTER slot selection and its invalidation effects.
func (m *RAM) NativeCodeContext() (engine.CodeStamp, *emu.MemoryContext) {
	domain := m.checkDomain()
	if !m.usable() {
		return engine.CodeStamp{}, nil
	}
	slot := m.contextSlotFor(domain, m.lastSpace, m.lastSpaceHigh)
	identity, epoch := &m.identity[domain], m.codeEpoch
	if m.AddressSpace != nil || m.AddressSpacePair != nil {
		identity, epoch = &m.spaces[domain][slot].identity, m.spaces[domain][slot].epoch
	}
	if m.disabled {
		return engine.CodeStamp{}, nil
	}
	context := &m.native[domain][slot]
	context.Clock = &m.clock
	if m.direct != nil && domain == 0 {
		context.DirectGuest, context.DirectHost, context.DirectSize = m.direct.window()
	}
	return engine.CodeStamp{Identity: identity, Epoch: epoch}, context
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
	domain, slot := m.domain(), m.contextSlot()
	m.filled |= 1 << uint(domain*4+slot)
	m.native[domain][slot].Fill(number, (*[4096]byte)(m.Data[base:base+4096]), permissions, &m.clock)
}

func (m *RAM) NativeBacking() []byte { return m.Data }

// InstructionBytes is not a mapping or code-ownership cache. The provider may
// expose only execute-checked ordinary RAM, for this one synchronous decode.
func (m *RAM) InstructionBytes(address uint64) []byte {
	if m.CodeWindow == nil || !m.usable() {
		return nil
	}
	return m.CodeWindow(address)
}
