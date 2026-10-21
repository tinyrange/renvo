package systemmemory

import (
	"renvo.dev/internal/rfe/engine"
	emu "renvo.dev/internal/rfe/runtime"
)

type proofPage struct {
	view *emu.MemoryContext
	page uint64
}
type proofEdge struct {
	proof             *codeProof
	physical, virtual uint64
	first, end        uint16
}
type codeProof struct {
	ranges   []engine.CodeRange
	bytes    []byte
	released bool
	book     *proofBook
	view     *emu.MemoryContext
	edges    []proofEdge
	link     *emu.NativeLink
	entry    uint64
	valid    bool
}
type proofBook struct {
	refs     map[uint64]*[4096]uint32
	owner    *RAM
	physical map[uint64]map[*proofEdge]struct{}
	virtual  map[proofPage]map[*proofEdge]struct{}
	live     map[*codeProof]struct{}
}

func (m *RAM) EnableCodeProofs() {
	if m.proofs != nil {
		return
	}
	m.proofs = &proofBook{owner: m, refs: make(map[uint64]*[4096]uint32), physical: make(map[uint64]map[*proofEdge]struct{}), virtual: make(map[proofPage]map[*proofEdge]struct{}), live: make(map[*codeProof]struct{})}
}
func (m *RAM) CodeProofsEnabled() bool { return m.proofs != nil }
func (m *RAM) CodeMutation() uint64    { return m.clock }
func (p *codeProof) Valid() bool       { return p.valid }
func (p *codeProof) Bind(link *emu.NativeLink, entry int) {
	if !p.valid {
		if link != nil && link.Entry == uint64(entry) {
			link.Instructions = 0
		}
		return
	}
	p.link, p.entry = link, uint64(entry)
}
func (p *codeProof) Close() { p.invalidate(); p.released = true; p.bytes = nil; p.ranges = nil }
func (p *codeProof) invalidate() {
	if len(p.edges) == 0 {
		return
	}
	p.valid = false
	if p.link != nil && p.link.Entry == p.entry {
		p.link.Instructions = 0
	}
	p.link = nil
	for i := range p.edges {
		edge := &p.edges[i]
		physical := p.book.physical[edge.physical]
		if p.book.owner.ownedWatches != nil {
			counts := p.book.refs[edge.physical]
			for at := edge.first; at < edge.end; at++ {
				counts[at]--
				if counts[at] == 0 {
					p.book.owner.ownedWatches.Unmark(edge.physical<<12|uint64(at), 1, 1)
				}
			}
		}
		delete(physical, edge)
		if len(physical) == 0 {
			delete(p.book.physical, edge.physical)
			delete(p.book.refs, edge.physical)
		}
		key := proofPage{p.view, edge.virtual}
		virtual := p.book.virtual[key]
		delete(virtual, edge)
		if len(virtual) == 0 {
			delete(p.book.virtual, key)
		}
	}
	delete(p.book.live, p)
	p.edges = nil
}

// A full mapping flush revokes execution/publication authority immediately,
// but unchanged physical dependencies may be retained for allocation-free
// revalidation. They remain watched across aliases until revalidated or closed.
func (b *proofBook) suspend() {
	for p := range b.live {
		p.valid = false
		if p.link != nil && p.link.Entry == p.entry {
			p.link.Instructions = 0
		}
		p.link = nil
	}
}
func (b *proofBook) clear() {
	for p := range b.live {
		p.invalidate()
	}
}
func (b *proofBook) write(offset uint64, n int) {
	last := offset + uint64(n)
	for page := offset >> 12; page <= (last-1)>>12; page++ {
		first, end := max(offset, page<<12), min(last, (page+1)<<12)
		for edge := range b.physical[page] {
			if uint64(edge.first) < end-(page<<12) && first-(page<<12) < uint64(edge.end) {
				edge.proof.invalidate()
			}
		}
	}
}
func (b *proofBook) revoke(view *emu.MemoryContext, address, size uint64) {
	if size == 4096 {
		for edge := range b.virtual[proofPage{view, address >> 12}] {
			edge.proof.invalidate()
		}
		return
	}
	for key, edges := range b.virtual {
		if key.view == view && key.page >= address>>12 && key.page-(address>>12) < size>>12 {
			for edge := range edges {
				edge.proof.invalidate()
			}
		}
	}
}

// resolve checks execute access on each virtual page, compares the exact
// instruction bytes for revalidation, and publishes dependencies only after a
// consistent discovery epoch. A remap to identical bytes may reuse its native
// body; a changed opcode or revoked execute grant cannot.
func (p *codeProof) resolve(mutation uint64, compare bool) bool {
	m := p.book.owner
	if p.released || m.proofs != p.book || !m.usable() || mutation != m.clock {
		return false
	}
	p.view = m.NativeContext()
	var edges []proofEdge
	var bytes []byte
	consumed := 0
	for _, r := range p.ranges {
		if r.Start >= r.End {
			return false
		}
		for at := r.Start; at < r.End; {
			size := min(r.End-at, 4096-(at&4095))
			offset, _, err := m.Resolve(at, true)
			if err != nil || offset >= uint64(len(m.Data)) || size > uint64(len(m.Data))-offset || offset&4095 != at&4095 {
				return false
			}
			part := m.Data[offset : offset+size]
			if compare {
				if len(p.bytes)-consumed < len(part) {
					return false
				}
				for i, b := range part {
					if b != p.bytes[consumed+i] {
						return false
					}
				}
			} else {
				bytes = append(bytes, part...)
			}
			consumed += len(part)
			edges = append(edges, proofEdge{proof: p, physical: offset >> 12, virtual: at >> 12, first: uint16(offset & 4095), end: uint16((offset & 4095) + size)})
			m.MarkCodeBytes(offset, int(size))
			at += size
		}
	}
	if mutation != m.clock {
		return false
	}
	if !compare {
		p.bytes = bytes
	}
	p.edges = edges
	p.valid = true
	m.proofs.live[p] = struct{}{}
	for i := range p.edges {
		edge := &p.edges[i]
		if m.proofs.physical[edge.physical] == nil {
			m.proofs.physical[edge.physical] = make(map[*proofEdge]struct{})
		}
		m.proofs.physical[edge.physical][edge] = struct{}{}
		if m.ownedWatches != nil {
			if m.proofs.refs[edge.physical] == nil {
				m.proofs.refs[edge.physical] = new([4096]uint32)
			}
			counts := m.proofs.refs[edge.physical]
			for at := edge.first; at < edge.end; at++ {
				counts[at]++
			}
			m.ownedWatches.Mark(edge.physical<<12|uint64(edge.first), int(edge.end-edge.first), 1)
		}
		key := proofPage{p.view, edge.virtual}
		if m.proofs.virtual[key] == nil {
			m.proofs.virtual[key] = make(map[*proofEdge]struct{})
		}
		m.proofs.virtual[key][edge] = struct{}{}
	}
	return true
}
func (p *codeProof) Revalidate() bool {
	if p.Valid() {
		return true
	}
	if len(p.edges) != 0 && p.restore() {
		return true
	}
	p.invalidate()
	return p.resolve(p.book.owner.CodeMutation(), true)
}
func (m *RAM) ProveCode(ranges []engine.CodeRange, mutation uint64) engine.CodeProof {
	if m.proofs == nil || len(ranges) == 0 {
		return nil
	}
	p := &codeProof{book: m.proofs, ranges: append([]engine.CodeRange(nil), ranges...)}
	if !p.resolve(mutation, false) {
		return nil
	}
	return p
}

// restore keeps intrusive ownership stable when a mapping flush changed no
// executed byte or physical identity. No dependency maps, masks or snapshots
// are rebuilt merely because the architecture required a fresh execute check.
func (p *codeProof) restore() bool {
	m := p.book.owner
	if p.released || m.proofs != p.book || !m.usable() || m.NativeContext() != p.view {
		return false
	}
	mutation := m.clock
	edges := p.edges
	edgeIndex, consumed := 0, 0
	for _, r := range p.ranges {
		for at := r.Start; at < r.End; {
			size := min(r.End-at, 4096-(at&4095))
			offset, _, err := m.Resolve(at, true)
			if err != nil || edgeIndex >= len(edges) {
				return false
			}
			old := edges[edgeIndex]
			if offset != old.physical<<12|uint64(old.first) || size != uint64(old.end-old.first) || size > uint64(len(m.Data))-offset || int(size) > len(p.bytes)-consumed {
				return false
			}
			for i, b := range m.Data[offset : offset+size] {
				if b != p.bytes[consumed+i] {
					return false
				}
			}
			edgeIndex++
			consumed += int(size)
			at += size
		}
	}
	if mutation != m.clock || edgeIndex != len(edges) || len(p.edges) != len(edges) {
		return false
	}
	p.valid = true
	return true
}
