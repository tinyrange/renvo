// Package rfeabi defines the GC-visible context shared by the emulator and
// native-call runtime. Native byte offsets are declared by the emitter and
// verified against these types before any entry is admitted.
package rfeabi

type Page struct {
	Number      uint64
	Data        *[4096]byte
	Permissions uint64
	Epoch       *uint64
}

type Descriptor struct {
	PC, Entry, Instructions, Reserved uint64
	Prefix                            [17]uint8
	Padding                           [7]uint8
	Generation                        uint64
}

// Context contains no owner token. Encoded borrowed addresses must be zero
// outside the serialized native call. Pointers remain typed for Go's GC;
// foreign execution uses a separately pinned, pointer-free staging image.
type LinkTable struct {
	Blocks        [1024]Descriptor
	Epoch, Victim uint64
	Base          *Descriptor
	Mask          uint64
	entries       []Descriptor
	used          int
}

type Context struct {
	Retired, Status, Address                   uint64
	Clock                                      *uint64
	Pages                                      [64]Page
	Remaining, Total, MemoryTotal              uint64
	CodeView                                   [4]uint64
	Blocks                                     [1024]Descriptor
	LoopExits, LoopIterations, PreparedTargets uint64
	DescriptorBase, AdmissionEpoch             uint64
	// Owned direct window; usable only by registered foreign fault sites.
	DirectGuest, DirectHost, DirectSize uint64
	Memory                              *OwnedMemory // optional scalable owned-RAM mapping and exact store barriers
	Stop                                *uint32      // host atomic notification
	LinkEpoch                           uint64       // O(1) descriptor revocation; zero is the initial generation

	Links          *LinkTable // owner-selected code-domain publications, separate from RAM mappings
	DescriptorMask uint64     // zero retains the fixed four-way compatibility table
}

func (m *Context) LinkDescriptors() *[1024]Descriptor {
	if m.Links != nil {
		return &m.Links.Blocks
	}
	return &m.Blocks
}

// Growable publications never evict a live PC because of a hash collision.
// Growth starts an empty publication generation rather than copying entries:
// exact proof bindings into retired storage can never authorize a moved copy.
func NewLinkTable() *LinkTable { t := &LinkTable{Epoch: 1}; t.grow(16); return t }
func (t *LinkTable) grow(size int) {
	t.entries = make([]Descriptor, size)
	t.Base = &t.entries[0]
	t.Mask = uint64(size - 1)
	t.used = 0
}
func (t *LinkTable) Dynamic() bool         { return t != nil && t.Base != nil }
func (t *LinkTable) Entries() []Descriptor { return t.entries }
func (t *LinkTable) Valid() bool {
	return !t.Dynamic() || len(t.entries) > 0 && len(t.entries)&(len(t.entries)-1) == 0 && t.Mask == uint64(len(t.entries)-1) && t.Base == &t.entries[0]
}
func (m *Context) DescriptorPointer() *Descriptor {
	if m.Links.Dynamic() {
		return m.Links.Base
	}
	return &m.LinkDescriptors()[0]
}
func linkHash(pc uint64) uint64 { return pc ^ (pc >> 12) }
func (t *LinkTable) Lookup(pc, epoch uint64) *Descriptor {
	for i, probes := linkHash(pc)&t.Mask, 0; probes < len(t.entries); i, probes = (i+1)&t.Mask, probes+1 {
		p := &t.entries[i]
		if p.Generation != epoch {
			return nil
		}
		if p.PC == pc {
			if p.Instructions == 0 {
				return nil
			}
			return p
		}
	}
	return nil
}
func (t *LinkTable) Publish(d Descriptor) {
	if old := t.Lookup(d.PC, d.Generation); old != nil {
		*old = d
		return
	}
	if (t.used+1)*2 > len(t.entries) {
		t.grow(len(t.entries) * 2)
	}
	for i := linkHash(d.PC) & t.Mask; ; i = (i + 1) & t.Mask {
		p := &t.entries[i]
		if p.Generation != d.Generation {
			*p = d
			t.used++
			return
		}
		if p.PC == d.PC {
			*p = d
			return
		}
	}
}
func (t *LinkTable) ClearDynamic() {
	t.used = 0
	if t.Epoch == 0 {
		clear(t.entries)
	}
}
