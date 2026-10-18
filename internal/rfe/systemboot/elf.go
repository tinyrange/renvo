package systemboot

import (
	"encoding/binary"
	"fmt"
)

// KernelELF unwraps a Linux bzImage gzip payload or accepts a direct ELF image.
// Returned slices are borrowed or temporary; callers must copy loads before
// releasing their asset arena. No input slice is retained by this package.
func KernelELF(kernel []byte, limit int) (image, header []byte, err error) {
	if len(kernel) > 0x268 && string(kernel[0x202:0x206]) == "HdrS" {
		if len(kernel) < 4096 {
			return nil, nil, fmt.Errorf("truncated Linux setup header")
		}
		off := (uint64(kernel[0x1f1]) + 1) * 512
		if kernel[0x1f1] == 0 {
			off = 5 * 512
		}
		po := uint64(binary.LittleEndian.Uint32(kernel[0x248:]))
		pn := uint64(binary.LittleEndian.Uint32(kernel[0x24c:]))
		if off+po > uint64(len(kernel)) || pn > uint64(len(kernel))-off-po {
			return nil, nil, fmt.Errorf("invalid kernel payload range")
		}
		image, err = Gunzip(kernel[off+po:off+po+pn], limit)
		return image, kernel[:4096], err
	}
	return kernel, nil, nil
}

// LoadELF64X86 checks all program ranges before writing any guest RAM. Entry
// translation uses the original virtual entry, not a previously translated one.
func LoadELF64X86(image, ram []byte) (entry, highest uint64, err error) {
	if len(image) < 64 || string(image[:4]) != "\x7fELF" || image[4] != 2 || image[5] != 1 || image[6] != 1 || binary.LittleEndian.Uint16(image[18:]) != 62 || binary.LittleEndian.Uint32(image[20:]) != 1 || binary.LittleEndian.Uint16(image[52:]) < 64 {
		return 0, 0, fmt.Errorf("kernel must be little-endian x86-64 ELF")
	}
	b := binary.LittleEndian
	original := b.Uint64(image[24:])
	entry = original
	off := b.Uint64(image[32:])
	stride := uint64(b.Uint16(image[54:]))
	count := uint64(b.Uint16(image[56:]))
	if count == 0 || stride < 56 || off > uint64(len(image)) || count > (uint64(len(image))-off)/stride {
		return 0, 0, fmt.Errorf("invalid ELF program table")
	}
	loads := 0
	for j := uint64(0); j < count; j++ {
		p := image[off+j*stride:]
		if b.Uint32(p) != 1 {
			continue
		}
		loads++
		file := b.Uint64(p[8:])
		va := b.Uint64(p[16:])
		pa := b.Uint64(p[24:])
		filesz := b.Uint64(p[32:])
		memsz := b.Uint64(p[40:])
		if pa > uint64(len(ram)) || memsz > uint64(len(ram))-pa || filesz > memsz || file > uint64(len(image)) || filesz > uint64(len(image))-file {
			return 0, 0, fmt.Errorf("kernel load range outside RAM")
		}
		if pa+memsz > highest {
			highest = pa + memsz
		}
		if original >= va && original-va < memsz {
			entry = pa + (original - va)
		}
	}
	if loads == 0 || entry >= uint64(len(ram)) {
		return 0, 0, fmt.Errorf("kernel entry outside RAM")
	}
	for j := uint64(0); j < count; j++ {
		p := image[off+j*stride:]
		if b.Uint32(p) != 1 {
			continue
		}
		file := b.Uint64(p[8:])
		pa := b.Uint64(p[24:])
		filesz := b.Uint64(p[32:])
		memsz := b.Uint64(p[40:])
		copy(ram[pa:pa+filesz], image[file:file+filesz])
		clear(ram[pa+filesz : pa+memsz])
	}
	return entry, highest, nil
}
