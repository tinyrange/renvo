//go:build !renvo

// Package linkedimage decodes Renvo's format-neutral linked-image transport.
//
// An Image presents the same code/data/entry model for ELF, PE, Mach-O, and
// WebAssembly payloads. Native remains available as a lossless fallback for an
// operating-system loader; consumers do not need to understand its container
// merely to inspect the linked layout.
package linkedimage

import (
	"encoding/binary"
	"errors"
)

const (
	Read = 1 << iota
	Write
	Execute
)

var (
	ErrHeader   = errors.New("invalid Renvo linked-image header")
	ErrChecksum = errors.New("invalid Renvo linked-image checksum")
	ErrPayload  = errors.New("invalid native linked-image payload")
)

type Segment struct {
	Name        string
	Address     uint64
	MemorySize  uint64
	Alignment   uint64
	FileOffset  uint64
	Permissions uint8
	Data        []byte
}

type Image struct {
	Target   string
	Format   int
	Base     uint64
	Entry    uint64
	Segments []Segment
	Native   []byte
}

func Decode(src []byte) (Image, error) {
	target, format, native, status := payload(src)
	if status == payloadHeader {
		return Image{}, ErrHeader
	}
	if status == payloadChecksum {
		return Image{}, ErrChecksum
	}
	image := Image{
		Target: targetName(target),
		Format: format,
		Native: native,
	}
	var ok bool
	switch image.Format {
	case FormatELF:
		ok = parseELF(&image)
	case FormatPE:
		ok = parsePE(&image)
	case FormatMachO:
		ok = parseMachO(&image)
	case FormatWasm:
		ok = parseWasm(&image)
	case FormatBytecode:
		ok = parseBytecode(&image)
	}
	if !ok || image.Target == "" {
		return Image{}, ErrPayload
	}
	return image, nil
}

func targetName(target int) string {
	switch target {
	case 1:
		return "linux/amd64"
	case 2:
		return "linux/386"
	case 3:
		return "linux/aarch64"
	case 4:
		return "linux/arm"
	case 5:
		return "windows/amd64"
	case 6:
		return "windows/386"
	case 7:
		return "wasi/wasm32"
	case 8:
		return "darwin/arm64"
	case 9:
		return "linux-kernel/amd64"
	case 10:
		return "windows/arm64"
	case 11:
		return "vm/vm32"
	default:
		return ""
	}
}

func parseBytecode(image *Image) bool {
	data := image.Native
	if len(data) < 40 || string(data[:4]) != "RNVB" ||
		binary.LittleEndian.Uint16(data[4:6]) != 1 ||
		binary.LittleEndian.Uint16(data[6:8]) != 40 {
		return false
	}
	codeSize := int(binary.LittleEndian.Uint32(data[8:12]))
	dataSize := int(binary.LittleEndian.Uint32(data[12:16]))
	routineCount := int(binary.LittleEndian.Uint32(data[28:32]))
	tableSize := routineCount * 12
	payloadSize := 40 + tableSize + codeSize + dataSize
	if codeSize <= 0 || routineCount <= 0 || tableSize < 0 || payloadSize < 40 || payloadSize != len(data) {
		return false
	}
	a, b := imageAdler(data[40:])
	checksum := uint32(a) | uint32(b)<<16
	if binary.LittleEndian.Uint32(data[32:36]) != checksum {
		return false
	}
	image.Segments = append(image.Segments, Segment{
		Name:        ".bytecode",
		MemorySize:  uint64(len(data)),
		Alignment:   1,
		Permissions: Read | Execute,
		Data:        data,
	})
	return true
}

func parseELF(image *Image) bool {
	data := image.Native
	if len(data) < 52 || string(data[:4]) != "\x7fELF" || data[5] != 1 {
		return false
	}
	class := data[4]
	var entry, phoff uint64
	var phentsize, phnum int
	if class == 2 {
		if len(data) < 64 {
			return false
		}
		entry = binary.LittleEndian.Uint64(data[24:32])
		phoff = binary.LittleEndian.Uint64(data[32:40])
		phentsize = int(binary.LittleEndian.Uint16(data[54:56]))
		phnum = int(binary.LittleEndian.Uint16(data[56:58]))
	} else if class == 1 {
		entry = uint64(binary.LittleEndian.Uint32(data[24:28]))
		phoff = uint64(binary.LittleEndian.Uint32(data[28:32]))
		phentsize = int(binary.LittleEndian.Uint16(data[42:44]))
		phnum = int(binary.LittleEndian.Uint16(data[44:46]))
	} else {
		return false
	}
	base := ^uint64(0)
	for i := 0; i < phnum; i++ {
		at, ok := tableEntry(phoff, phentsize, i, len(data))
		if !ok || phentsize < 32 || binary.LittleEndian.Uint32(data[at:at+4]) != 1 {
			if !ok {
				return false
			}
			continue
		}
		var flags uint32
		var offset, address, fileSize, memorySize, align uint64
		if class == 2 {
			if phentsize < 56 {
				return false
			}
			flags = binary.LittleEndian.Uint32(data[at+4 : at+8])
			offset = binary.LittleEndian.Uint64(data[at+8 : at+16])
			address = binary.LittleEndian.Uint64(data[at+16 : at+24])
			fileSize = binary.LittleEndian.Uint64(data[at+32 : at+40])
			memorySize = binary.LittleEndian.Uint64(data[at+40 : at+48])
			align = binary.LittleEndian.Uint64(data[at+48 : at+56])
		} else {
			offset = uint64(binary.LittleEndian.Uint32(data[at+4 : at+8]))
			address = uint64(binary.LittleEndian.Uint32(data[at+8 : at+12]))
			fileSize = uint64(binary.LittleEndian.Uint32(data[at+16 : at+20]))
			memorySize = uint64(binary.LittleEndian.Uint32(data[at+20 : at+24]))
			flags = binary.LittleEndian.Uint32(data[at+24 : at+28])
			align = uint64(binary.LittleEndian.Uint32(data[at+28 : at+32]))
		}
		payload, ok := byteRange(data, offset, fileSize)
		if !ok || fileSize > memorySize {
			return false
		}
		if address < base {
			base = address
		}
		image.Segments = append(image.Segments, Segment{
			Address: address, MemorySize: memorySize, Alignment: align,
			FileOffset: offset, Permissions: elfPermissions(flags), Data: payload,
		})
	}
	if len(image.Segments) == 0 {
		return false
	}
	image.Base = base
	image.Entry = entry - base
	for i := range image.Segments {
		image.Segments[i].Address -= base
	}
	return true
}

func elfPermissions(flags uint32) uint8 {
	var result uint8
	if flags&4 != 0 {
		result |= Read
	}
	if flags&2 != 0 {
		result |= Write
	}
	if flags&1 != 0 {
		result |= Execute
	}
	return result
}

func parsePE(image *Image) bool {
	data := image.Native
	if len(data) < 64 || string(data[:2]) != "MZ" {
		return false
	}
	pe := int(binary.LittleEndian.Uint32(data[60:64]))
	if pe < 0 || pe+24 > len(data) || string(data[pe:pe+4]) != "PE\x00\x00" {
		return false
	}
	sections := int(binary.LittleEndian.Uint16(data[pe+6 : pe+8]))
	optionalSize := int(binary.LittleEndian.Uint16(data[pe+20 : pe+22]))
	optional := pe + 24
	sectionTable := optional + optionalSize
	if optionalSize < 64 || sectionTable < optional || sectionTable > len(data) {
		return false
	}
	magic := binary.LittleEndian.Uint16(data[optional : optional+2])
	image.Entry = uint64(binary.LittleEndian.Uint32(data[optional+16 : optional+20]))
	if magic == 0x20b {
		if optionalSize < 68 {
			return false
		}
		image.Base = binary.LittleEndian.Uint64(data[optional+24 : optional+32])
	} else if magic == 0x10b {
		image.Base = uint64(binary.LittleEndian.Uint32(data[optional+28 : optional+32]))
	} else {
		return false
	}
	headerSize := uint64(binary.LittleEndian.Uint32(data[optional+60 : optional+64]))
	headers, ok := byteRange(data, 0, headerSize)
	if !ok {
		return false
	}
	image.Segments = append(image.Segments, Segment{Name: "headers", MemorySize: headerSize, Alignment: 1, Permissions: Read, Data: headers})
	for i := 0; i < sections; i++ {
		at, ok := tableEntry(uint64(sectionTable), 40, i, len(data))
		if !ok {
			return false
		}
		virtualSize := uint64(binary.LittleEndian.Uint32(data[at+8 : at+12]))
		address := uint64(binary.LittleEndian.Uint32(data[at+12 : at+16]))
		rawSize := uint64(binary.LittleEndian.Uint32(data[at+16 : at+20]))
		rawOffset := uint64(binary.LittleEndian.Uint32(data[at+20 : at+24]))
		memorySize := virtualSize
		if rawSize > memorySize {
			memorySize = rawSize
		}
		payload, ok := byteRange(data, rawOffset, rawSize)
		if !ok {
			return false
		}
		characteristics := binary.LittleEndian.Uint32(data[at+36 : at+40])
		image.Segments = append(image.Segments, Segment{
			Name: peName(data[at : at+8]), Address: address, MemorySize: memorySize,
			Alignment: 1, FileOffset: rawOffset,
			Permissions: pePermissions(characteristics), Data: payload,
		})
	}
	return len(image.Segments) > 1
}

func peName(data []byte) string {
	end := 0
	for end < len(data) && data[end] != 0 {
		end++
	}
	return string(data[:end])
}

func pePermissions(flags uint32) uint8 {
	var result uint8
	if flags&0x40000000 != 0 {
		result |= Read
	}
	if flags&0x80000000 != 0 {
		result |= Write
	}
	if flags&0x20000000 != 0 {
		result |= Execute
	}
	return result
}

func parseMachO(image *Image) bool {
	data := image.Native
	if len(data) < 32 || binary.LittleEndian.Uint32(data[:4]) != 0xfeedfacf {
		return false
	}
	commandCount := int(binary.LittleEndian.Uint32(data[16:20]))
	pos := 32
	entryFileOffset := ^uint64(0)
	base := ^uint64(0)
	for i := 0; i < commandCount; i++ {
		if pos+8 > len(data) {
			return false
		}
		command := binary.LittleEndian.Uint32(data[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		if size < 8 || pos+size < pos || pos+size > len(data) {
			return false
		}
		if command == 0x19 {
			if size < 72 {
				return false
			}
			address := binary.LittleEndian.Uint64(data[pos+24 : pos+32])
			memorySize := binary.LittleEndian.Uint64(data[pos+32 : pos+40])
			fileOffset := binary.LittleEndian.Uint64(data[pos+40 : pos+48])
			fileSize := binary.LittleEndian.Uint64(data[pos+48 : pos+56])
			if address != 0 || fileSize != 0 {
				payload, ok := byteRange(data, fileOffset, fileSize)
				if !ok || fileSize > memorySize {
					return false
				}
				if address < base {
					base = address
				}
				protection := binary.LittleEndian.Uint32(data[pos+60 : pos+64])
				image.Segments = append(image.Segments, Segment{
					Name: peName(data[pos+8 : pos+24]), Address: address,
					MemorySize: memorySize, Alignment: 4096, FileOffset: fileOffset,
					Permissions: machPermissions(protection), Data: payload,
				})
			}
		} else if command == 0x80000028 {
			if size < 24 {
				return false
			}
			entryFileOffset = binary.LittleEndian.Uint64(data[pos+8 : pos+16])
		}
		pos += size
	}
	if len(image.Segments) == 0 || entryFileOffset == ^uint64(0) {
		return false
	}
	entryAddress := uint64(0)
	entryFound := false
	for _, segment := range image.Segments {
		fileStart := segment.FileOffset
		if entryFileOffset >= fileStart && entryFileOffset-fileStart < uint64(len(segment.Data)) {
			entryAddress = segment.Address + entryFileOffset - fileStart
			entryFound = true
			break
		}
	}
	if !entryFound {
		return false
	}
	image.Base = base
	image.Entry = entryAddress - base
	for i := range image.Segments {
		image.Segments[i].Address -= base
	}
	return true
}

func machPermissions(protection uint32) uint8 {
	var result uint8
	if protection&1 != 0 {
		result |= Read
	}
	if protection&2 != 0 {
		result |= Write
	}
	if protection&4 != 0 {
		result |= Execute
	}
	return result
}

func parseWasm(image *Image) bool {
	if len(image.Native) < 8 || string(image.Native[:4]) != "\x00asm" {
		return false
	}
	image.Segments = []Segment{{
		Name: "module", MemorySize: uint64(len(image.Native)), Alignment: 1,
		Permissions: Read | Execute, Data: image.Native,
	}}
	return true
}

func tableEntry(offset uint64, size int, index int, limit int) (int, bool) {
	if size <= 0 || index < 0 || offset > uint64(limit) {
		return 0, false
	}
	at := offset + uint64(size)*uint64(index)
	if at > uint64(limit) || uint64(size) > uint64(limit)-at {
		return 0, false
	}
	return int(at), true
}

func byteRange(data []byte, offset uint64, size uint64) ([]byte, bool) {
	if size == 0 {
		return data[len(data):], true
	}
	if offset > uint64(len(data)) || size > uint64(len(data))-offset {
		return nil, false
	}
	return data[int(offset):int(offset+size)], true
}
