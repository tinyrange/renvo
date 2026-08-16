package main

// The hosted C/Go object path deliberately uses one compact, table-driven ELF
// writer. Section placement is represented as ranges over the emitter's code
// and virtual global address spaces, so common instruction lowering never
// needs to know an ELF section name or symbol binding.

type renvoObjectELFSection struct {
	name                          string
	typ, flags, alignment         int
	link, info, entrySize, offset int
	size                          int
	data                          []byte
	relocations                   []renvoObjectELFRelocation
}

type renvoObjectELFRelocation struct {
	offset, symbol, typ, addend int
}

type renvoObjectELFSymbol struct {
	name                      string
	info, visibility, section int
	value, size               int
}

type renvoObjectCodeRange struct {
	start, end, section, local int
	name                       string
	alignment                  int
}

type renvoObjectDataRange struct {
	start, end, section, local int
}

func renvoAsmImageKernelObjectAmd64(a *renvoAsm) []byte {
	renvoNonNil(a)
	sections := []renvoObjectELFSection{{}}
	ranges := renvoObjectCodeRanges(a)
	codeMap := make([]renvoObjectCodeRange, 0, len(ranges)*2+1)
	position := 0
	for i := 0; i < len(ranges); i++ {
		r := ranges[i]
		if r.start < position || r.start < 0 || r.end > len(a.code) || r.end <= r.start {
			return nil
		}
		if position < r.start {
			section := renvoObjectELFSectionIndex(&sections, ".text", 1, 6, 16)
			local := len(sections[section].data)
			sections[section].data = append(sections[section].data, a.code[position:r.start]...)
			sections[section].size = len(sections[section].data)
			codeMap = append(codeMap, renvoObjectCodeRange{start: position, end: r.start, section: section, local: local})
		}
		name := r.name
		if name == "" {
			name = ".text"
		}
		alignment := r.alignment
		if alignment < 1 {
			alignment = 16
		}
		section := renvoObjectELFSectionIndex(&sections, name, 1, 6, alignment)
		local := renvoAlignValue(len(sections[section].data), alignment)
		sections[section].data = renvoObjectUntil(sections[section].data, local)
		sections[section].data = append(sections[section].data, a.code[r.start:r.end]...)
		sections[section].size = len(sections[section].data)
		codeMap = append(codeMap, renvoObjectCodeRange{start: r.start, end: r.end, section: section, local: local})
		position = r.end
	}
	if position < len(a.code) || len(a.code) == 0 {
		section := renvoObjectELFSectionIndex(&sections, ".text", 1, 6, 16)
		local := len(sections[section].data)
		sections[section].data = append(sections[section].data, a.code[position:]...)
		sections[section].size = len(sections[section].data)
		if position < len(a.code) {
			codeMap = append(codeMap, renvoObjectCodeRange{start: position, end: len(a.code), section: section, local: local})
		}
	}

	dataMap := make([]renvoObjectDataRange, 0, len(a.objectData)+1)
	if len(a.data) != 0 {
		section := renvoObjectELFSectionIndex(&sections, ".rodata", 1, 2, 8)
		sections[section].data = append(sections[section].data, a.data...)
		sections[section].size = len(sections[section].data)
	}
	for i := 0; i < len(a.objectData); i++ {
		item := &a.objectData[i]
		if item.size <= 0 {
			continue
		}
		name := renvoObjectStoredText(a, item.sectionStart, item.sectionEnd)
		if name == "" {
			if item.initialized != 0 {
				name = ".data"
			} else {
				name = ".bss"
			}
		}
		nobits := item.initialized == 0 && (name == ".bss" || renvoObjectStringPrefix(name, ".bss."))
		typ, flags := 1, 3
		if nobits {
			typ = 8
		}
		if renvoObjectStringPrefix(name, ".rodata") {
			flags = 2
		}
		alignment := item.alignment
		if alignment < 1 {
			alignment = renvoObjectNaturalAlignment(item.size)
		}
		section := renvoObjectELFSectionIndex(&sections, name, typ, flags, alignment)
		local := renvoAlignValue(sections[section].size, alignment)
		storageSize := item.storageSize
		if storageSize < item.size {
			storageSize = item.size
		}
		sections[section].size = local + storageSize
		if !nobits {
			sections[section].data = renvoObjectUntil(sections[section].data, local+storageSize)
			for at := 0; at < item.size; at++ {
				sections[section].data[local+at] = byte(item.value >> (at * 8))
			}
		}
		dataMap = append(dataMap, renvoObjectDataRange{start: item.offset, end: item.offset + storageSize, section: section, local: local})
	}

	sectionSymbols := make([]int, len(sections))
	symbols := []renvoObjectELFSymbol{{}}
	for i := 1; i < len(sections); i++ {
		sectionSymbols[i] = len(symbols)
		symbols = append(symbols, renvoObjectELFSymbol{info: 3, section: i})
	}
	// ELF requires all local symbols to precede the first global symbol.
	for bindingPass := 0; bindingPass < 2; bindingPass++ {
		for i := 0; i < len(a.symbols); i++ {
			s := &a.symbols[i]
			binding := s.binding
			if binding == 0 && bindingPass != 0 || binding != 0 && bindingPass == 0 {
				continue
			}
			position := renvoAsmLabelPosition(a, s.label)
			section, value, ok := renvoObjectMapCode(codeMap, position)
			if !ok {
				return nil
			}
			size := 0
			if s.endLabel > 0 {
				endSection, endValue, endOK := renvoObjectMapCodeEnd(codeMap, renvoAsmLabelPosition(a, s.endLabel))
				if endOK && endSection == section {
					size = endValue - value
				}
			}
			symbols = append(symbols, renvoObjectELFSymbol{name: renvoObjectStoredText(a, s.nameStart, s.nameEnd),
				info: binding<<4 | 2, visibility: s.visibility, section: section, value: value, size: size})
		}
		for i := 0; i < len(a.objectData); i++ {
			d := &a.objectData[i]
			if d.size <= 0 || d.binding == 0 && bindingPass != 0 || d.binding != 0 && bindingPass == 0 {
				continue
			}
			section, value, ok := renvoObjectMapData(dataMap, d.offset)
			if !ok {
				return nil
			}
			symbols = append(symbols, renvoObjectELFSymbol{name: renvoObjectStoredText(a, d.nameStart, d.nameEnd),
				info: d.binding<<4 | 1, visibility: d.visibility, section: section, value: value, size: d.size})
		}
	}
	firstGlobal := len(symbols)
	for i := 1; i < len(symbols); i++ {
		if symbols[i].info>>4 != 0 {
			firstGlobal = i
			break
		}
	}
	importSymbols := make([]int, renvoKernelAmd64ExternalImportCount(a))
	for i := 0; i < len(importSymbols); i++ {
		importSymbols[i] = len(symbols)
		symbols = append(symbols, renvoObjectELFSymbol{name: renvoKernelAmd64ExternalImportName(a, i), info: 16})
	}
	for i := 0; i < len(a.objectData); i++ {
		d := &a.objectData[i]
		if d.size != 0 || d.targetEnd <= d.targetStart {
			continue
		}
		target := renvoObjectStoredText(a, d.targetStart, d.targetEnd)
		if target == "-" {
			continue
		}
		targetIndex := renvoObjectFindSymbol(symbols, target)
		if targetIndex < 0 {
			return nil
		}
		base := symbols[targetIndex]
		base.name = renvoObjectStoredText(a, d.nameStart, d.nameEnd)
		base.info = d.binding<<4 | base.info&15
		base.visibility = d.visibility
		symbols = append(symbols, base)
	}

	// Convert all label references into either same-section displacements or
	// section-symbol relocations. This is what makes function sections truthful:
	// moving a wrapper or implementation never leaves a prepatched PC behind.
	for i := 0; i+1 < len(a.relocs); i += 2 {
		at := int(renvo_runtime_UnsafeInt32At(a.relocs, i)) & 2147483647
		label := int(renvo_runtime_UnsafeInt32At(a.relocs, i+1)) & 2147483647
		targetPosition := renvoAsmLabelPosition(a, label)
		sourceSection, sourceOffset, sourceOK := renvoObjectMapCode(codeMap, at)
		targetSection, targetOffset, targetOK := renvoObjectMapCode(codeMap, targetPosition)
		if !sourceOK || !targetOK || sourceOffset+4 > len(sections[sourceSection].data) {
			return nil
		}
		addend := renvoGet32At(sections[sourceSection].data, sourceOffset)
		if sourceSection == targetSection {
			renvoPut32At(sections[sourceSection].data, sourceOffset, targetOffset+addend-(sourceOffset+4))
		} else {
			renvoPut32At(sections[sourceSection].data, sourceOffset, 0)
			sections[sourceSection].relocations = append(sections[sourceSection].relocations,
				renvoObjectELFRelocation{offset: sourceOffset, symbol: sectionSymbols[targetSection], typ: 2, addend: targetOffset + addend - 4})
		}
	}
	for i := 0; i+2 < len(a.absRelocs); i += 3 {
		at := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i)) & 2147483647
		addend := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+1)) & 2147483647
		kind := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+2)) & 2147483647
		sourceSection, sourceOffset, ok := renvoObjectMapCode(codeMap, at)
		if !ok || sourceOffset+4 > len(sections[sourceSection].data) {
			return nil
		}
		targetSymbol, targetOffset, relocationType := 0, 0, 2
		if kind == renvoKernelAmd64RelocationImport {
			if addend < 0 || addend >= len(importSymbols) {
				return nil
			}
			targetSymbol = importSymbols[addend]
			relocationType = 4
		} else if kind == renvoKernelAmd64RelocationAbsoluteData {
			section := renvoObjectFindSection(sections, ".rodata")
			if section < 0 {
				return nil
			}
			targetSymbol, targetOffset = sectionSymbols[section], addend
		} else if kind == renvoKernelAmd64RelocationAbsoluteBSS {
			section, value, found := renvoObjectMapData(dataMap, addend)
			if !found {
				return nil
			}
			targetSymbol, targetOffset = sectionSymbols[section], value
		} else if kind == renvoKernelAmd64RelocationAbsoluteBSSEnd {
			section := renvoObjectFindSection(sections, ".bss")
			if section < 0 {
				return nil
			}
			targetSymbol, targetOffset = sectionSymbols[section], sections[section].size
		} else {
			return nil
		}
		renvoPut32At(sections[sourceSection].data, sourceOffset, 0)
		sections[sourceSection].relocations = append(sections[sourceSection].relocations,
			renvoObjectELFRelocation{offset: sourceOffset, symbol: targetSymbol, typ: relocationType, addend: targetOffset - 4})
	}
	for i := 0; i < len(a.objectDataRelocs); i++ {
		r := &a.objectDataRelocs[i]
		sourceSection, sourceOffset, found := renvoObjectMapData(dataMap, r.offset)
		if !found {
			return nil
		}
		target := renvoObjectStoredText(a, r.targetStart, r.targetEnd)
		targetSymbol := renvoObjectFindSymbol(symbols, target)
		if targetSymbol < 0 {
			return nil
		}
		sections[sourceSection].relocations = append(sections[sourceSection].relocations,
			renvoObjectELFRelocation{offset: sourceOffset, symbol: targetSymbol, typ: r.typ, addend: r.addend})
	}
	for i := 1; i < len(sections); i++ {
		if !renvoObjectStringPrefix(sections[i].name, ".gnu.linkonce.") {
			continue
		}
		signature := -1
		for j := 1; j < len(symbols); j++ {
			if symbols[j].section == i && symbols[j].name != "" {
				signature = j
				break
			}
		}
		if signature < 0 {
			return nil
		}
		sections[i].flags |= 512
		groupData := renvoElfAmd64Append32(nil, 1)
		groupData = renvoElfAmd64Append32(groupData, i)
		sections = append(sections, renvoObjectELFSection{name: ".group", typ: 17, alignment: 4,
			info: signature, entrySize: 4, data: groupData, size: len(groupData)})
	}

	contentCount := len(sections)
	for i := 1; i < contentCount; i++ {
		if len(sections[i].relocations) == 0 {
			continue
		}
		name := ".rela" + sections[i].name
		relocationSection := renvoObjectELFSection{name: name, typ: 4, flags: 64, alignment: 8, info: i, entrySize: 24}
		for j := 0; j < len(sections[i].relocations); j++ {
			r := sections[i].relocations[j]
			relocationSection.data = renvoElfAmd64AppendRelocation(relocationSection.data, r.offset, r.symbol, r.typ, r.addend)
		}
		relocationSection.size = len(relocationSection.data)
		sections = append(sections, relocationSection)
	}

	strtab := []byte{0}
	symtab := make([]byte, 0, len(symbols)*24)
	for i := 0; i < len(symbols); i++ {
		nameOffset := 0
		if symbols[i].name != "" {
			nameOffset = len(strtab)
			strtab = renvoObjectAppendString(strtab, symbols[i].name)
		}
		start := len(symtab)
		symtab = renvoElfAmd64AppendSymbol(symtab, nameOffset, symbols[i].info, symbols[i].section, symbols[i].value, symbols[i].size)
		symtab[start+5] = byte(symbols[i].visibility)
	}
	symtabIndex := len(sections)
	sections = append(sections, renvoObjectELFSection{name: ".symtab", typ: 2, alignment: 8, info: firstGlobal, entrySize: 24, data: symtab, size: len(symtab)})
	strtabIndex := len(sections)
	sections = append(sections, renvoObjectELFSection{name: ".strtab", typ: 3, alignment: 1, data: strtab, size: len(strtab)})
	sections[symtabIndex].link = strtabIndex
	for i := 1; i < symtabIndex; i++ {
		if sections[i].typ == 4 || sections[i].typ == 17 {
			sections[i].link = symtabIndex
		}
	}
	sections = append(sections, renvoObjectELFSection{name: ".note.GNU-stack", typ: 1, alignment: 1})
	shstrtabIndex := len(sections)
	sections = append(sections, renvoObjectELFSection{name: ".shstrtab", typ: 3, alignment: 1})
	sectionNameOffsets := make([]int, len(sections))
	shstrtab := []byte{0}
	for i := 1; i < len(sections); i++ {
		sectionNameOffsets[i] = len(shstrtab)
		shstrtab = renvoObjectAppendString(shstrtab, sections[i].name)
	}
	sections[shstrtabIndex].data = shstrtab
	sections[shstrtabIndex].size = len(shstrtab)

	image := make([]byte, 64)
	for i := 1; i < len(sections); i++ {
		alignment := sections[i].alignment
		if alignment < 1 {
			alignment = 1
		}
		offset := renvoAlignValue(len(image), alignment)
		image = renvoObjectUntil(image, offset)
		sections[i].offset = offset
		if sections[i].typ != 8 {
			image = append(image, sections[i].data...)
		}
	}
	sectionOffset := renvoAlignValue(len(image), 8)
	image = renvoObjectUntil(image, sectionOffset)
	for i := 0; i < len(sections); i++ {
		s := sections[i]
		image = renvoElfAmd64AppendSection(image, sectionNameOffsets[i], s.typ, s.flags, s.offset, s.size,
			s.link, s.info, s.alignment, s.entrySize)
	}
	header := renvoElfAmd64AppendHeader(nil, sectionOffset, len(sections), shstrtabIndex)
	for i := 0; i < len(header); i++ {
		image[i] = header[i]
	}
	return image
}

func renvoObjectCodeRanges(a *renvoAsm) []renvoObjectCodeRange {
	var ranges []renvoObjectCodeRange
	for i := 0; i < len(a.symbols); i++ {
		s := &a.symbols[i]
		if s.endLabel <= 0 || s.sectionEnd <= s.sectionStart && s.alignment <= 0 {
			continue
		}
		ranges = append(ranges, renvoObjectCodeRange{start: renvoAsmLabelPosition(a, s.label),
			end: renvoAsmLabelPosition(a, s.endLabel), name: renvoObjectStoredText(a, s.sectionStart, s.sectionEnd), alignment: s.alignment})
	}
	for i := 0; i < len(a.objectFunctions); i++ {
		f := &a.objectFunctions[i]
		ranges = append(ranges, renvoObjectCodeRange{start: renvoAsmLabelPosition(a, f.label), end: f.end,
			name: renvoObjectStoredText(a, f.sectionStart, f.sectionEnd), alignment: f.alignment})
	}
	for i := 1; i < len(ranges); i++ {
		value := ranges[i]
		j := i
		for j > 0 && ranges[j-1].start > value.start {
			ranges[j] = ranges[j-1]
			j--
		}
		ranges[j] = value
	}
	return ranges
}

func renvoObjectELFSectionIndex(sections *[]renvoObjectELFSection, name string, typ int, flags int, alignment int) int {
	for i := 1; i < len(*sections); i++ {
		if (*sections)[i].name == name {
			if alignment > (*sections)[i].alignment {
				(*sections)[i].alignment = alignment
			}
			return i
		}
	}
	*sections = append(*sections, renvoObjectELFSection{name: name, typ: typ, flags: flags, alignment: alignment})
	return len(*sections) - 1
}

func renvoObjectMapCode(ranges []renvoObjectCodeRange, position int) (int, int, bool) {
	for i := 0; i < len(ranges); i++ {
		if position >= ranges[i].start && position < ranges[i].end {
			return ranges[i].section, ranges[i].local + position - ranges[i].start, true
		}
	}
	return 0, 0, false
}

func renvoObjectMapCodeEnd(ranges []renvoObjectCodeRange, position int) (int, int, bool) {
	for i := 0; i < len(ranges); i++ {
		if position >= ranges[i].start && position <= ranges[i].end {
			return ranges[i].section, ranges[i].local + position - ranges[i].start, true
		}
	}
	return 0, 0, false
}

func renvoObjectMapData(ranges []renvoObjectDataRange, position int) (int, int, bool) {
	for i := 0; i < len(ranges); i++ {
		if position >= ranges[i].start && position < ranges[i].end {
			return ranges[i].section, ranges[i].local + position - ranges[i].start, true
		}
	}
	return 0, 0, false
}

func renvoObjectStoredText(a *renvoAsm, start int, end int) string {
	if start < 0 || end <= start || end > len(a.symbolName) {
		return ""
	}
	return string(a.symbolName[start:end])
}

func renvoObjectFindSection(sections []renvoObjectELFSection, name string) int {
	for i := 1; i < len(sections); i++ {
		if sections[i].name == name {
			return i
		}
	}
	return -1
}

func renvoObjectFindSymbol(symbols []renvoObjectELFSymbol, name string) int {
	for i := 1; i < len(symbols); i++ {
		if symbols[i].name == name {
			return i
		}
	}
	return -1
}

func renvoObjectUntil(out []byte, size int) []byte {
	for len(out) < size {
		out = append(out, 0)
	}
	return out
}

func renvoObjectAppendString(out []byte, value string) []byte {
	for i := 0; i < len(value); i++ {
		out = append(out, value[i])
	}
	return append(out, 0)
}

func renvoObjectNaturalAlignment(size int) int {
	alignment := 1
	for alignment < size && alignment < 8 {
		alignment *= 2
	}
	return alignment
}

func renvoObjectStringPrefix(value string, prefix string) bool {
	if len(value) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if value[i] != prefix[i] {
			return false
		}
	}
	return true
}
