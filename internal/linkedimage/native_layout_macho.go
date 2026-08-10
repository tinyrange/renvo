//go:build !renvo || darwin

package linkedimage

type imageMachSegment struct {
	address     int
	memorySize  int
	fileOffset  int
	fileSize    int
	permissions int
}

// DarwinLayout decodes the subset of Mach-O load and dyld bind commands
// emitted by Renvo's arm64 backend.
func DarwinLayout(native []byte) (entry int, memorySize int, segments []NativeSegment, imports []NativeImport, libraries []string, ok bool) {
	if len(native) < 32 || native[0] != 0xcf || native[1] != 0xfa ||
		native[2] != 0xed || native[3] != 0xfe {
		return 0, 0, nil, nil, nil, false
	}
	commandCount := imageRead32(native, 16)
	commandBytes := imageRead32(native, 20)
	if commandCount < 1 || commandBytes < 8 || 32+commandBytes > len(native) {
		return 0, 0, nil, nil, nil, false
	}
	allSegments := make([]imageMachSegment, 0, 4)
	base := -1
	entryFileOffset := -1
	bindOffset := 0
	bindSize := 0
	at := 32
	for i := 0; i < commandCount; i++ {
		if at+8 > len(native) {
			return 0, 0, nil, nil, nil, false
		}
		command := imageRead32(native, at)
		size := imageRead32(native, at+4)
		if size < 8 || at+size < at || at+size > len(native) {
			return 0, 0, nil, nil, nil, false
		}
		if command == 0x19 {
			if size < 72 {
				return 0, 0, nil, nil, nil, false
			}
			address, addressOK := imageRead64Int(native, at+24)
			segmentMemory, memoryOK := imageRead64Int(native, at+32)
			fileOffset, fileOK := imageRead64Int(native, at+40)
			fileSize, sizeOK := imageRead64Int(native, at+48)
			if !addressOK || !memoryOK || !fileOK || !sizeOK ||
				address < 0 || segmentMemory < 0 || fileOffset < 0 ||
				fileSize < 0 || fileSize > segmentMemory ||
				fileOffset+fileSize < fileOffset || fileOffset+fileSize > len(native) {
				return 0, 0, nil, nil, nil, false
			}
			allSegments = append(allSegments, imageMachSegment{
				address: address, memorySize: segmentMemory,
				fileOffset: fileOffset, fileSize: fileSize,
				permissions: imageMachNativePermissions(imageRead32(native, at+60)),
			})
			if address != 0 || fileSize != 0 {
				if base < 0 || address < base {
					base = address
				}
			}
		} else if command&0x7fffffff == 0x28 && native[at+3]&0x80 != 0 {
			var entryOK bool
			entryFileOffset, entryOK = imageRead64Int(native, at+8)
			if size < 24 || !entryOK {
				return 0, 0, nil, nil, nil, false
			}
		} else if command&0x7fffffff == 0x22 && native[at+3]&0x80 != 0 {
			if size < 48 {
				return 0, 0, nil, nil, nil, false
			}
			bindOffset = imageRead32(native, at+16)
			bindSize = imageRead32(native, at+20)
		} else if command == 0x0c || command == 0x18 || command == 0x1f {
			if size < 24 {
				return 0, 0, nil, nil, nil, false
			}
			pathOffset := imageRead32(native, at+8)
			if pathOffset < 0 || pathOffset >= size {
				return 0, 0, nil, nil, nil, false
			}
			library, libraryOK := imageStringZ(native, at+pathOffset)
			if !libraryOK || at+pathOffset+len(library) >= at+size {
				return 0, 0, nil, nil, nil, false
			}
			libraries = append(libraries, library)
		}
		at += size
	}
	if base < 0 || entryFileOffset < 0 || len(allSegments) == 0 {
		return 0, 0, nil, nil, nil, false
	}
	entryAddress := -1
	segments = make([]NativeSegment, 0, len(allSegments))
	for i := 0; i < len(allSegments); i++ {
		segment := allSegments[i]
		if segment.address == 0 && segment.fileSize == 0 {
			continue
		}
		address := segment.address - base
		end := address + segment.memorySize
		if address < 0 || end < address {
			return 0, 0, nil, nil, nil, false
		}
		if end > memorySize {
			memorySize = end
		}
		if entryFileOffset >= segment.fileOffset &&
			entryFileOffset-segment.fileOffset < segment.fileSize {
			entryAddress = address + entryFileOffset - segment.fileOffset
		}
		segments = append(segments, NativeSegment{
			FileOffset: segment.fileOffset, Address: address,
			FileSize: segment.fileSize, MemorySize: segment.memorySize,
			Permissions: segment.permissions,
		})
	}
	if entryAddress < 0 || entryAddress >= memorySize {
		return 0, 0, nil, nil, nil, false
	}
	if bindSize > 0 {
		if bindOffset < 0 || bindSize < 0 || bindOffset+bindSize < bindOffset ||
			bindOffset+bindSize > len(native) {
			return 0, 0, nil, nil, nil, false
		}
		var bindOK bool
		imports, bindOK = imageDarwinImports(native[bindOffset:bindOffset+bindSize], allSegments, base, libraries)
		if !bindOK {
			return 0, 0, nil, nil, nil, false
		}
		for i := 0; i < len(imports); i++ {
			if imports[i].Address < 0 || imports[i].Address+8 < imports[i].Address ||
				imports[i].Address+8 > memorySize {
				return 0, 0, nil, nil, nil, false
			}
		}
	}
	return entryAddress, memorySize, segments, imports, libraries, true
}

// Mach VM protections encode read=1, write=2, execute=4, while NativeSegment
// deliberately follows the ELF convention read=4, write=2, execute=1.
func imageMachNativePermissions(protection int) int {
	permissions := 0
	if protection&1 != 0 {
		permissions |= 4
	}
	if protection&2 != 0 {
		permissions |= 2
	}
	if protection&4 != 0 {
		permissions |= 1
	}
	return permissions
}

func imageDarwinImports(bind []byte, segments []imageMachSegment, base int, libraries []string) ([]NativeImport, bool) {
	var imports []NativeImport
	ordinal := 0
	symbol := ""
	address := -1
	for at := 0; at < len(bind); {
		opcode := int(bind[at])
		at++
		operation := opcode & 0xf0
		immediate := opcode & 15
		if operation == 0 {
			return imports, true
		}
		if operation == 0x10 {
			ordinal = immediate
			continue
		}
		if operation == 0x40 {
			var symbolOK bool
			symbol, symbolOK = imageStringZ(bind, at)
			if !symbolOK {
				return nil, false
			}
			at += len(symbol) + 1
			continue
		}
		if operation == 0x50 {
			continue
		}
		if operation == 0x70 {
			offset, next, offsetOK := imageReadULEB(bind, at)
			if !offsetOK || immediate >= len(segments) {
				return nil, false
			}
			at = next
			address = segments[immediate].address - base + offset
			continue
		}
		if operation == 0x80 {
			offset, next, offsetOK := imageReadULEB(bind, at)
			if !offsetOK || address < 0 {
				return nil, false
			}
			at = next
			address += offset
			continue
		}
		if operation == 0x90 {
			if address < 0 || symbol == "" || ordinal <= 0 || ordinal > len(libraries) {
				return nil, false
			}
			imports = append(imports, NativeImport{
				Library: libraries[ordinal-1], Name: symbol, Address: address,
			})
			address += 8
			continue
		}
		return nil, false
	}
	return imports, true
}

func imageReadULEB(src []byte, at int) (int, int, bool) {
	value := 0
	shift := 0
	for at < len(src) && shift < 28 {
		ch := int(src[at])
		at++
		value |= (ch & 127) << shift
		if ch&128 == 0 {
			return value, at, value >= 0
		}
		shift += 7
	}
	return 0, 0, false
}

func imageRead64Int(src []byte, at int) (int, bool) {
	if at < 0 || at+8 > len(src) {
		return 0, false
	}
	low := imageRead32(src, at)
	high := imageRead32(src, at+4)
	factor := 65536
	factor = factor * 65536
	if high != 0 && factor == 0 {
		return 0, false
	}
	value := low + high*factor
	return value, low >= 0 && high >= 0 && value >= 0
}
