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
	priority                   int
}

type renvoObjectDataRange struct {
	start, end, section, local int
}

func renvoObjectImageFail(reason string) []byte {
	renvoPrintErr("renvo: object image failed: ")
	renvoPrintErr(reason)
	renvoPrintErr("\n")
	return nil
}

func renvoAsmImageKernelObjectAmd64(a *renvoAsm) []byte {
	renvoNonNil(a)
	sections := []renvoObjectELFSection{{}}
	ranges := renvoObjectCodeRanges(a)
	boundaries := make([]int, 0, len(ranges)*2+2)
	boundaries = append(boundaries, 0, len(a.code))
	for i := 0; i < len(ranges); i++ {
		r := &ranges[i]
		if r.start < 0 || r.end > len(a.code) || r.end <= r.start {
			return renvoObjectImageFail("invalid code range")
		}
		boundaries = append(boundaries, r.start, r.end)
	}
	for i := 1; i < len(boundaries); i++ {
		value := boundaries[i]
		j := i
		for j > 0 && boundaries[j-1] > value {
			boundaries[j] = boundaries[j-1]
			j--
		}
		boundaries[j] = value
	}
	unique := boundaries[:0]
	for i := 0; i < len(boundaries); i++ {
		if len(unique) == 0 || unique[len(unique)-1] != boundaries[i] {
			unique = append(unique, boundaries[i])
		}
	}
	codeMap := make([]renvoObjectCodeRange, 0, len(unique))
	for i := 0; i+1 < len(unique); i++ {
		start, end := unique[i], unique[i+1]
		if end <= start {
			continue
		}
		chosen := -1
		for j := 0; j < len(ranges); j++ {
			r := &ranges[j]
			if r.start <= start && r.end >= end &&
				(chosen < 0 || r.priority > ranges[chosen].priority ||
					r.priority == ranges[chosen].priority && r.end-r.start < ranges[chosen].end-ranges[chosen].start) {
				chosen = j
			}
		}
		name := ".text"
		alignment := 1
		priority := 0
		if chosen >= 0 {
			r := &ranges[chosen]
			name = r.name
			priority = r.priority
			if start == r.start {
				alignment = r.alignment
				if alignment < 1 {
					alignment = 16
				}
			}
		}
		if name == "" {
			name = ".text"
		}
		section := renvoObjectELFSectionIndex(&sections, name, 1, 6, alignment)
		local := renvoAlignValue(len(sections[section].data), alignment)
		sections[section].data = renvoObjectCodeUntil(sections[section].data, local)
		sections[section].data = append(sections[section].data, a.code[start:end]...)
		sections[section].size = len(sections[section].data)
		codeMap = append(codeMap, renvoObjectCodeRange{start: start, end: end, section: section, local: local, priority: priority})
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
			target := renvoObjectStoredText(a, item.targetStart, item.targetEnd)
			if target != "" && target != "-" {
				continue
			}
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
		if renvoObjectStringPrefix(name, ".initcall") {
			// Linux emits initcall entries through file-scope assembly with an
			// alloc-only ("a") section declaration.
			flags = 2
		}
		if item.kind == renvoObjectDeclStaticCall {
			flags = 6
		}
		if name == "__ex_table" || renvoObjectStringPrefix(name, "runtime_ptr_") || renvoObjectStringPrefix(name, "runtime_shift_") {
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
			valueSize := item.valueEnd - item.valueStart
			if valueSize > 0 {
				if item.valueStart < 0 || item.valueEnd > len(a.objectDataValues) || valueSize > storageSize {
					return renvoObjectImageFail("invalid data value range")
				}
				copy(sections[section].data[local:local+valueSize], a.objectDataValues[item.valueStart:item.valueEnd])
			} else {
				for at := 0; at < item.size; at++ {
					sections[section].data[local+at] = byte(item.value >> (at * 8))
				}
			}
		}
		dataMap = append(dataMap, renvoObjectDataRange{start: item.offset, end: item.offset + storageSize, section: section, local: local})
	}
	declaredRanges := len(dataMap)
	mappedEnd := 0
	for i := 0; i < declaredRanges; i++ {
		r := dataMap[i]
		if r.start < mappedEnd {
			return renvoObjectImageFail("overlapping or unordered data ranges")
		}
		if r.start > mappedEnd {
			section := renvoObjectELFSectionIndex(&sections, ".bss", 8, 3, 8)
			local := renvoAlignValue(sections[section].size, 8)
			sections[section].size = local + r.start - mappedEnd
			dataMap = append(dataMap, renvoObjectDataRange{start: mappedEnd, end: r.start, section: section, local: local})
		}
		mappedEnd = r.end
	}
	if a.bssSize > mappedEnd {
		section := renvoObjectELFSectionIndex(&sections, ".bss", 8, 3, 8)
		local := renvoAlignValue(sections[section].size, 8)
		sections[section].size = local + a.bssSize - mappedEnd
		dataMap = append(dataMap, renvoObjectDataRange{start: mappedEnd, end: a.bssSize, section: section, local: local})
	}
	sectionSymbols := make([]int, len(sections))
	symbols := []renvoObjectELFSymbol{{}}
	for i := 1; i < len(sections); i++ {
		sectionSymbols[i] = len(symbols)
		symbols = append(symbols, renvoObjectELFSymbol{info: 3, section: i})
	}
	// Exported SysV entrypoints are small ABI wrappers around Renvo's internal
	// calling convention. Keep every implementation body visible as a local
	// STT_FUNC as well: linkers do not require it, but kernel objtool needs the
	// function boundary to validate direct calls and stack/control flow.
	for i := 0; i < len(a.objectFunctions); i++ {
		f := &a.objectFunctions[i]
		section, value, ok := renvoObjectMapCode(codeMap, renvoAsmLabelPosition(a, f.label))
		endSection, endValue, endOK := renvoObjectMapCodeEnd(codeMap, f.end)
		if !ok || !endOK || endSection != section || endValue <= value {
			return renvoObjectImageFail("invalid object function range")
		}
		name := renvoObjectStoredText(a, f.nameStart, f.nameEnd)
		symbols = append(symbols, renvoObjectELFSymbol{name: name,
			info: 2, section: section, value: value, size: endValue - value})
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
				return renvoObjectImageFail("invalid code symbol range")
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
			target := renvoObjectStoredText(a, d.targetStart, d.targetEnd)
			alias := d.size <= 0 && target != "" && target != "-"
			if alias || d.binding == 0 && bindingPass != 0 || d.binding != 0 && bindingPass == 0 {
				continue
			}
			name := renvoObjectStoredText(a, d.nameStart, d.nameEnd)
			if name == "" {
				continue
			}
			section, value, ok := renvoObjectMapData(dataMap, d.offset)
			if !ok {
				return renvoObjectImageFail("invalid data symbol range")
			}
			symbolType := 1
			if d.kind == renvoObjectDeclStaticCall {
				symbolType = 2
			}
			symbols = append(symbols, renvoObjectELFSymbol{name: name,
				info: d.binding<<4 | symbolType, visibility: d.visibility, section: section, value: value, size: d.size})
		}
	}
	firstGlobal := len(symbols)
	for i := 1; i < len(symbols); i++ {
		if symbols[i].info>>4 != 0 {
			firstGlobal = i
			break
		}
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
			return renvoObjectImageFail("alias target missing")
		}
		base := symbols[targetIndex]
		base.name = renvoObjectStoredText(a, d.nameStart, d.nameEnd)
		base.info = d.binding<<4 | base.info&15
		base.visibility = d.visibility
		symbols = append(symbols, base)
	}
	importSymbols := make([]int, renvoKernelAmd64ExternalImportCount(a))
	for i := 0; i < len(importSymbols); i++ {
		name := renvoKernelAmd64ExternalImportName(a, i)
		// A C forward reference can first enter the backend as an import and
		// later acquire a definition through an alias attribute. Reuse that
		// defined symbol instead of emitting a second SHN_UNDEF entry.
		importSymbols[i] = renvoObjectFindSymbol(symbols, name)
		if importSymbols[i] >= 0 {
			continue
		}
		importSymbols[i] = len(symbols)
		symbols = append(symbols, renvoObjectELFSymbol{name: name, info: 16})
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
			return renvoObjectImageFail("label relocation range missing")
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
			return renvoObjectImageFail("absolute relocation source missing")
		}
		targetSymbol, targetOffset, relocationType := 0, 0, 2
		if kind == renvoKernelAmd64RelocationImport {
			if addend < 0 || addend >= len(importSymbols) {
				return renvoObjectImageFail("absolute import index invalid")
			}
			targetSymbol = importSymbols[addend]
			relocationType = 4
		} else if kind == renvoKernelAmd64RelocationAbsoluteData {
			section := renvoObjectFindSection(sections, ".rodata")
			if section < 0 {
				return renvoObjectImageFail("absolute rodata section missing")
			}
			targetSymbol, targetOffset = sectionSymbols[section], addend
		} else if kind == renvoKernelAmd64RelocationAbsoluteBSS {
			externalID, externalAddend := renvoObjectMapExternal(a, addend)
			if externalID >= 0 {
				if externalID >= len(importSymbols) {
					return renvoObjectImageFail("absolute external index invalid")
				}
				targetSymbol, targetOffset = importSymbols[externalID], externalAddend
			} else {
				section, value, found := renvoObjectMapData(dataMap, addend)
				if !found {
					renvoPrintErr("renvo: absolute data target offset ")
					renvoPrintIntErr(addend)
					renvoPrintErr(" bss size ")
					renvoPrintIntErr(a.bssSize)
					renvoPrintErr("\n")
					return renvoObjectImageFail("absolute data target missing")
				}
				targetSymbol, targetOffset = sectionSymbols[section], value
			}
		} else if kind == renvoKernelAmd64RelocationAbsoluteBSSEnd {
			section := renvoObjectFindSection(sections, ".bss")
			if section < 0 {
				return renvoObjectImageFail("absolute bss-end section missing")
			}
			targetSymbol, targetOffset = sectionSymbols[section], sections[section].size
		} else {
			return renvoObjectImageFail("unknown absolute relocation kind")
		}
		renvoPut32At(sections[sourceSection].data, sourceOffset, 0)
		sections[sourceSection].relocations = append(sections[sourceSection].relocations,
			renvoObjectELFRelocation{offset: sourceOffset, symbol: targetSymbol, typ: relocationType, addend: targetOffset - 4})
	}
	for i := 0; i < len(a.objectDataRelocs); i++ {
		r := &a.objectDataRelocs[i]
		sourceSection, sourceOffset, found := renvoObjectMapData(dataMap, r.offset)
		if !found {
			renvoPrintErr("renvo: object data relocation source missing ")
			renvoPrintIntErr(i)
			renvoPrintErr("\n")
			return renvoObjectImageFail("data relocation source missing")
		}
		targetSymbol := -1
		addend := r.addend
		if r.codeTarget > 0 || r.codeLabel > 0 {
			targetPosition := r.codeTarget - 1
			if r.codeLabel > 0 {
				targetPosition = renvoAsmLabelPosition(a, r.codeLabel-1)
			}
			targetSection, targetOffset, ok := renvoObjectMapCode(codeMap, targetPosition)
			if !ok {
				renvoPrintErr("renvo: object data relocation code target missing\n")
				return renvoObjectImageFail("data relocation code target missing")
			}
			targetSymbol = sectionSymbols[targetSection]
			addend += targetOffset
		} else if target := renvoObjectStoredText(a, r.targetStart, r.targetEnd); target == "" {
			section := renvoObjectFindSection(sections, ".rodata")
			if section < 0 {
				renvoPrintErr("renvo: object data relocation rodata missing\n")
				return renvoObjectImageFail("data relocation rodata missing")
			}
			targetSymbol = sectionSymbols[section]
		} else {
			targetSymbol = renvoObjectFindSymbol(symbols, target)
			if targetSymbol < 0 {
				renvoPrintErr("renvo: object data relocation symbol missing: ")
				renvoPrintErr(target)
				renvoPrintErr("\n")
				return renvoObjectImageFail("data relocation symbol missing")
			}
		}
		sections[sourceSection].relocations = append(sections[sourceSection].relocations,
			renvoObjectELFRelocation{offset: sourceOffset, symbol: targetSymbol, typ: r.typ, addend: addend})
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
			return renvoObjectImageFail("linkonce signature missing")
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

func renvoObjectMapExternal(a *renvoAsm, offset int) (int, int) {
	for i := 0; i < len(a.objectExternals); i++ {
		external := &a.objectExternals[i]
		if offset >= external.offset && offset < external.offset+renvoObjectExternalStride {
			return external.importID, offset - external.offset
		}
	}
	return -1, 0
}

func renvoObjectCodeRanges(a *renvoAsm) []renvoObjectCodeRange {
	var ranges []renvoObjectCodeRange
	for i := 0; i < len(a.symbols); i++ {
		s := &a.symbols[i]
		if s.endLabel <= 0 {
			continue
		}
		ranges = append(ranges, renvoObjectCodeRange{start: renvoAsmLabelPosition(a, s.label),
			end: renvoAsmLabelPosition(a, s.endLabel), name: renvoObjectStoredText(a, s.sectionStart, s.sectionEnd), alignment: s.alignment, priority: 2})
	}
	for i := 0; i < len(a.objectFunctions); i++ {
		f := &a.objectFunctions[i]
		ranges = append(ranges, renvoObjectCodeRange{start: renvoAsmLabelPosition(a, f.label), end: f.end,
			name: renvoObjectStoredText(a, f.sectionStart, f.sectionEnd), alignment: f.alignment, priority: 1})
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

func renvoObjectCodeUntil(out []byte, size int) []byte {
	// Executable-section alignment must remain a valid instruction stream.
	// Zero bytes can straddle the next function boundary when decoded from the
	// preceding byte, which prevents Linux objtool from finding that function.
	for len(out) < size {
		out = append(out, 0x90)
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
