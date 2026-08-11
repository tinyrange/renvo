package main

func renvoUnitRead32(src []byte, pos int) int {
	return int(src[pos]) | (int(src[pos+1]) << 8) | (int(src[pos+2]) << 16) | (int(src[pos+3]) << 24)
}

type renvoUnitReader struct {
	src []byte
	pos int
	end int
	ok  bool
}

func renvoUnitReadVar(r *renvoUnitReader) int {
	renvoNonNil(r)
	if !r.ok || r.pos >= r.end {
		r.ok = false
		return 0
	}
	first := r.src[r.pos]
	r.pos++
	if first < 0x80 {
		return int(first)
	}
	value := int(first & 0x7f)
	shift := 7
	for r.pos < r.end && shift <= 28 {
		b := r.src[r.pos]
		r.pos++
		if shift >= 28 && b >= 0x10 {
			r.ok = false
			return 0
		}
		value = value | (int(b&0x7f) << shift)
		if b < 0x80 {
			if shift > 0 && b == 0 {
				r.ok = false
				return 0
			}
			return value
		}
		shift = shift + 7
	}
	r.ok = false
	return 0
}

func renvoDecodeUnitTokens(text []byte, data []byte) ([]int32, []int32, bool) {
	r := renvoUnitReader{src: data, end: len(data), ok: true}
	count := renvoUnitReadVar(&r)
	if !r.ok {
		return nil, nil, false
	}
	out := make([]int32, count*renvoTokenStride)
	var lineBases []int32
	start := 0
	line := 0
	discardStart := 0
	nextDiscard := 65536
	for i := 0; i < count; i++ {
		kind := 0
		delta := 0
		size := 0
		lineDelta := 0
		if r.ok && r.pos+4 <= r.end &&
			r.src[r.pos]|r.src[r.pos+1]|r.src[r.pos+2]|r.src[r.pos+3] < 128 {
			kind = int(r.src[r.pos])
			delta = int(r.src[r.pos+1])
			size = int(r.src[r.pos+2])
			lineDelta = int(r.src[r.pos+3])
			r.pos += 4
		} else {
			kind = renvoUnitReadVar(&r)
			delta = renvoUnitReadVar(&r)
			size = renvoUnitReadVar(&r)
			lineDelta = renvoUnitReadVar(&r)
		}
		if !r.ok {
			return nil, nil, false
		}
		start = start + delta
		line = line + lineDelta
		if kind > 255 || start > 0xffffff || start+size > len(text) {
			return nil, nil, false
		}
		if kind == renvoTokOp {
			if size > 255 {
				return nil, nil, false
			}
		} else if size > 0xffff {
			return nil, nil, false
		}
		base := i * renvoTokenStride
		highBits := size >> 8 << 24
		if kind == renvoTokOp && size == 1 {
			highBits = int(text[start]) << 24
		}
		lineHigh := line >> 16
		if lineHigh != 0 && (len(lineBases) == 0 || lineHigh != int(lineBases[len(lineBases)-1])>>24&255) {
			lineBases = append(lineBases, int32(i&0xffffff|lineHigh<<24))
		}
		out[base] = int32(kind | (line&65535)<<8 | highBits)
		out[base+1] = int32(start&0xffffff | (size&255)<<24)
		if r.pos >= nextDiscard {
			// Token records are decoded in order and never revisited. Retire
			// consumed pages while the decoded table grows so both forms do not
			// contribute to the self-host compiler's peak resident set.
			renvo_runtime_ArenaDiscardBytes(data[discardStart:r.pos])
			discardStart = r.pos - 4096
			nextDiscard = r.pos + 65536
		}
	}
	if r.pos != r.end {
		return nil, nil, false
	}
	renvo_runtime_ArenaDiscardBytes(data[discardStart:r.pos])
	return out, lineBases, true
}

func renvoUnitUsesPanic(p *renvoProgram) bool {
	renvoNonNil(p)
	data := p.toks.data
	src := p.src
	for base := 0; base+1 < len(data); base += renvoTokenStride {
		first := int(renvo_runtime_UnsafeInt32At(data, base))
		if first&255 == renvoTokIdent {
			packed := int(renvo_runtime_UnsafeInt32At(data, base+1))
			start := packed & 0xffffff
			size := packed>>24&255 | first>>16&0xff00
			if size == 5 {
				first := renvo_runtime_UnsafeByteAt(src, start)
				if first == 'd' && renvo_runtime_UnsafeByteAt(src, start+1) == 'e' && renvo_runtime_UnsafeByteAt(src, start+2) == 'f' && renvo_runtime_UnsafeByteAt(src, start+3) == 'e' && renvo_runtime_UnsafeByteAt(src, start+4) == 'r' {
					return true
				}
				if first == 'p' && renvo_runtime_UnsafeByteAt(src, start+1) == 'a' && renvo_runtime_UnsafeByteAt(src, start+2) == 'n' && renvo_runtime_UnsafeByteAt(src, start+3) == 'i' && renvo_runtime_UnsafeByteAt(src, start+4) == 'c' {
					return true
				}
			} else if size == 7 && renvo_runtime_UnsafeByteAt(src, start) == 'r' && renvo_runtime_UnsafeByteAt(src, start+1) == 'e' && renvo_runtime_UnsafeByteAt(src, start+2) == 'c' && renvo_runtime_UnsafeByteAt(src, start+3) == 'o' && renvo_runtime_UnsafeByteAt(src, start+4) == 'v' && renvo_runtime_UnsafeByteAt(src, start+5) == 'e' && renvo_runtime_UnsafeByteAt(src, start+6) == 'r' {
				return true
			}
		}
		if first>>24&255 == '.' && base+renvoTokenStride < len(data) && int(renvo_runtime_UnsafeInt32At(data, base+renvoTokenStride))>>24&255 == '(' {
			return true
		}
	}
	return false
}

func renvoDecodeUnitProgram(src []byte) (renvoProgram, bool, bool) {
	var prog renvoProgram
	if len(src) < 4 {
		return prog, false, true
	}
	if src[0] != renvoUnitMagic[0] || src[1] != renvoUnitMagic[1] || src[2] != renvoUnitMagic[2] || src[3] != renvoUnitMagic[3] {
		return prog, false, true
	}
	ok := renvoDecodeUnitProgramBody(src, &prog)
	return prog, true, ok
}

func renvoUnitBindingMatchesTarget(src []byte, target int) bool {
	expectedTarget, expectedDefinition, expectedVersion, ok := renvoRTGTargetBinding(target)
	bindingStart := len(src) - 52 - len(expectedTarget)
	if !ok || len(expectedDefinition) != 32 || bindingStart < 14 ||
		src[0] != renvoUnitMagic[0] || src[1] != renvoUnitMagic[1] ||
		src[2] != renvoUnitMagic[2] || src[3] != renvoUnitMagic[3] {
		return false
	}
	if renvoUnitRead32(src, 10) != len(src)-14 {
		return false
	}
	targetData := bindingStart + 6
	hashHeader := targetData + len(expectedTarget)
	hashData := hashHeader + 6
	versionHeader := hashData + 32
	versionData := versionHeader + 6
	return int(src[bindingStart])|int(src[bindingStart+1])<<8 == 4 &&
		renvoUnitRead32(src, bindingStart+2) == len(expectedTarget) &&
		string(src[targetData:hashHeader]) == expectedTarget &&
		int(src[hashHeader])|int(src[hashHeader+1])<<8 == 5 &&
		renvoUnitRead32(src, hashHeader+2) == 32 &&
		string(src[hashData:versionHeader]) == expectedDefinition &&
		int(src[versionHeader])|int(src[versionHeader+1])<<8 == 6 &&
		renvoUnitRead32(src, versionHeader+2) == 2 &&
		int(src[versionData])|int(src[versionData+1])<<8 == expectedVersion
}

func renvoDecodeUnitProgramBody(src []byte, prog *renvoProgram) bool {
	renvoNonNil(prog)
	if len(src) < 14 {
		return false
	}
	if int(src[4])|(int(src[5])<<8) != renvoUnitVersion {
		return false
	}
	if int(src[6])|(int(src[7])<<8) != 0 {
		return false
	}
	length := renvoUnitRead32(src, 10)
	if int(src[8])|(int(src[9])<<8) != renvoUnitTagUnit || length < 0 {
		return false
	}
	rootStart := 14
	rootEnd := rootStart + length
	if rootEnd != len(src) || rootEnd < rootStart {
		return false
	}
	var text []byte
	textStart := 0
	textEnd := 0
	var tokenData []byte
	var declData []byte
	var funcData []byte
	var packageData []byte
	seenLow := 0
	seenHigh := 0
	pos := rootStart
	for pos < rootEnd {
		if pos+6 > rootEnd {
			return false
		}
		tag := int(src[pos]) | (int(src[pos+1]) << 8)
		length := renvoUnitRead32(src, pos+2)
		pos = pos + 6
		if length < 0 {
			return false
		}
		next := pos + length
		if next < pos || next > rootEnd {
			return false
		}
		if tag == renvoUnitTagUnit {
			return false
		}
		tagIndex := renvoUnitChildTagIndex(tag)
		if tagIndex >= 0 {
			if tagIndex < 16 {
				bit := 1 << tagIndex
				if seenLow&bit != 0 {
					return false
				}
				seenLow = seenLow | bit
			} else {
				bit := 1 << (tagIndex - 16)
				if seenHigh&bit != 0 {
					return false
				}
				seenHigh = seenHigh | bit
			}
		}
		if tag == renvoUnitTagPackage {
			if length == 0 {
				return false
			}
		}
		if tag == renvoUnitTagText {
			text = src[pos:next]
			textStart = pos
			textEnd = next
		}
		if tag == renvoUnitTagTokens {
			tokenData = src[pos:next]
		}
		if tag == renvoUnitTagDecls {
			declData = src[pos:next]
		}
		if tag == renvoUnitTagFuncs {
			funcData = src[pos:next]
		}
		if tag == renvoUnitTagPackages {
			packageData = src[pos:next]
		}
		pos = next
	}
	if seenLow&renvoUnitRequiredChildMaskLow != renvoUnitRequiredChildMaskLow || seenHigh&renvoUnitRequiredChildMaskHigh != renvoUnitRequiredChildMaskHigh {
		return false
	}
	if len(text) == 0 || len(tokenData) == 0 {
		return false
	}
	tokens, lineBases, tokensOK := renvoDecodeUnitTokens(text, tokenData)
	if !tokensOK {
		return false
	}
	tokenCount := len(tokens) / renvoTokenStride
	if tokenCount <= 0 {
		return false
	}
	if int(tokens[(tokenCount-1)*renvoTokenStride])&255 != renvoTokEOF {
		return false
	}
	prog.src = text
	prog.toks.data = tokens
	prog.toks.lineBases = lineBases
	prog.toks.count = tokenCount
	prog.toks.panicEnabled = renvoUnitUsesPanic(prog)
	declReader := renvoUnitReader{src: declData, end: len(declData), ok: true}
	declCount := renvoUnitReadVar(&declReader)
	if !declReader.ok {
		return false
	}
	prog.decls = make([]renvoDecl, 0, declCount)
	for i := 0; i < declCount; i++ {
		var decl renvoDecl
		nameSize := 0
		tokCount := 0
		decl.kind = renvoUnitReadVar(&declReader)
		decl.nameStart = renvoUnitReadVar(&declReader)
		nameSize = renvoUnitReadVar(&declReader)
		decl.startTok = renvoUnitReadVar(&declReader)
		tokCount = renvoUnitReadVar(&declReader)
		if !declReader.ok {
			return false
		}
		decl.nameEnd = decl.nameStart + nameSize
		decl.endTok = decl.startTok + tokCount
		if !renvoUnitValidRange(len(text), decl.nameStart, decl.nameEnd) || !renvoUnitValidTokenRange(tokenCount, decl.startTok, decl.endTok) {
			return false
		}
		prog.decls = append(prog.decls, decl)
	}
	if declReader.pos != declReader.end {
		return false
	}
	funcReader := renvoUnitReader{src: funcData, end: len(funcData), ok: true}
	funcCount := renvoUnitReadVar(&funcReader)
	if !funcReader.ok {
		return false
	}
	prog.funcs = make([]renvoFuncDecl, 0, funcCount)
	for i := 0; i < funcCount; i++ {
		var fn renvoFuncDecl
		nameSize := 0
		nameTokDelta := 0
		receiverCount := 0
		bodyCount := 0
		endCount := 0
		fn.nameStart = renvoUnitReadVar(&funcReader)
		nameSize = renvoUnitReadVar(&funcReader)
		fn.startTok = renvoUnitReadVar(&funcReader)
		nameTokDelta = renvoUnitReadVar(&funcReader)
		fn.receiverStart = renvoUnitReadVar(&funcReader)
		receiverCount = renvoUnitReadVar(&funcReader)
		fn.bodyStart = renvoUnitReadVar(&funcReader)
		bodyCount = renvoUnitReadVar(&funcReader)
		endCount = renvoUnitReadVar(&funcReader)
		if !funcReader.ok {
			return false
		}
		fn.nameEnd = fn.nameStart + nameSize
		fn.nameTok = fn.startTok + nameTokDelta
		fn.receiverEnd = fn.receiverStart + receiverCount
		fn.bodyEnd = fn.bodyStart + bodyCount
		fn.endTok = fn.bodyEnd + endCount
		if !renvoUnitValidRange(len(text), fn.nameStart, fn.nameEnd) || !renvoUnitValidTokenRange(tokenCount, fn.startTok, fn.endTok) {
			return false
		}
		if fn.nameTok < 0 || fn.nameTok >= tokenCount || fn.bodyStart < 0 || fn.bodyEnd >= tokenCount || fn.bodyStart > fn.bodyEnd {
			return false
		}
		prog.funcs = append(prog.funcs, fn)
	}
	if funcReader.pos != funcReader.end {
		return false
	}
	if len(packageData) > 0 {
		packageReader := renvoUnitReader{src: packageData, end: len(packageData), ok: true}
		packageCount := renvoUnitReadVar(&packageReader)
		if !packageReader.ok {
			return false
		}
		prog.packageTable = &renvoPackageTable{items: make([]renvoPackageInfo, 0, packageCount)}
		for i := 0; i < packageCount; i++ {
			nameLength := renvoUnitReadVar(&packageReader)
			if !packageReader.ok || nameLength <= 0 || packageReader.pos+nameLength > packageReader.end {
				return false
			}
			packageReader.pos += nameLength
			pathLength := renvoUnitReadVar(&packageReader)
			if !packageReader.ok || pathLength <= 0 || packageReader.pos+pathLength > packageReader.end {
				return false
			}
			pathStart := packageReader.pos
			pathKeyA, pathKeyB := renvoObjectHashRange(1879, 3761, packageReader.src, pathStart, pathStart+pathLength)
			packageReader.pos += pathLength
			if packageReader.pos+16 > packageReader.end {
				return false
			}
			var item renvoPackageInfo
			item.graphKeyA = renvoUnitRead32(packageReader.src, packageReader.pos)
			item.graphKeyB = renvoUnitRead32(packageReader.src, packageReader.pos+4)
			item.sourceKeyA = renvoUnitRead32(packageReader.src, packageReader.pos+8)
			item.sourceKeyB = renvoUnitRead32(packageReader.src, packageReader.pos+12)
			item.pathKeyA = pathKeyA
			item.pathKeyB = pathKeyB
			packageReader.pos += 16
			textLength := 0
			tokenLength := 0
			declLength := 0
			funcLength := 0
			item.textStart = renvoUnitReadVar(&packageReader)
			textLength = renvoUnitReadVar(&packageReader)
			item.tokenStart = renvoUnitReadVar(&packageReader)
			tokenLength = renvoUnitReadVar(&packageReader)
			item.declStart = renvoUnitReadVar(&packageReader)
			declLength = renvoUnitReadVar(&packageReader)
			item.funcStart = renvoUnitReadVar(&packageReader)
			funcLength = renvoUnitReadVar(&packageReader)
			item.textEnd = item.textStart + textLength
			item.tokenEnd = item.tokenStart + tokenLength
			item.declEnd = item.declStart + declLength
			item.funcEnd = item.funcStart + funcLength
			if !packageReader.ok || !renvoUnitValidRange(len(text), item.textStart, item.textEnd) || !renvoUnitValidRange(tokenCount, item.tokenStart, item.tokenEnd) || !renvoUnitValidRange(len(prog.decls), item.declStart, item.declEnd) || !renvoUnitValidRange(len(prog.funcs), item.funcStart, item.funcEnd) {
				return false
			}
			prog.packageTable.items = append(prog.packageTable.items, item)
		}
		if packageReader.pos != packageReader.end {
			return false
		}
	}
	renvo_runtime_ArenaDiscardBytes(src[:textStart])
	renvo_runtime_ArenaDiscardBytes(src[textEnd:])
	renvoSetCompilerIntWidth(prog)
	prog.ok = true
	return true
}

func renvoUnitValidRange(limit int, start int, end int) bool {
	if start < 0 || end < start {
		return false
	}
	return end <= limit
}

func renvoUnitValidTokenRange(limit int, start int, end int) bool {
	if start < 0 || end < start {
		return false
	}
	return end <= limit
}
