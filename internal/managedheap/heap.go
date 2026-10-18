package managedheap

import "unsafe"

// Memory supplies nonmoving, writable, zeroed segments. Acquire must return
// exactly size bytes, aligned to Granule, or nil without retaining any storage.
// Release must release exactly the matching acquisition. Neither operation may
// allocate from this heap or invoke its collector. Adapter-owned bookkeeping
// must be included in Config.Overhead (or stored inside the supplied segments).
// The provider must keep segments alive independently of host Go's collector.
type Memory interface {
	Acquire(size uintptr) unsafe.Pointer
	Release(base unsafe.Pointer, size uintptr)
}

// Roots enumerates every live reference while application execution is stopped:
// globals, active and suspended stacks, saved registers, and runtime handles.
// It calls Mark for each pointer, including interior pointers. It must neither
// allocate from this heap nor resume application execution. Return false if root
// enumeration cannot finish; collection then preserves ALL allocations.
// A target must not enable collection until it implements this whole contract.
type Roots interface {
	MarkRoots(heap *Heap) bool
}

type Config struct {
	// Limit includes segments, Heap itself and Overhead; zero means no user cap.
	Limit uint64
	// Granule is the provider's allocation granularity, a power of two.
	Granule uintptr
	// Initial is a preferred first segment size, not a reservation or minimum.
	Initial uintptr
	// Overhead accounts for fixed provider and root-enumerator storage.
	Overhead uintptr
}

// Segment and block metadata live in the provider's storage. Collection uses no
// work queue, recursive tracing, slice growth, or managed allocation.
type segment struct {
	next *segment
	size uintptr
}

type block struct {
	span  uintptr // header plus payload, including alignment padding
	flags uintptr
}

const (
	allocated           = uintptr(1)
	marked              = uintptr(2)
	scanned             = uintptr(4)
	wordSize    uintptr = unsafe.Sizeof(uintptr(0))
	alignment   uintptr = 2 * wordSize
	segmentSize uintptr = unsafe.Sizeof(segment{})
	blockSize   uintptr = unsafe.Sizeof(block{})
)

// Heap must not be copied after Init. Its bookkeeping is charged to the budget
// even when the caller puts it in static storage. Methods require serialized
// access; this core does not stop threads or discover roots itself.
type Heap struct {
	memory   Memory
	roots    Roots
	first    *segment
	limit    uintptr
	reserved uintptr
	granule  uintptr
	nextSize uintptr
	busy     bool
}

// Init does not acquire storage. A false result leaves h untouched. A live heap
// cannot be reinitialized: call Close first, after all clients are finished.
func (h *Heap) Init(memory Memory, roots Roots, config Config) bool {
	if h.memory != nil || memory == nil || roots == nil {
		return false
	}
	granule := config.Granule
	if granule < segmentSize+blockSize+alignment || granule&(granule-1) != 0 {
		return false
	}
	maximum := ^uintptr(0)
	limit := maximum
	if config.Limit != 0 {
		if config.Limit > uint64(maximum) {
			return false
		}
		limit = uintptr(config.Limit)
	}
	if config.Overhead > maximum-unsafe.Sizeof(Heap{}) {
		return false
	}
	reserved := unsafe.Sizeof(Heap{}) + config.Overhead
	if reserved > limit {
		return false
	}
	initial := config.Initial
	if initial == 0 {
		initial = granule
	}
	if initial > maximum-(granule-1) {
		return false
	}
	initial = (initial + granule - 1) &^ (granule - 1)
	*h = Heap{memory: memory, roots: roots, limit: limit, reserved: reserved,
		granule: granule, nextSize: initial}
	return true
}

// Reserved includes retained free space and all charged bookkeeping, not only
// live payload bytes. It never exceeds the effective limit.
func (h *Heap) Reserved() uintptr { return h.reserved }

// Alloc returns zeroed, aligned storage or nil. It first reuses free blocks,
// then collects and retries, then attempts bounded growth. A target adapter
// converts nil into the ordinary recoverable out-of-memory panic, not a signal.
// Zero-byte allocations occupy one alignment unit so they have valid storage.
func (h *Heap) Alloc(size uintptr) unsafe.Pointer {
	if h.memory == nil || h.busy {
		return nil
	}
	if size == 0 {
		size = alignment
	}
	if size > ^uintptr(0)-(alignment-1)-blockSize {
		return nil
	}
	span := ((size + alignment - 1) &^ (alignment - 1)) + blockSize
	if pointer := h.take(span); pointer != nil {
		return pointer
	}
	if h.first != nil {
		if !h.Collect() {
			return nil
		}
		if pointer := h.take(span); pointer != nil {
			return pointer
		}
	}
	if span > ^uintptr(0)-segmentSize-(h.granule-1) {
		return nil
	}
	minimum := (span + segmentSize + h.granule - 1) &^ (h.granule - 1)
	available := (h.limit - h.reserved) &^ (h.granule - 1)
	if minimum > available {
		return nil
	}
	growth := h.nextSize
	if growth < minimum {
		growth = minimum
	}
	if growth > available {
		growth = available
	}
	h.busy = true
	base := h.memory.Acquire(growth)
	// A large preferred segment may fail even when the requested object fits.
	if base == nil && growth != minimum {
		growth = minimum
		base = h.memory.Acquire(growth)
	}
	h.busy = false
	if base == nil {
		return nil
	}
	s := (*segment)(base)
	s.next = h.first
	s.size = growth
	h.first = s
	b := (*block)(unsafe.Add(base, segmentSize))
	b.span = growth - segmentSize
	b.flags = 0
	h.reserved += growth
	if growth <= ^uintptr(0)/2 {
		h.nextSize = growth * 2
	} else {
		h.nextSize = growth
	}
	return h.take(span)
}

func (h *Heap) take(span uintptr) unsafe.Pointer {
	for s := h.first; s != nil; s = s.next {
		for offset := segmentSize; offset < s.size; {
			b := (*block)(unsafe.Add(unsafe.Pointer(s), offset))
			if b.flags == 0 && b.span >= span {
				if b.span-span >= blockSize+alignment {
					rest := (*block)(unsafe.Add(unsafe.Pointer(b), span))
					rest.span = b.span - span
					rest.flags = 0
					b.span = span
				}
				b.flags = allocated
				// Swept payloads and absorbed headers must not expose old data.
				for i := blockSize; i < b.span; i += wordSize {
					*(*uintptr)(unsafe.Add(unsafe.Pointer(b), i)) = 0
				}
				return unsafe.Add(unsafe.Pointer(b), blockSize)
			}
			offset += b.span
		}
	}
	return nil
}

// Mark retains the allocation containing pointer. Nonheap values, freed blocks
// and allocator metadata are ignored. Mark is only effective during collection.
func (h *Heap) Mark(pointer uintptr) {
	if !h.busy {
		return
	}
	for s := h.first; s != nil; s = s.next {
		base := uintptr(unsafe.Pointer(s))
		if pointer < base || pointer-base >= s.size {
			continue
		}
		for offset := segmentSize; offset < s.size; {
			b := (*block)(unsafe.Add(unsafe.Pointer(s), offset))
			if pointer-base >= offset+blockSize && pointer-base < offset+b.span {
				if b.flags&allocated != 0 {
					b.flags |= marked
				}
				return
			}
			offset += b.span
		}
		return
	}
}

// Collect is a nonmoving conservative mark/sweep pass. Payloads are scanned at
// native-word alignment, so a target must separately report any unaligned or
// encoded references. False-positive pointer words can retain garbage. Root
// enumeration must be complete; absence of a root is permission to reclaim it.
// False means collection was unavailable or root enumeration was incomplete.
func (h *Heap) Collect() bool {
	if h.memory == nil || h.busy {
		return false
	}
	h.busy = true
	if !h.roots.MarkRoots(h) {
		h.clearMarks()
		h.busy = false
		return false
	}
	// Fixed-point tracing trades scanning work for zero mark-stack allocation.
	// Marked blocks are scanned once; repeat when a scan marks an earlier block.
	for {
		progress := false
		for s := h.first; s != nil; s = s.next {
			for offset := segmentSize; offset < s.size; {
				b := (*block)(unsafe.Add(unsafe.Pointer(s), offset))
				if b.flags == allocated|marked {
					b.flags |= scanned
					progress = true
					for i := blockSize; i < b.span; i += wordSize {
						h.Mark(*(*uintptr)(unsafe.Add(unsafe.Pointer(b), i)))
					}
				}
				offset += b.span
			}
		}
		if !progress {
			break
		}
	}
	var previous *segment
	for s := h.first; s != nil; {
		var free *block
		live := false
		for offset := segmentSize; offset < s.size; {
			b := (*block)(unsafe.Add(unsafe.Pointer(s), offset))
			span := b.span
			if b.flags&marked != 0 {
				b.flags = allocated
				live = true
				free = nil
			} else {
				b.flags = 0
				if free != nil {
					free.span += span
				} else {
					free = b
				}
			}
			offset += span
		}
		next := s.next
		if !live {
			if previous == nil {
				h.first = next
			} else {
				previous.next = next
			}
			size := s.size
			h.memory.Release(unsafe.Pointer(s), size)
			h.reserved -= size
		} else {
			previous = s
		}
		s = next
	}
	h.busy = false
	return true
}

func (h *Heap) clearMarks() {
	for s := h.first; s != nil; s = s.next {
		for offset := segmentSize; offset < s.size; {
			b := (*block)(unsafe.Add(unsafe.Pointer(s), offset))
			b.flags &= allocated
			offset += b.span
		}
	}
}

// Close releases all segments, including live objects. The caller must ensure
// no client can access them afterward. It is not a substitute for Collect.
func (h *Heap) Close() {
	if h.memory == nil || h.busy {
		return
	}
	h.busy = true
	for s := h.first; s != nil; {
		next := s.next
		h.memory.Release(unsafe.Pointer(s), s.size)
		s = next
	}
	*h = Heap{}
}
