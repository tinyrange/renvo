// Package elflink implements the small static ELF linker used by Renvo's
// Linux/amd64 C toolchain. It intentionally accepts only ELF64 relocatable
// objects and the relocation forms emitted by Renvo.
package elflink

const (
	shnUndefined = 0
	shnAbsolute  = 0xfff1
	shtProgbits  = 1
	shtSymtab    = 2
	shtRela      = 4
	shtNobits    = 8
	shfWrite     = 1
	shfAlloc     = 2
	shfExec      = 4
	stbGlobal    = 1
	stbWeak      = 2
	rX8664_64    = 1
	rX8664PC32   = 2
	rX8664PLT32  = 4
	rX866432     = 10
	rX866432S    = 11
)

type Input struct {
	Name string
	Data []byte
}

type Error struct {
	Input   string
	Message string
}

type section struct {
	typ       uint32
	flags     uint64
	offset    uint64
	size      uint64
	link      uint32
	info      uint32
	align     uint64
	entrySize uint64
	address   uint64
	fileAt    uint64
}

type symbol struct {
	name    string
	info    byte
	section uint16
	value   uint64
	address uint64
}

type object struct {
	input    Input
	sections []section
	symbols  [][]symbol
}

type definition struct {
	name    string
	address uint64
}

const imageBase uint64 = 0x400000
const pageSize uint64 = 0x1000

var startup = []byte{
	0x48, 0x8b, 0x3c, 0x24, // mov rdi,[rsp]       argc
	0x48, 0x8d, 0x74, 0x24, 0x08, // lea rsi,[rsp+8] argv
	0x48, 0x83, 0xe4, 0xf0, // and rsp,-16
	0xe8, 0, 0, 0, 0, // call main
	0x89, 0xc7, // mov edi,eax
	0xb8, 0x3c, 0, 0, 0, // mov eax,60
	0x0f, 0x05, // syscall
}

// Link creates a statically linked ET_EXEC image with _start calling main.
func Link(inputs []Input) ([]byte, Error) {
	if len(inputs) == 0 {
		return nil, Error{Message: "no object files were provided"}
	}
	objects := make([]object, len(inputs))
	for i := 0; i < len(inputs); i++ {
		parsed, err := parseObject(inputs[i])
		if err.Message != "" {
			return nil, err
		}
		objects[i] = parsed
	}
	headerSize := uint64(64 + 2*56)
	textFileAt := pageSize
	textAddress := imageBase + textFileAt
	textCursor := textFileAt + uint64(len(startup))
	// Executable and read-only allocated sections share the RX segment.
	for pass := 0; pass < 2; pass++ {
		for oi := 0; oi < len(objects); oi++ {
			for si := 1; si < len(objects[oi].sections); si++ {
				section := &objects[oi].sections[si]
				readOnly := section.flags&shfAlloc != 0 && section.flags&shfWrite == 0 && section.typ != shtNobits
				if !readOnly || (pass == 0) != (section.flags&shfExec != 0) {
					continue
				}
				textCursor = align(textCursor, section.align)
				section.fileAt = textCursor
				section.address = imageBase + textCursor
				textCursor += section.size
			}
		}
	}
	dataFileAt := align(textCursor, pageSize)
	dataCursor := dataFileAt
	for oi := 0; oi < len(objects); oi++ {
		for si := 1; si < len(objects[oi].sections); si++ {
			section := &objects[oi].sections[si]
			if section.flags&shfAlloc == 0 || section.flags&shfWrite == 0 || section.typ == shtNobits {
				continue
			}
			dataCursor = align(dataCursor, section.align)
			section.fileAt = dataCursor
			section.address = imageBase + dataCursor
			dataCursor += section.size
		}
	}
	fileSize := dataCursor
	bssCursor := dataCursor
	for oi := 0; oi < len(objects); oi++ {
		for si := 1; si < len(objects[oi].sections); si++ {
			section := &objects[oi].sections[si]
			if section.flags&shfAlloc == 0 || section.typ != shtNobits {
				continue
			}
			bssCursor = align(bssCursor, section.align)
			section.fileAt = bssCursor
			section.address = imageBase + bssCursor
			bssCursor += section.size
		}
	}
	definitions := make([]definition, 0, 32)
	for oi := 0; oi < len(objects); oi++ {
		for ti := 0; ti < len(objects[oi].symbols); ti++ {
			for si := 0; si < len(objects[oi].symbols[ti]); si++ {
				sym := &objects[oi].symbols[ti][si]
				if sym.section != shnUndefined {
					if sym.section == shnAbsolute {
						sym.address = sym.value
					} else if int(sym.section) < len(objects[oi].sections) {
						sym.address = objects[oi].sections[sym.section].address + sym.value
					} else {
						return nil, Error{Input: objects[oi].input.Name, Message: "symbol uses an unsupported section index"}
					}
				}
				binding := sym.info >> 4
				if sym.name != "" && sym.section != shnUndefined && (binding == stbGlobal || binding == stbWeak) {
					for j := 0; j < len(definitions); j++ {
						if definitions[j].name == sym.name && binding != stbWeak {
							return nil, Error{Input: objects[oi].input.Name, Message: "duplicate symbol: " + sym.name}
						}
					}
					if findDefinition(definitions, sym.name) == 0 {
						definitions = append(definitions, definition{name: sym.name, address: sym.address})
					}
				}
			}
		}
	}
	mainAddress := findDefinition(definitions, "main")
	if mainAddress == 0 {
		return nil, Error{Message: "entry function main is undefined"}
	}
	image := make([]byte, fileSize)
	copy(image[textFileAt:], startup)
	mainDisplacement := int64(mainAddress) - int64(textAddress+18)
	if !fitsSigned32(mainDisplacement) {
		return nil, Error{Message: "main is outside the startup call range"}
	}
	put32(image[textFileAt+14:], uint32(int32(mainDisplacement)))
	for oi := 0; oi < len(objects); oi++ {
		for si := 1; si < len(objects[oi].sections); si++ {
			section := objects[oi].sections[si]
			if section.flags&shfAlloc == 0 || section.typ == shtNobits || section.size == 0 {
				continue
			}
			if section.offset+section.size > uint64(len(objects[oi].input.Data)) {
				return nil, Error{Input: objects[oi].input.Name, Message: "section data is truncated"}
			}
			copy(image[section.fileAt:section.fileAt+section.size], objects[oi].input.Data[section.offset:section.offset+section.size])
		}
	}
	for oi := 0; oi < len(objects); oi++ {
		if err := relocate(&objects[oi], definitions, image); err.Message != "" {
			return nil, err
		}
	}
	writeHeader(image, headerSize, textFileAt, textCursor, dataFileAt, fileSize, bssCursor)
	return image, Error{}
}

func parseObject(input Input) (object, Error) {
	data := input.Data
	if len(data) < 64 || string(data[:4]) != "\x7fELF" || data[4] != 2 || data[5] != 1 || u16(data, 16) != 1 || u16(data, 18) != 62 {
		return object{}, Error{Input: input.Name, Message: "not a Linux/amd64 ELF relocatable object"}
	}
	sectionAt := u64(data, 40)
	sectionSize, sectionCount := uint64(u16(data, 58)), uint64(u16(data, 60))
	if sectionSize < 64 || sectionAt+sectionSize*sectionCount > uint64(len(data)) {
		return object{}, Error{Input: input.Name, Message: "section table is truncated"}
	}
	result := object{input: input, sections: make([]section, sectionCount), symbols: make([][]symbol, sectionCount)}
	for i := uint64(0); i < sectionCount; i++ {
		at := sectionAt + i*sectionSize
		result.sections[i] = section{typ: u32(data, at+4), flags: u64(data, at+8), offset: u64(data, at+24), size: u64(data, at+32), link: u32(data, at+40), info: u32(data, at+44), align: u64(data, at+48), entrySize: u64(data, at+56)}
		if result.sections[i].align == 0 {
			result.sections[i].align = 1
		}
	}
	for i := 1; i < len(result.sections); i++ {
		section := result.sections[i]
		if section.typ != shtSymtab || section.entrySize < 24 || int(section.link) >= len(result.sections) {
			continue
		}
		strings := result.sections[section.link]
		if section.offset+section.size > uint64(len(data)) || strings.offset+strings.size > uint64(len(data)) {
			return object{}, Error{Input: input.Name, Message: "symbol table is truncated"}
		}
		count := section.size / section.entrySize
		result.symbols[i] = make([]symbol, count)
		for j := uint64(0); j < count; j++ {
			at := section.offset + j*section.entrySize
			nameAt := uint64(u32(data, at))
			name := ""
			if nameAt < strings.size {
				name = cString(data[strings.offset+nameAt : strings.offset+strings.size])
			}
			result.symbols[i][j] = symbol{name: name, info: data[at+4], section: u16(data, at+6), value: u64(data, at+8)}
		}
	}
	return result, Error{}
}

func relocate(object *object, definitions []definition, image []byte) Error {
	for i := 1; i < len(object.sections); i++ {
		relocations := object.sections[i]
		if relocations.typ != shtRela || relocations.entrySize < 24 || int(relocations.info) >= len(object.sections) || int(relocations.link) >= len(object.symbols) {
			continue
		}
		target := object.sections[relocations.info]
		symbols := object.symbols[relocations.link]
		if relocations.offset+relocations.size > uint64(len(object.input.Data)) {
			return Error{Input: object.input.Name, Message: "relocation table is truncated"}
		}
		for at := relocations.offset; at < relocations.offset+relocations.size; at += relocations.entrySize {
			offset, info, addend := u64(object.input.Data, at), u64(object.input.Data, at+8), int64(u64(object.input.Data, at+16))
			symbolIndex, kind := info>>32, uint32(info)
			if symbolIndex >= uint64(len(symbols)) || offset >= target.size {
				return Error{Input: object.input.Name, Message: "relocation references data outside its object"}
			}
			sym := symbols[symbolIndex]
			address := sym.address
			if sym.section == shnUndefined {
				address = findDefinition(definitions, sym.name)
				if address == 0 && sym.info>>4 != stbWeak {
					return Error{Input: object.input.Name, Message: "undefined symbol: " + sym.name}
				}
			}
			place := target.address + offset
			writeAt := target.fileAt + offset
			switch kind {
			case rX8664_64:
				if writeAt+8 > uint64(len(image)) {
					return Error{Input: object.input.Name, Message: "64-bit relocation is outside file data"}
				}
				put64(image[writeAt:], uint64(int64(address)+addend))
			case rX8664PC32, rX8664PLT32:
				value := int64(address) + addend - int64(place)
				if writeAt+4 > uint64(len(image)) || !fitsSigned32(value) {
					return Error{Input: object.input.Name, Message: "PC-relative relocation is out of range"}
				}
				put32(image[writeAt:], uint32(int32(value)))
			case rX866432, rX866432S:
				value := int64(address) + addend
				if writeAt+4 > uint64(len(image)) || kind == rX866432S && !fitsSigned32(value) || kind == rX866432 && (value < 0 || uint64(value) > 0xffffffff) {
					return Error{Input: object.input.Name, Message: "32-bit relocation is out of range"}
				}
				put32(image[writeAt:], uint32(value))
			default:
				return Error{Input: object.input.Name, Message: "unsupported x86-64 relocation type " + decimal(uint64(kind))}
			}
		}
	}
	return Error{}
}

func writeHeader(image []byte, headerSize, textAt, textEnd, dataAt, fileSize, memoryEnd uint64) {
	copy(image[:16], []byte{'\x7f', 'E', 'L', 'F', 2, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	put16(image[16:], 2)
	put16(image[18:], 62)
	put32(image[20:], 1)
	put64(image[24:], imageBase+textAt)
	put64(image[32:], 64)
	put16(image[52:], 64)
	put16(image[54:], 56)
	put16(image[56:], 2)
	writeProgramHeader(image[64:], 1, 5, 0, imageBase, textEnd, textEnd, pageSize)
	writeProgramHeader(image[120:], 1, 6, dataAt, imageBase+dataAt, fileSize-dataAt, memoryEnd-dataAt, pageSize)
	_ = headerSize
}

func writeProgramHeader(out []byte, typ, flags uint32, offset, address, fileSize, memorySize, alignment uint64) {
	put32(out, typ)
	put32(out[4:], flags)
	put64(out[8:], offset)
	put64(out[16:], address)
	put64(out[24:], address)
	put64(out[32:], fileSize)
	put64(out[40:], memorySize)
	put64(out[48:], alignment)
}

func findDefinition(values []definition, name string) uint64 {
	for i := 0; i < len(values); i++ {
		if values[i].name == name {
			return values[i].address
		}
	}
	return 0
}
func align(value, alignment uint64) uint64 {
	if alignment <= 1 {
		return value
	}
	return (value + alignment - 1) &^ (alignment - 1)
}
func fitsSigned32(value int64) bool     { return value >= -2147483648 && value <= 2147483647 }
func u16(data []byte, at uint64) uint16 { return uint16(data[at]) | uint16(data[at+1])<<8 }
func u32(data []byte, at uint64) uint32 { return uint32(u16(data, at)) | uint32(u16(data, at+2))<<16 }
func u64(data []byte, at uint64) uint64 { return uint64(u32(data, at)) | uint64(u32(data, at+4))<<32 }
func put16(out []byte, value uint16)    { out[0] = byte(value); out[1] = byte(value >> 8) }
func put32(out []byte, value uint32)    { put16(out, uint16(value)); put16(out[2:], uint16(value>>16)) }
func put64(out []byte, value uint64)    { put32(out, uint32(value)); put32(out[4:], uint32(value>>32)) }
func cString(data []byte) string {
	end := 0
	for end < len(data) && data[end] != 0 {
		end++
	}
	return string(data[:end])
}
func decimal(value uint64) string {
	if value == 0 {
		return "0"
	}
	out := make([]byte, 0, 20)
	for value > 0 {
		out = append(out, byte('0'+value%10))
		value /= 10
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}
