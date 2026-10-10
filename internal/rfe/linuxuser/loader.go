// Package linuxuser implements bounded ELF64 loading and Linux user services.
// ISA-specific register and trap conventions are supplied by the guest adapter.
package linuxuser

import (
	"encoding/binary"
	"fmt"
	"strings"
)

const StackTop uint64 = 0x7fff00000000
const StackSize uint64 = 1 << 20

// ELFABI makes machine identity, accepted flags and minimum executable entry
// extent explicit. It is not permission to execute an unsupported instruction.
type ELFABI struct {
	Machine        uint16
	AllowedFlags   uint32
	EntryAlignment uint64
	EntryBytes     int
	// DirectMemory requests the optional host-mapped backend; unsupported
	// hosts retain checked pages without changing guest addresses or limits.
	DirectMemory bool
}

type Process struct {
	Entry, StackPointer        uint64
	Memory                     *Memory
	Break, breakBase, mmapNext uint64
	// Clock supplies realtime (0) or monotonic elapsed (1) seconds/nanoseconds.
	// A nil capability reports ENOSYS; it is never replaced with guest ticks.
	Clock func(uint64) (int64, int64, bool)
}

// Load accepts only static ET_EXEC files. Section tables are irrelevant to
// execution and are never parsed or allocated. The bounded program table is
// validated before any guest memory allocation. Failures expose no address space.
// entropy must fill its entire buffer with cryptographically secure bytes or
// return an error. These bytes supply AT_RANDOM. The host
// supplies this capability; the portable loader never substitutes weak entropy.
func Load(image []byte, args []string, abi ELFABI, entropy func([]byte) error) (*Process, error) {
	if entropy == nil {
		return nil, fmt.Errorf("missing ELF startup entropy")
	}
	if abi.Machine == 0 || abi.EntryAlignment == 0 || abi.EntryAlignment > 8 || abi.EntryAlignment&(abi.EntryAlignment-1) != 0 || abi.EntryBytes != 1 && abi.EntryBytes != 2 && abi.EntryBytes != 4 && abi.EntryBytes != 8 {
		return nil, fmt.Errorf("invalid ELF guest ABI")
	}
	if len(image) < 64 || len(image) > 64<<20 {
		return nil, fmt.Errorf("invalid ELF image size")
	}
	u16, u32, u64 := binary.LittleEndian.Uint16, binary.LittleEndian.Uint32, binary.LittleEndian.Uint64
	if string(image[:4]) != "\x7fELF" || image[4] != 2 || image[5] != 1 || image[6] != 1 || image[7] != 0 && image[7] != 3 || u16(image[16:]) != 2 || u16(image[18:]) != abi.Machine || u32(image[20:]) != 1 || u32(image[48:])&^abi.AllowedFlags != 0 || u16(image[52:]) != 64 {
		return nil, fmt.Errorf("requires compatible little-endian static ET_EXEC ELF")
	}
	entry, phoff, count := u64(image[24:]), u64(image[32:]), uint64(u16(image[56:]))
	phsize := count * 56
	if count == 0 || count > 128 || entry&(abi.EntryAlignment-1) != 0 || u16(image[54:]) != 56 || phoff > uint64(len(image)) || phsize > uint64(len(image))-phoff {
		return nil, fmt.Errorf("invalid ELF entry or program headers")
	}
	type program struct {
		kind, flags                                      uint32
		offset, address, fileSize, memorySize, alignment uint64
	}
	programs := make([]program, int(count))
	for i := range programs {
		record := image[phoff+uint64(i)*56:]
		p := program{u32(record), u32(record[4:]), u64(record[8:]), u64(record[16:]), u64(record[32:]), u64(record[40:]), u64(record[48:])}
		programs[i] = p
		if p.kind == 2 || p.kind == 3 || p.kind == 7 {
			return nil, fmt.Errorf("dynamic linking and TLS startup unsupported")
		}
		if p.kind == 0x6474e551 && p.flags&1 != 0 {
			return nil, fmt.Errorf("executable stack unsupported")
		}
		if p.kind != 1 {
			continue
		}
		if p.fileSize > p.memorySize || p.offset > uint64(len(image)) || p.fileSize > uint64(len(image))-p.offset || !validRange(p.address, p.memorySize) || p.memorySize > MemoryLimit || p.flags&^uint32(7) != 0 {
			return nil, fmt.Errorf("invalid ELF load segment")
		}
		if p.alignment > 1 && (p.alignment&(p.alignment-1) != 0 || p.address%p.alignment != p.offset%p.alignment) {
			return nil, fmt.Errorf("invalid ELF segment alignment")
		}
	}
	m := NewMemory()
	if abi.DirectMemory {
		if direct, err := NewDirectMemory(); err == nil {
			m = direct
		}
	}
	loaded := false
	defer func() {
		if !loaded {
			_ = m.Close()
		}
	}()
	var high, phdr uint64
	type interval struct{ start, end uint64 }
	var segments []interval
	for _, p := range programs {
		if p.kind != 1 || p.memorySize == 0 {
			continue
		}
		end := p.address + p.memorySize
		for _, prior := range segments {
			if p.address < prior.end && end > prior.start {
				return nil, fmt.Errorf("overlapping ELF segments")
			}
		}
		segments = append(segments, interval{p.address, end})
		permissions := uint8(0)
		if p.flags&4 != 0 {
			permissions |= ReadPermission
		}
		if p.flags&2 != 0 {
			permissions |= WritePermission
		}
		if p.flags&1 != 0 {
			permissions |= ExecutePermission
		}
		for at := p.address &^ (PageSize - 1); at < pageUp(end); at += PageSize {
			existing := m.pages[at/PageSize]
			if existing != nil {
				existing.permissions |= permissions
			} else if err := m.Map(at, PageSize, permissions); err != nil {
				return nil, err
			}
		}
		m.copyIn(p.address, image[p.offset:p.offset+p.fileSize])
		if end > high {
			high = end
		}
		if phoff >= p.offset && phoff-p.offset <= p.fileSize && phsize <= p.fileSize-(phoff-p.offset) {
			phdr = p.address + phoff - p.offset
		}
	}
	if len(segments) == 0 || phdr == 0 {
		return nil, fmt.Errorf("ELF program headers must be mapped")
	}
	if _, err := m.Read(entry, abi.EntryBytes, true); err != nil {
		return nil, fmt.Errorf("ELF entry: %w", err)
	}
	if err := m.Check(phdr, phsize, ReadPermission); err != nil {
		return nil, fmt.Errorf("ELF program headers: %w", err)
	}
	var err error
	if err = m.Map(StackTop-StackSize, StackSize, ReadPermission|WritePermission); err != nil {
		return nil, err
	}
	if len(args) == 0 {
		args = []string{"guest"}
	}
	if len(args) > 256 {
		return nil, fmt.Errorf("too many guest arguments")
	}
	cursor := StackTop
	push := func(data []byte) (uint64, error) {
		if uint64(len(data)) > cursor-(StackTop-StackSize) {
			return 0, fmt.Errorf("initial stack overflow")
		}
		cursor -= uint64(len(data))
		return cursor, m.WriteBytes(cursor, data)
	}
	pointers := make([]uint64, len(args))
	total := 0
	for i := len(args) - 1; i >= 0; i-- {
		total += len(args[i]) + 1
		if total > 65536 || strings.ContainsRune(args[i], 0) {
			return nil, fmt.Errorf("invalid guest argument")
		}
		pointers[i], err = push(append([]byte(args[i]), 0))
		if err != nil {
			return nil, err
		}
	}
	random := make([]byte, 16)
	if err = entropy(random); err != nil {
		return nil, err
	}
	randomAddress, err := push(random)
	if err != nil {
		return nil, err
	}
	// argc, argv[], NULL, empty envp, and a conservative auxiliary vector.
	words := []uint64{uint64(len(args))}
	words = append(words, pointers...)
	words = append(words, 0, 0)
	words = append(words, 3, phdr, 4, 56, 5, count, 6, PageSize, 7, 0, 9, entry, 11, 0, 12, 0, 13, 0, 14, 0, 16, 0, 23, 0, 25, randomAddress, 31, pointers[0], 0, 0)
	cursor = (cursor - uint64(len(words))*8) &^ uint64(15)
	for i, value := range words {
		if err = m.Write(cursor+uint64(i)*8, 8, value); err != nil {
			return nil, err
		}
	}
	end := pageUp(high)
	p := &Process{Memory: m, Break: end, breakBase: end, mmapNext: 0x100000000}
	p.Entry, p.StackPointer = entry, cursor
	loaded = true
	return p, nil
}
