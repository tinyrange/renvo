package main

const rtgAbsBssReloc = 1

type rtgLabelRef struct {
	at    int
	label int
}

type rtgAbsRef struct {
	at   int
	off  int
	kind int
}

type rtgAsm struct {
	code       []byte
	labelPos   []int
	labelSet   []bool
	relocs     []rtgLabelRef
	absRelocs  []rtgAbsRef
	data       []byte
	bssSize    int
	codeOffset int
	dataOffset int
}

func rtgAsmInit(a *rtgAsm) {
	var code []byte
	var labelPos []int
	var labelSet []bool
	var relocs []rtgLabelRef
	var absRelocs []rtgAbsRef
	var data []byte
	a.code = code
	a.labelPos = labelPos
	a.labelSet = labelSet
	a.relocs = relocs
	a.absRelocs = absRelocs
	a.data = data
	a.bssSize = 0
	a.codeOffset = 0
	a.dataOffset = 0
}

func rtgAsmNewLabel(a *rtgAsm) int {
	a.labelPos = append(a.labelPos, 0)
	a.labelSet = append(a.labelSet, false)
	label := len(a.labelPos) - 1
	return label
}

func rtgAsmMarkLabel(a *rtgAsm, label int) {
	if label >= 0 && label < len(a.labelPos) {
		codeLen := len(a.code)
		a.labelPos[label] = codeLen
		a.labelSet[label] = true
	}
}

func rtgAsmEmit8(a *rtgAsm, v int) {
	a.code = append(a.code, byte(v))
}

func rtgAsmEmit2(a *rtgAsm, v0 int, v1 int) {
	a.code = append(a.code, byte(v0))
	a.code = append(a.code, byte(v1))
}

func rtgAsmEmit3(a *rtgAsm, v0 int, v1 int, v2 int) {
	a.code = append(a.code, byte(v0))
	a.code = append(a.code, byte(v1))
	a.code = append(a.code, byte(v2))
}

func rtgAsmEmit4(a *rtgAsm, v0 int, v1 int, v2 int, v3 int) {
	a.code = append(a.code, byte(v0))
	a.code = append(a.code, byte(v1))
	a.code = append(a.code, byte(v2))
	a.code = append(a.code, byte(v3))
}

func rtgAsmAddAbsReloc(a *rtgAsm, at int, off int, kind int) {
	a.absRelocs = append(a.absRelocs, rtgAbsRef{at: at, off: off, kind: kind})
}

func rtgAsmAddReloc(a *rtgAsm, at int, label int) {
	a.relocs = append(a.relocs, rtgLabelRef{at: at, label: label})
}

func rtgAsmEmit32(a *rtgAsm, v int) {
	a.code = rtgAppend32(a.code, v)
}

func rtgAsmEmit64(a *rtgAsm, v int) {
	a.code = rtgAppend64(a.code, v)
}

func rtgAsmEmit16(a *rtgAsm, v int) {
	rtgAsmEmit8(a, v)
	rtgAsmEmit8(a, v>>8)
}

func rtgAsmEmit24(a *rtgAsm, v int) {
	rtgAsmEmit8(a, v)
	rtgAsmEmit8(a, v>>8)
	rtgAsmEmit8(a, v>>16)
}

func rtgAsmPatch(a *rtgAsm) {
	for i := 0; i < len(a.relocs); i++ {
		r := a.relocs[i]
		if r.label >= 0 && r.label < len(a.labelPos) && a.labelSet[r.label] {
			target := a.labelPos[r.label]
			disp := target - (r.at + 4)
			rtgPut32At(a.code, r.at, disp)
		}
	}
	a.dataOffset = a.codeOffset + len(a.code)
	for i := 0; i < len(a.absRelocs); i++ {
		r := a.absRelocs[i]
		target := a.dataOffset + r.off
		if r.kind == rtgAbsBssReloc {
			target = a.dataOffset + len(a.data) + r.off
		}
		next := a.codeOffset + r.at + 4
		disp := target - next
		rtgPut32At(a.code, r.at, disp)
	}
}

func rtgPut32At(out []byte, at int, v int) {
	b0 := byte(v)
	b1 := byte(v >> 8)
	b2 := byte(v >> 16)
	b3 := byte(v >> 24)
	out[at] = b0
	out[at+1] = b1
	out[at+2] = b2
	out[at+3] = b3
}

func rtgAppend16(out []byte, v int) []byte {
	out = append(out, byte(v))
	out = append(out, byte(v>>8))
	return out
}

func rtgAppend32(out []byte, v int) []byte {
	out = append(out, byte(v))
	out = append(out, byte(v>>8))
	out = append(out, byte(v>>16))
	out = append(out, byte(v>>24))
	return out
}

func rtgAppend64(out []byte, v int) []byte {
	out = append(out, byte(v))
	out = append(out, byte(v>>8))
	out = append(out, byte(v>>16))
	out = append(out, byte(v>>24))
	out = append(out, byte(v>>32))
	out = append(out, byte(v>>40))
	out = append(out, byte(v>>48))
	out = append(out, byte(v>>56))
	return out
}

const rtgTokEOF = 0
const rtgTokIdent = 1
const rtgTokNumber = 2
const rtgTokFloat = 3
const rtgTokString = 4
const rtgTokChar = 5
const rtgTokPackage = 6
const rtgTokConst = 7
const rtgTokVar = 8
const rtgTokType = 9
const rtgTokFunc = 10
const rtgTokStruct = 11
const rtgTokReturn = 12
const rtgTokIf = 13
const rtgTokElse = 14
const rtgTokFor = 15
const rtgTokBreak = 16
const rtgTokContinue = 17
const rtgTokGoto = 18
const rtgTokSwitch = 19
const rtgTokCase = 20
const rtgTokDefault = 21
const rtgTokOp = 22

type rtgToken struct {
	kind  int
	start int
	end   int
	line  int
}

type rtgDecl struct {
	kind      int
	nameStart int
	nameEnd   int
	startTok  int
	endTok    int
}

type rtgFuncDecl struct {
	nameStart     int
	nameEnd       int
	startTok      int
	nameTok       int
	receiverStart int
	receiverEnd   int
	bodyStart     int
	bodyEnd       int
	endTok        int
}

type rtgProgram struct {
	src   []byte
	toks  []rtgToken
	decls []rtgDecl
	funcs []rtgFuncDecl
	ok    bool
}

const rtgExprBad = 0
const rtgExprIdent = 1
const rtgExprInt = 2
const rtgExprFloat = 3
const rtgExprString = 4
const rtgExprChar = 5
const rtgExprBool = 6
const rtgExprUnary = 7
const rtgExprBinary = 8
const rtgExprCall = 9
const rtgExprIndex = 10
const rtgExprSelector = 11
const rtgExprComposite = 12
const rtgExprSlice = 13

const rtgStmtBad = 0
const rtgStmtReturn = 1
const rtgStmtIf = 2
const rtgStmtFor = 3
const rtgStmtBreak = 4
const rtgStmtContinue = 5
const rtgStmtGoto = 6
const rtgStmtLabel = 7
const rtgStmtVar = 8
const rtgStmtShort = 9
const rtgStmtAssign = 10
const rtgStmtExpr = 11
const rtgStmtSwitch = 12
const rtgStmtBlock = 13

type rtgExpr struct {
	kind      int
	tok       int
	left      int
	right     int
	firstArg  int
	argCount  int
	nameStart int
	nameEnd   int
}

type rtgExprParse struct {
	prog     *rtgProgram
	pos      int
	end      int
	exprs    []rtgExpr
	args     []int
	fields   []rtgCompositeField
	ok       bool
	hasFloat bool
}

type rtgCompositeField struct {
	nameStart int
	nameEnd   int
	expr      int
}

type rtgStmt struct {
	kind      int
	startTok  int
	endTok    int
	exprStart int
	exprEnd   int
	bodyStart int
	bodyEnd   int
	elseStart int
	elseEnd   int
	nameStart int
	nameEnd   int
}

type rtgBodyParse struct {
	prog  *rtgProgram
	stmts []rtgStmt
	ok    bool
}

const rtgTypeInvalid = 0
const rtgTypeInt = 1
const rtgTypeInt64 = 2
const rtgTypeByte = 3
const rtgTypeBool = 4
const rtgTypeString = 5
const rtgTypeFloat64 = 6
const rtgTypeInt16 = 7
const rtgTypeInt32 = 8
const rtgTypePointer = 9
const rtgTypeSlice = 10
const rtgTypeStruct = 11
const rtgTypeNamed = 12

type rtgTypeInfo struct {
	kind      int
	elem      int
	first     int
	count     int
	size      int
	nameStart int
	nameEnd   int
}

type rtgFieldInfo struct {
	nameStart int
	nameEnd   int
	typ       int
	offset    int
}

type rtgSymbolInfo struct {
	nameStart    int
	nameEnd      int
	kind         int
	typ          int
	initStart    int
	initEnd      int
	iotaValue    int
	constValue   int
	constValueOK int
}

type rtgFuncInfo struct {
	declIndex    int
	nameStart    int
	nameEnd      int
	firstParam   int
	paramCount   int
	resultType   int
	receiverType int
	bodyStart    int
	bodyEnd      int
}

type rtgMeta struct {
	prog    *rtgProgram
	types   []rtgTypeInfo
	fields  []rtgFieldInfo
	globals []rtgSymbolInfo
	params  []rtgSymbolInfo
	funcs   []rtgFuncInfo
	ok      bool
}

type rtgCompileResult struct {
	data []byte
	ok   bool
}

type rtgConstResult struct {
	value int
	ok    bool
}

type rtgTypeResult struct {
	typ  int
	next int
}

const rtgIdentAppend = 1
const rtgIdentByteSlice = 2
const rtgIdentMake = 3
const rtgIdentInt = 5
const rtgIdentInt64 = 6
const rtgIdentByte = 7
const rtgIdentLen = 8
const rtgIdentOpen = 9
const rtgIdentClose = 10
const rtgIdentRead = 11
const rtgIdentWrite = 12
const rtgIdentChmod = 13
const rtgIdentCopy = 14
const rtgIdentInt16 = 15
const rtgIdentInt32 = 16

const rtgDiagParseMissingPackage = 1
const rtgDiagParseMissingPackageName = 2
const rtgDiagParsePackageName = 3
const rtgDiagParseGroupedDecl = 4
const rtgDiagParseTopDecl = 5
const rtgDiagParseFuncDecl = 6
const rtgDiagParseStatement = 7
const rtgDiagParseExpression = 8
const rtgDiagParseComposite = 9
const rtgDiagParseCall = 10
const rtgDiagParseIndex = 11
const rtgDiagParseParen = 12
const rtgDiagMetaConstDecl = 20
const rtgDiagMetaTopDecl = 21
const rtgDiagMetaFuncDecl = 22
const rtgDiagMetaResultType = 23
const rtgDiagMetaParamList = 24
const rtgDiagAppMainRequired = 40
const rtgDiagMainRequiresAppMain = 41
const rtgDiagAppMainSignature = 42
const rtgDiagGlobalCodegen = 50
const rtgDiagFunctionCodegen = 51
const rtgDiagCompileFailed = 52
const rtgDiagFunctionParams = 53
const rtgDiagStatementCodegen = 54
const rtgDiagAssignmentCodegen = 55
const rtgDiagReturnCodegen = 56
const rtgDiagConditionCodegen = 57
const rtgDiagSwitchCodegen = 58
const rtgDiagCallCodegen = 59
const rtgDiagBreakOutsideLoop = 60
const rtgDiagContinueOutsideLoop = 61
const rtgDiagUnsupportedStatement = 62

func rtgProgramError(p *rtgProgram, diag int) {
	p.ok = false
}

func rtgMetaError(m *rtgMeta, diag int) {
	m.ok = false
}

func rtgExprError(ep *rtgExprParse, diag int) {
	ep.ok = false
}

func rtgParseProgram(src []byte) rtgProgram {
	var p rtgProgram
	p.src = src
	p.toks = rtgScan(src)
	p.ok = true

	i := 0
	if !rtgTokIsKind(&p, i, rtgTokPackage) {
		rtgProgramError(&p, rtgDiagParseMissingPackage)
		return p
	}
	i++
	if !rtgTokIsKind(&p, i, rtgTokIdent) {
		rtgProgramError(&p, rtgDiagParseMissingPackageName)
		return p
	}
	i++

	for i < len(p.toks) && p.toks[i].kind != rtgTokEOF {
		if rtgTokIsKind(&p, i, rtgTokPackage) {
			i++
			if !rtgTokIsKind(&p, i, rtgTokIdent) {
				rtgProgramError(&p, rtgDiagParsePackageName)
				return p
			}
			i++
			continue
		}
		if rtgTokIsKind(&p, i, rtgTokConst) || rtgTokIsKind(&p, i, rtgTokVar) || rtgTokIsKind(&p, i, rtgTokType) {
			start := i
			kind := p.toks[i].kind
			i++
			if rtgTokCharIs(&p, i, '(') {
				end := rtgSkipBalanced(&p, i, '(', ')')
				if end <= i {
					rtgProgramError(&p, rtgDiagParseGroupedDecl)
					return p
				}
				var decl rtgDecl
				decl.kind = kind
				decl.nameStart = p.toks[start].start
				decl.nameEnd = p.toks[start].end
				decl.startTok = start
				decl.endTok = end
				p.decls = append(p.decls, decl)
				i = end
				continue
			}
			if !rtgTokIsKind(&p, i, rtgTokIdent) {
				rtgProgramError(&p, rtgDiagParseTopDecl)
				return p
			}
			name := &p.toks[i]
			i++
			end := rtgSkipTopLevelLine(&p, i)
			var decl rtgDecl
			decl.kind = kind
			decl.nameStart = name.start
			decl.nameEnd = name.end
			decl.startTok = start
			decl.endTok = end
			p.decls = append(p.decls, decl)
			i = end
			continue
		}
		if rtgTokIsKind(&p, i, rtgTokFunc) {
			var fn rtgFuncDecl
			rtgParseFuncDecl(&p, i, &fn)
			if fn.endTok <= i {
				rtgProgramError(&p, rtgDiagParseFuncDecl)
				return p
			}
			p.funcs = append(p.funcs, fn)
			i = fn.endTok
			continue
		}
		i++
	}

	return p
}

func rtgParseFuncDecl(p *rtgProgram, start int, fn *rtgFuncDecl) {
	fn.startTok = start
	i := start + 1
	if !rtgTokIsKind(p, i, rtgTokIdent) {
		receiverEnd := i + 1
		for receiverEnd < len(p.toks) && !rtgTokCharIs(p, receiverEnd, ')') {
			receiverEnd++
		}
		if receiverEnd <= i {
			return
		}
		fn.receiverStart = i + 1
		fn.receiverEnd = receiverEnd
		i = receiverEnd + 1
	}
	if !rtgTokIsKind(p, i, rtgTokIdent) {
		return
	}
	fn.nameTok = i
	fn.nameStart = p.toks[i].start
	fn.nameEnd = p.toks[i].end
	i++

	for i < len(p.toks) && !rtgTokCharIs(p, i, '{') && p.toks[i].kind != rtgTokEOF {
		i++
	}
	if !rtgTokCharIs(p, i, '{') {
		return
	}
	fn.bodyStart = i
	depth := 1
	i++
	for i < len(p.toks) && depth > 0 {
		if rtgTokCharIs(p, i, '{') {
			depth++
		} else if rtgTokCharIs(p, i, '}') {
			depth--
		}
		i++
	}
	if depth != 0 {
		return
	}
	fn.bodyEnd = i - 1
	fn.endTok = i
}

func rtgSkipBalanced(p *rtgProgram, start int, open byte, close byte) int {
	if !rtgTokCharIs(p, start, open) {
		return start
	}
	depth := 1
	i := start + 1
	for i < len(p.toks) && depth > 0 {
		if rtgTokCharIs(p, i, open) {
			depth++
		} else if rtgTokCharIs(p, i, close) {
			depth--
		}
		i++
	}
	if depth != 0 {
		return start
	}
	return i
}

func rtgSkipTopLevelLine(p *rtgProgram, start int) int {
	if start >= len(p.toks) {
		return start
	}
	line := p.toks[start-1].line
	i := start
	depth := 0
	for i < len(p.toks) {
		if p.toks[i].kind == rtgTokEOF {
			return i
		}
		if p.toks[i].line != line && depth == 0 {
			return i
		}
		if rtgTokCharIs(p, i, '{') || rtgTokCharIs(p, i, '(') {
			depth++
		} else if rtgTokCharIs(p, i, '}') || rtgTokCharIs(p, i, ')') {
			depth--
		}
		i++
	}
	return i
}

func rtgScan(src []byte) []rtgToken {
	var toks []rtgToken
	i := 0
	line := 1
	for i < len(src) {
		c := src[i]
		if c == ' ' || c == '\t' || c == '\r' {
			i++
			continue
		}
		if c == '\n' {
			line++
			i++
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '/' {
			i += 2
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '*' {
			i += 2
			for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
				if src[i] == '\n' {
					line++
				}
				i++
			}
			if i+1 < len(src) {
				i += 2
			}
			continue
		}
		if rtgIsIdentStart(c) {
			i++
			start := i - 1
			for i < len(src) && rtgIsIdentPart(src[i]) {
				i++
			}
			toks = append(toks, rtgToken{kind: rtgKeywordKind(src, start, i), start: start, end: i, line: line})
			continue
		}
		if rtgIsDigit(c) {
			start := i
			kind := rtgTokNumber
			if c == '0' && i+1 < len(src) && (src[i+1] == 'x' || src[i+1] == 'X' || src[i+1] == 'b' || src[i+1] == 'B') {
				i += 2
				for i < len(src) && rtgIsIdentPart(src[i]) {
					i++
				}
			} else {
				i++
				for i < len(src) && rtgIsDigit(src[i]) {
					i++
				}
				if i < len(src) && src[i] == '.' {
					kind = rtgTokFloat
					i++
					for i < len(src) && rtgIsDigit(src[i]) {
						i++
					}
				}
			}
			toks = append(toks, rtgToken{kind: kind, start: start, end: i, line: line})
			continue
		}
		if c == '"' {
			start := i
			i++
			for i < len(src) && src[i] != '"' {
				if src[i] == '\\' && i+1 < len(src) {
					i += 2
				} else {
					if src[i] == '\n' {
						line++
					}
					i++
				}
			}
			if i < len(src) {
				i++
			}
			toks = append(toks, rtgToken{kind: rtgTokString, start: start, end: i, line: line})
			continue
		}
		if c == '\'' {
			start := i
			i++
			for i < len(src) && src[i] != '\'' {
				if src[i] == '\\' && i+1 < len(src) {
					i += 2
				} else {
					i++
				}
			}
			if i < len(src) {
				i++
			}
			toks = append(toks, rtgToken{kind: rtgTokChar, start: start, end: i, line: line})
			continue
		}
		start := i
		i = rtgScanOperator(src, i)
		toks = append(toks, rtgToken{kind: rtgTokOp, start: start, end: i, line: line})
	}
	toks = append(toks, rtgToken{kind: rtgTokEOF, start: len(src), end: len(src), line: line})
	return toks
}

func rtgScanOperator(src []byte, i int) int {
	if i+2 <= len(src) {
		c0 := src[i]
		c1 := src[i+1]
		if c1 == '=' {
			if c0 == ':' || c0 == '=' || c0 == '!' || c0 == '<' || c0 == '>' || c0 == '+' || c0 == '-' || c0 == '*' || c0 == '/' || c0 == '%' {
				return i + 2
			}
		}
		if c0 == '&' && (c1 == '&' || c1 == '^') {
			return i + 2
		}
		if c0 == '|' && c1 == '|' {
			return i + 2
		}
		if c0 == '<' && c1 == '<' {
			return i + 2
		}
		if c0 == '>' && c1 == '>' {
			return i + 2
		}
		if c0 == '+' && c1 == '+' {
			return i + 2
		}
		if c0 == '-' && c1 == '-' {
			return i + 2
		}
	}
	return i + 1
}

func rtgKeywordKind(src []byte, start int, end int) int {
	n := end - start
	h := 0
	for i := start; i < end; i++ {
		h = h*5 + int(src[i])
	}
	if n == 2 {
		if h == 627 {
			return rtgTokIf
		}
	}
	if n == 3 {
		if h == 3549 {
			return rtgTokVar
		}
		if h == 3219 {
			return rtgTokFor
		}
	}
	if n == 4 {
		if h == 18186 {
			return rtgTokType
		}
		if h == 16324 {
			return rtgTokFunc
		}
		if h == 16001 {
			return rtgTokElse
		}
		if h == 16341 {
			return rtgTokGoto
		}
		if h == 15476 {
			return rtgTokCase
		}
	}
	if n == 5 {
		if h == 79191 {
			return rtgTokConst
		}
		if h == 78617 {
			return rtgTokBreak
		}
	}
	if n == 6 {
		if h == 449661 {
			return rtgTokStruct
		}
		if h == 437480 {
			return rtgTokReturn
		}
		if h == 450374 {
			return rtgTokSwitch
		}
	}
	if n == 7 {
		if h == 2131416 {
			return rtgTokPackage
		}
		if h == 1957581 {
			return rtgTokDefault
		}
	}
	if n == 8 {
		if h == 9901561 {
			return rtgTokContinue
		}
	}
	return rtgTokIdent
}

func rtgTokIsKind(p *rtgProgram, i int, kind int) bool {
	return i >= 0 && i < len(p.toks) && p.toks[i].kind == kind
}

func rtgTokCharIs(p *rtgProgram, i int, c byte) bool {
	if i < 0 || i >= len(p.toks) {
		return false
	}
	start := p.toks[i].start
	end := p.toks[i].end
	return end-start == 1 && p.src[start] == c
}

func rtgTok2Is(p *rtgProgram, i int, a byte, b byte) bool {
	if i < 0 || i >= len(p.toks) {
		return false
	}
	start := p.toks[i].start
	end := p.toks[i].end
	return end-start == 2 && p.src[start] == a && p.src[start+1] == b
}

func rtgBoolTokenValue(p *rtgProgram, tok int) int {
	start := p.toks[tok].start
	if p.src[start] == 't' {
		return 1
	}
	return 0
}

func rtgTokIsCompoundAssign(p *rtgProgram, i int) bool {
	if i < 0 || i >= len(p.toks) {
		return false
	}
	start := p.toks[i].start
	end := p.toks[i].end
	if end-start != 2 || p.src[start+1] != '=' {
		return false
	}
	c := p.src[start]
	return c == '+' || c == '-' || c == '*' || c == '/' || c == '%'
}

func rtgExprIdentCode(p *rtgProgram, ep *rtgExprParse, idx int) int {
	e := ep.exprs[idx]
	if e.kind != rtgExprIdent {
		return 0
	}
	if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "append") {
		return rtgIdentAppend
	}
	if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "[]byte") {
		return rtgIdentByteSlice
	}
	if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "make") {
		return rtgIdentMake
	}
	if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "int") {
		return rtgIdentInt
	}
	if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "int16") {
		return rtgIdentInt16
	}
	if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "int32") {
		return rtgIdentInt32
	}
	if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "int64") {
		return rtgIdentInt64
	}
	if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "byte") {
		return rtgIdentByte
	}
	if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "len") {
		return rtgIdentLen
	}
	if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "open") {
		return rtgIdentOpen
	}
	if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "close") {
		return rtgIdentClose
	}
	if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "read") {
		return rtgIdentRead
	}
	if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "write") {
		return rtgIdentWrite
	}
	if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "chmod") {
		return rtgIdentChmod
	}
	if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "copy") {
		return rtgIdentCopy
	}
	return 0
}

func rtgIsIdentStart(c byte) bool {
	if c >= 'a' {
		if c <= 'z' {
			return true
		}
	}
	if c >= 'A' {
		if c <= 'Z' {
			return true
		}
	}
	if c == '_' {
		return true
	}
	return false
}

func rtgIsIdentPart(c byte) bool {
	if rtgIsIdentStart(c) {
		return true
	}
	if rtgIsDigit(c) {
		return true
	}
	return false
}

func rtgIsDigit(c byte) bool {
	if c >= '0' {
		if c <= '9' {
			return true
		}
	}
	return false
}

func rtgBytesEqualText(src []byte, start int, end int, text string) bool {
	if end-start != len(text) {
		return false
	}
	for i := 0; i < len(text); i++ {
		if src[start+i] != text[i] {
			return false
		}
	}
	return true
}

func rtgDecodeStringToken(p *rtgProgram, tokIndex int) []byte {
	tok := &p.toks[tokIndex]
	src := p.src
	var out []byte
	i := tok.start + 1
	end := tok.end - 1
	for i < end {
		if src[i] == '\\' && i+1 < end {
			i++
			if src[i] == 'n' {
				out = append(out, '\n')
			} else if src[i] == 't' {
				out = append(out, '\t')
			} else if src[i] == 'r' {
				out = append(out, '\r')
			} else if src[i] == '"' {
				out = append(out, '"')
			} else if src[i] == '\\' {
				out = append(out, '\\')
			} else {
				out = append(out, src[i])
			}
			i++
			continue
		}
		out = append(out, src[i])
		i++
	}
	return out
}

func rtgParseIntToken(p *rtgProgram, tokIndex int) int {
	tok := &p.toks[tokIndex]
	src := p.src
	start := tok.start
	base := 10
	if tok.end-start > 2 && src[start] == '0' && (src[start+1] == 'x' || src[start+1] == 'X') {
		base = 16
		start += 2
	} else if tok.end-start > 2 && src[start] == '0' && (src[start+1] == 'b' || src[start+1] == 'B') {
		base = 2
		start += 2
	} else if tok.end-start > 1 && src[start] == '0' {
		base = 8
		start++
	}
	n := 0
	for i := start; i < tok.end; i++ {
		d := 0
		if src[i] >= '0' && src[i] <= '9' {
			d = int(src[i] - '0')
		} else if src[i] >= 'a' && src[i] <= 'f' {
			d = int(src[i]-'a') + 10
		} else if src[i] >= 'A' && src[i] <= 'F' {
			d = int(src[i]-'A') + 10
		}
		n = n*base + d
	}
	return n
}

func rtgParseFloatTokenScaled(p *rtgProgram, tokIndex int) int {
	tok := &p.toks[tokIndex]
	value := 0
	i := tok.start
	for i < tok.end && p.src[i] != '.' {
		if p.src[i] >= '0' && p.src[i] <= '9' {
			value = value*10 + int(p.src[i]-'0')
		}
		i++
	}
	value = value * 4
	if i < tok.end && p.src[i] == '.' {
		i++
		frac := 0
		scale := 1
		for i < tok.end {
			if p.src[i] >= '0' && p.src[i] <= '9' {
				frac = frac*10 + int(p.src[i]-'0')
				scale = scale * 10
			}
			i++
		}
		if scale > 1 {
			value += (frac * 4) / scale
		}
	}
	return value
}

func rtgParseCharToken(p *rtgProgram, tokIndex int) int {
	tok := &p.toks[tokIndex]
	src := p.src
	i := tok.start + 1
	if i >= tok.end-1 {
		return 0
	}
	if src[i] != '\\' {
		return int(src[i])
	}
	i++
	if i >= tok.end-1 {
		return 0
	}
	if src[i] == 'n' {
		return 10
	}
	if src[i] == 't' {
		return 9
	}
	if src[i] == 'r' {
		return 13
	}
	if src[i] == '\\' {
		return 92
	}
	if src[i] == '\'' {
		return 39
	}
	return int(src[i])
}

func rtgEvalConstByName(g *rtgLinearGen, nameStart int, nameEnd int) rtgConstResult {
	builtin := rtgEvalBuiltinConst(g, nameStart, nameEnd)
	if builtin.ok {
		return builtin
	}
	for i := 0; i < len(g.meta.globals); i++ {
		s := &g.meta.globals[i]
		if s.kind == rtgTokConst && rtgBytesEqualRange(g.prog.src, s.nameStart, s.nameEnd, nameStart, nameEnd) {
			if s.constValueOK != 0 {
				return rtgConstResultOk(s.constValue)
			}
			ep := rtgParseExpression(g.prog, s.initStart, s.initEnd)
			if !ep.ok || len(ep.exprs) == 0 {
				var r rtgConstResult
				return r
			}
			rootIndex := len(ep.exprs) - 1
			oldIota := g.constEvalIota
			oldIotaValid := g.constEvalIotaValid
			g.constEvalIota = s.iotaValue
			g.constEvalIotaValid = 1
			result := rtgEvalConstExpr(g, &ep, rootIndex)
			value := result.value
			ok := result.ok
			g.constEvalIota = oldIota
			g.constEvalIotaValid = oldIotaValid
			if ok {
				return rtgConstResultOk(value)
			}
			var r rtgConstResult
			return r
		}
	}
	var r rtgConstResult
	return r
}

func rtgConstResultOk(value int) rtgConstResult {
	var r rtgConstResult
	r.value = value
	r.ok = true
	return r
}

func rtgConvertConstInt(value int, kind int) int {
	if kind == rtgTypeByte {
		return value & 0xff
	}
	if kind == rtgTypeInt && rtgNativeIntSize == 4 {
		kind = rtgTypeInt32
	}
	if kind == rtgTypeInt16 {
		value = value & 0xffff
		if value >= 0x8000 {
			value -= 0x10000
		}
		return value
	}
	if kind == rtgTypeInt32 {
		limit := 2147483647
		if value > limit {
			value -= limit
			value -= limit
			value -= 2
		}
		return value
	}
	return value
}

func rtgEvalConstExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) rtgConstResult {
	p := g.prog
	e := &ep.exprs[idx]
	if e.kind == rtgExprInt {
		value := rtgParseIntToken(p, e.tok)
		return rtgConstResultOk(value)
	}
	if e.kind == rtgExprFloat {
		value := rtgParseFloatTokenScaled(p, e.tok)
		return rtgConstResultOk(value)
	}
	if e.kind == rtgExprChar {
		value := rtgParseCharToken(p, e.tok)
		return rtgConstResultOk(value)
	}
	if e.kind == rtgExprBool {
		value := rtgBoolTokenValue(p, e.tok)
		return rtgConstResultOk(value)
	}
	if e.kind == rtgExprIdent {
		result := rtgEvalConstByName(g, e.nameStart, e.nameEnd)
		return result
	}
	if e.kind == rtgExprCall {
		callee := rtgExprIdentCode(p, ep, e.left)
		if e.argCount == 1 && (callee == rtgIdentInt || callee == rtgIdentByte || callee == rtgIdentInt16 || callee == rtgIdentInt32 || callee == rtgIdentInt64) {
			result := rtgEvalConstExpr(g, ep, ep.args[e.firstArg])
			if result.ok {
				if callee == rtgIdentByte {
					result.value = rtgConvertConstInt(result.value, rtgTypeByte)
				}
				if callee == rtgIdentInt16 {
					result.value = rtgConvertConstInt(result.value, rtgTypeInt16)
				}
				if callee == rtgIdentInt32 {
					result.value = rtgConvertConstInt(result.value, rtgTypeInt32)
				}
			}
			return result
		}
		if e.argCount == 1 {
			calleeExpr := &ep.exprs[e.left]
			if calleeExpr.kind == rtgExprIdent {
				namedType := rtgFindTypeByRange(g, calleeExpr.nameStart, calleeExpr.nameEnd)
				resolved := rtgResolveType(g.meta, namedType)
				if resolved.kind == rtgTypeInt || resolved.kind == rtgTypeInt16 || resolved.kind == rtgTypeInt32 || resolved.kind == rtgTypeInt64 || resolved.kind == rtgTypeBool {
					result := rtgEvalConstExpr(g, ep, ep.args[e.firstArg])
					if result.ok {
						result.value = rtgConvertConstInt(result.value, resolved.kind)
					}
					return result
				}
				if resolved.kind == rtgTypeByte {
					result := rtgEvalConstExpr(g, ep, ep.args[e.firstArg])
					if result.ok {
						result.value = rtgConvertConstInt(result.value, rtgTypeByte)
					}
					return result
				}
			}
		}
		var r rtgConstResult
		return r
	}
	if e.kind == rtgExprUnary {
		inner := rtgEvalConstExpr(g, ep, e.left)
		if !inner.ok {
			var r rtgConstResult
			return r
		}
		if rtgTokCharIs(p, e.tok, '-') {
			return rtgConstResultOk(-inner.value)
		}
		if rtgTokCharIs(p, e.tok, '+') {
			return rtgConstResultOk(inner.value)
		}
		if rtgTokCharIs(p, e.tok, '!') {
			if inner.value == 0 {
				return rtgConstResultOk(1)
			}
			return rtgConstResultOk(0)
		}
		var r rtgConstResult
		return r
	}
	if e.kind == rtgExprBinary {
		opTok := e.tok
		rightIndex := e.right
		rightExpr := &ep.exprs[rightIndex]
		rightKind := rightExpr.kind
		rightTok := rightExpr.tok
		left := rtgEvalConstExpr(g, ep, e.left)
		if !left.ok {
			var r rtgConstResult
			return r
		}
		if rtgTok2Is(p, e.tok, '&', '&') {
			if left.value == 0 {
				return rtgConstResultOk(0)
			}
			right := rtgEvalConstExpr(g, ep, rightIndex)
			if !right.ok {
				var r rtgConstResult
				return r
			}
			if right.value != 0 {
				return rtgConstResultOk(1)
			}
			return rtgConstResultOk(0)
		}
		if rtgTok2Is(p, e.tok, '|', '|') {
			if left.value != 0 {
				return rtgConstResultOk(1)
			}
			right := rtgEvalConstExpr(g, ep, rightIndex)
			if !right.ok {
				var r rtgConstResult
				return r
			}
			if right.value != 0 {
				return rtgConstResultOk(1)
			}
			return rtgConstResultOk(0)
		}
		var right rtgConstResult
		if rightKind == rtgExprInt {
			value := rtgParseIntToken(p, rightTok)
			right = rtgConstResultOk(value)
		} else if rightKind == rtgExprChar {
			value := rtgParseCharToken(p, rightTok)
			right = rtgConstResultOk(value)
		} else if rightKind == rtgExprBool {
			value := rtgBoolTokenValue(p, rightTok)
			right = rtgConstResultOk(value)
		} else {
			right = rtgEvalConstExpr(g, ep, rightIndex)
		}
		if !right.ok {
			var r rtgConstResult
			return r
		}
		result := rtgEvalConstBinary(g, opTok, left.value, right.value)
		return result
	}
	var r rtgConstResult
	return r
}

func rtgEvalConstBinary(g *rtgLinearGen, tok int, left int, right int) rtgConstResult {
	p := g.prog
	if tok < 0 || tok >= len(p.toks) {
		var r rtgConstResult
		return r
	}
	start := p.toks[tok].start
	end := p.toks[tok].end
	n := end - start
	value := 0
	ok := true
	if n == 1 {
		c := p.src[start]
		if c == '+' {
			value = left + right
		} else if c == '-' {
			value = left - right
		} else if c == '*' {
			value = left * right
		} else if c == '/' {
			if right == 0 {
				var r rtgConstResult
				return r
			}
			value = left / right
		} else if c == '%' {
			if right == 0 {
				var r rtgConstResult
				return r
			}
			value = left % right
		} else if c == '&' {
			value = left & right
		} else if c == '|' {
			value = left | right
		} else if c == '^' {
			value = left ^ right
		} else if c == '<' {
			if left < right {
				value = 1
			} else {
				value = 0
			}
		} else if c == '>' {
			if left > right {
				value = 1
			} else {
				value = 0
			}
		} else {
			ok = false
		}
	} else if n == 2 {
		c0 := p.src[start]
		c1 := p.src[start+1]
		if c0 == '&' && c1 == '^' {
			value = left &^ right
		} else if c0 == '<' && c1 == '<' {
			value = left << right
		} else if c0 == '>' && c1 == '>' {
			value = left >> right
		} else if c0 == '=' && c1 == '=' {
			if left == right {
				value = 1
			} else {
				value = 0
			}
		} else if c0 == '!' && c1 == '=' {
			if left != right {
				value = 1
			} else {
				value = 0
			}
		} else if c0 == '<' && c1 == '=' {
			if left <= right {
				value = 1
			} else {
				value = 0
			}
		} else if c0 == '>' && c1 == '=' {
			if left >= right {
				value = 1
			} else {
				value = 0
			}
		} else {
			ok = false
		}
	} else {
		ok = false
	}
	if ok {
		return rtgConstResultOk(value)
	}
	var r rtgConstResult
	return r
}

func rtgExprIsIdentText(p *rtgProgram, ep *rtgExprParse, idx int, text string) bool {
	e := &ep.exprs[idx]
	if e.kind != rtgExprIdent {
		return false
	}
	return rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, text)
}

func rtgParseExpression(p *rtgProgram, start int, end int) rtgExprParse {
	var ep rtgExprParse
	ep.prog = p
	ep.pos = start
	ep.end = end
	ep.ok = true
	rtgParseBinaryExpr(&ep, 1)
	if ep.pos < ep.end {
		rtgExprError(&ep, rtgDiagParseExpression)
	}
	return ep
}

func rtgParseBinaryExpr(ep *rtgExprParse, minPrec int) int {
	left := rtgParseUnaryExpr(ep)
	for ep.ok && ep.pos < ep.end {
		prec := rtgTokenPrecedence(ep.prog, ep.pos)
		if prec < minPrec {
			break
		}
		opTok := ep.pos
		ep.pos++
		right := rtgParseBinaryExpr(ep, prec+1)
		left = rtgAddExpr(ep, rtgExprBinary, opTok, left, right, 0, 0, 0, 0)
	}
	return left
}

func rtgParseUnaryExpr(ep *rtgExprParse) int {
	if ep.pos >= ep.end {
		rtgExprError(ep, rtgDiagParseExpression)
		return 0
	}
	if rtgTokCharIs(ep.prog, ep.pos, '+') || rtgTokCharIs(ep.prog, ep.pos, '-') || rtgTokCharIs(ep.prog, ep.pos, '!') || rtgTokCharIs(ep.prog, ep.pos, '&') || rtgTokCharIs(ep.prog, ep.pos, '*') {
		opTok := ep.pos
		ep.pos++
		inner := rtgParseUnaryExpr(ep)
		return rtgAddExpr(ep, rtgExprUnary, opTok, inner, 0, 0, 0, 0, 0)
	}
	return rtgParsePostfixExpr(ep)
}

func rtgParsePostfixExpr(ep *rtgExprParse) int {
	left := rtgParsePrimaryExpr(ep)
	for ep.ok && ep.pos < ep.end {
		if rtgTokCharIs(ep.prog, ep.pos, '{') {
			base := &ep.exprs[left]
			if base.kind != rtgExprIdent {
				rtgExprError(ep, rtgDiagParseComposite)
				return left
			}
			var compositeFields []rtgCompositeField
			ep.pos++
			for ep.ok && ep.pos < ep.end && !rtgTokCharIs(ep.prog, ep.pos, '}') {
				var field rtgCompositeField
				if rtgTokIsKind(ep.prog, ep.pos, rtgTokIdent) && rtgTokCharIs(ep.prog, ep.pos+1, ':') {
					nameTok := ep.prog.toks[ep.pos]
					ep.pos += 2
					fieldEnd := rtgFindExprBoundary(ep.prog, ep.pos, ep.end)
					oldEnd := ep.end
					ep.end = fieldEnd
					fieldRoot := rtgParseBinaryExpr(ep, 1)
					ep.end = oldEnd
					field.nameStart = nameTok.start
					field.nameEnd = nameTok.end
					field.expr = fieldRoot
					ep.pos = fieldEnd
				} else if rtgTokCharIs(ep.prog, ep.pos, '{') {
					fieldEnd := rtgFindExprBoundary(ep.prog, ep.pos, ep.end)
					oldEnd := ep.end
					ep.end = fieldEnd
					field.expr = rtgParseImplicitCompositeExpr(ep)
					ep.end = oldEnd
					ep.pos = fieldEnd
				} else {
					fieldEnd := rtgFindExprBoundary(ep.prog, ep.pos, ep.end)
					oldEnd := ep.end
					ep.end = fieldEnd
					field.expr = rtgParseBinaryExpr(ep, 1)
					ep.end = oldEnd
					ep.pos = fieldEnd
				}
				compositeFields = append(compositeFields, field)
				if rtgTokCharIs(ep.prog, ep.pos, ',') {
					ep.pos++
				}
			}
			if !rtgTokCharIs(ep.prog, ep.pos, '}') {
				rtgExprError(ep, rtgDiagParseComposite)
				return left
			}
			ep.pos++
			first := len(ep.fields)
			for i := 0; i < len(compositeFields); i++ {
				field := compositeFields[i]
				ep.fields = append(ep.fields, field)
			}
			count := len(compositeFields)
			left = rtgAddExpr(ep, rtgExprComposite, base.tok, 0, 0, first, count, base.nameStart, base.nameEnd)
			continue
		}
		if rtgTokCharIs(ep.prog, ep.pos, '(') {
			callTok := ep.pos
			var callArgs []int
			callExpanded := false
			ep.pos++
			for ep.ok && ep.pos < ep.end && !rtgTokCharIs(ep.prog, ep.pos, ')') {
				argEnd := rtgFindExprBoundary(ep.prog, ep.pos, ep.end)
				if rtgTokCharIs(ep.prog, argEnd, '{') {
					closeTok := rtgSkipBalanced(ep.prog, argEnd, '{', '}')
					if closeTok > argEnd {
						argEnd = closeTok
					}
				}
				parseEnd := argEnd
				if argEnd-ep.pos >= 4 && rtgTokCharIs(ep.prog, argEnd-3, '.') && rtgTokCharIs(ep.prog, argEnd-2, '.') && rtgTokCharIs(ep.prog, argEnd-1, '.') {
					callExpanded = true
					parseEnd = argEnd - 3
				}
				oldEnd := ep.end
				ep.end = parseEnd
				argRoot := rtgParseBinaryExpr(ep, 1)
				ep.end = oldEnd
				callArgs = append(callArgs, argRoot)
				ep.pos = argEnd
				if rtgTokCharIs(ep.prog, ep.pos, ',') {
					ep.pos++
				}
			}
			if !rtgTokCharIs(ep.prog, ep.pos, ')') {
				rtgExprError(ep, rtgDiagParseCall)
				return left
			}
			ep.pos++
			first := len(ep.args)
			for i := 0; i < len(callArgs); i++ {
				ep.args = append(ep.args, callArgs[i])
			}
			count := len(callArgs)
			expanded := 0
			if callExpanded {
				expanded = 1
			}
			left = rtgAddExpr(ep, rtgExprCall, callTok, left, 0, first, count, expanded, 0)
			continue
		}
		if rtgTokCharIs(ep.prog, ep.pos, '[') {
			indexTok := ep.pos
			ep.pos++
			indexStart := ep.pos
			indexEnd := rtgFindMatchingExprClose(ep.prog, ep.pos, ep.end, '[', ']')
			if indexEnd <= ep.pos {
				rtgExprError(ep, rtgDiagParseIndex)
				return left
			}
			colon := rtgFindSliceColon(ep.prog, indexStart, indexEnd)
			if colon >= 0 {
				low := -1
				high := -1
				oldEnd := ep.end
				if colon > indexStart {
					ep.pos = indexStart
					ep.end = colon
					low = rtgParseBinaryExpr(ep, 1)
				}
				if colon+1 < indexEnd {
					ep.pos = colon + 1
					ep.end = indexEnd
					high = rtgParseBinaryExpr(ep, 1)
				}
				ep.end = oldEnd
				ep.pos = indexEnd + 1
				left = rtgAddExpr(ep, rtgExprSlice, indexTok, left, high, low, 0, 0, 0)
				continue
			}
			oldEnd := ep.end
			ep.end = indexEnd
			right := rtgParseBinaryExpr(ep, 1)
			ep.end = oldEnd
			ep.pos = indexEnd + 1
			left = rtgAddExpr(ep, rtgExprIndex, indexTok, left, right, 0, 0, 0, 0)
			continue
		}
		if rtgTokCharIs(ep.prog, ep.pos, '.') && rtgTokIsKind(ep.prog, ep.pos+1, rtgTokIdent) {
			dotTok := ep.pos
			nameTok := ep.prog.toks[ep.pos+1]
			ep.pos += 2
			left = rtgAddExpr(ep, rtgExprSelector, dotTok, left, 0, 0, 0, nameTok.start, nameTok.end)
			continue
		}
		break
	}
	return left
}

func rtgFindSliceColon(p *rtgProgram, start int, end int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && rtgTokCharIs(p, i, ':') {
			return i
		}
		if rtgTokCharIs(p, i, '(') {
			paren++
		} else if rtgTokCharIs(p, i, ')') {
			paren--
		} else if rtgTokCharIs(p, i, '[') {
			brack++
		} else if rtgTokCharIs(p, i, ']') {
			brack--
		} else if rtgTokCharIs(p, i, '{') {
			brace++
		} else if rtgTokCharIs(p, i, '}') {
			brace--
		}
	}
	return -1
}

func rtgParseImplicitCompositeExpr(ep *rtgExprParse) int {
	openTok := ep.pos
	if !rtgTokCharIs(ep.prog, ep.pos, '{') {
		rtgExprError(ep, rtgDiagParseComposite)
		return 0
	}
	var compositeFields []rtgCompositeField
	ep.pos++
	for ep.ok && ep.pos < ep.end && !rtgTokCharIs(ep.prog, ep.pos, '}') {
		if !rtgTokIsKind(ep.prog, ep.pos, rtgTokIdent) || !rtgTokCharIs(ep.prog, ep.pos+1, ':') {
			rtgExprError(ep, rtgDiagParseComposite)
			return 0
		}
		nameTok := ep.prog.toks[ep.pos]
		ep.pos += 2
		fieldEnd := rtgFindExprBoundary(ep.prog, ep.pos, ep.end)
		oldEnd := ep.end
		ep.end = fieldEnd
		fieldRoot := rtgParseBinaryExpr(ep, 1)
		ep.end = oldEnd
		compositeFields = append(compositeFields, rtgCompositeField{nameStart: nameTok.start, nameEnd: nameTok.end, expr: fieldRoot})
		ep.pos = fieldEnd
		if rtgTokCharIs(ep.prog, ep.pos, ',') {
			ep.pos++
		}
	}
	if !rtgTokCharIs(ep.prog, ep.pos, '}') {
		rtgExprError(ep, rtgDiagParseComposite)
		return 0
	}
	ep.pos++
	first := len(ep.fields)
	for i := 0; i < len(compositeFields); i++ {
		field := compositeFields[i]
		ep.fields = append(ep.fields, field)
	}
	count := len(compositeFields)
	return rtgAddExpr(ep, rtgExprComposite, openTok, 0, 0, first, count, 0, 0)
}

func rtgParsePrimaryExpr(ep *rtgExprParse) int {
	if ep.pos >= ep.end {
		rtgExprError(ep, rtgDiagParseExpression)
		return 0
	}
	tok := &ep.prog.toks[ep.pos]
	if rtgTokCharIs(ep.prog, ep.pos, '[') && rtgTokCharIs(ep.prog, ep.pos+1, ']') && rtgTokIsKind(ep.prog, ep.pos+2, rtgTokIdent) {
		startTok := ep.pos
		nameTok := ep.prog.toks[ep.pos+2]
		ep.pos += 3
		return rtgAddExpr(ep, rtgExprIdent, startTok, 0, 0, 0, 0, ep.prog.toks[startTok].start, nameTok.end)
	}
	if tok.kind == rtgTokIdent {
		ep.pos++
		if rtgBytesEqualText(ep.prog.src, tok.start, tok.end, "true") {
			return rtgAddExpr(ep, rtgExprBool, ep.pos-1, 0, 0, 0, 0, 0, 0)
		}
		if rtgBytesEqualText(ep.prog.src, tok.start, tok.end, "false") {
			return rtgAddExpr(ep, rtgExprBool, ep.pos-1, 0, 0, 0, 0, 0, 0)
		}
		return rtgAddExpr(ep, rtgExprIdent, ep.pos-1, 0, 0, 0, 0, tok.start, tok.end)
	}
	if tok.kind == rtgTokNumber {
		ep.pos++
		return rtgAddExpr(ep, rtgExprInt, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if tok.kind == rtgTokFloat {
		ep.pos++
		ep.hasFloat = true
		return rtgAddExpr(ep, rtgExprFloat, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if tok.kind == rtgTokString {
		ep.pos++
		return rtgAddExpr(ep, rtgExprString, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if tok.kind == rtgTokChar {
		ep.pos++
		return rtgAddExpr(ep, rtgExprChar, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if rtgTokCharIs(ep.prog, ep.pos, '(') {
		ep.pos++
		inner := rtgParseBinaryExpr(ep, 1)
		if !rtgTokCharIs(ep.prog, ep.pos, ')') {
			rtgExprError(ep, rtgDiagParseParen)
			return inner
		}
		ep.pos++
		return inner
	}
	rtgExprError(ep, rtgDiagParseExpression)
	return 0
}

func rtgAddExpr(ep *rtgExprParse, kind int, tok int, left int, right int, firstArg int, argCount int, nameStart int, nameEnd int) int {
	var e rtgExpr
	e.kind = kind
	e.tok = tok
	e.left = left
	e.right = right
	e.firstArg = firstArg
	e.argCount = argCount
	e.nameStart = nameStart
	e.nameEnd = nameEnd
	ep.exprs = append(ep.exprs, e)
	index := len(ep.exprs) - 1
	return index
}

func rtgTokenPrecedence(p *rtgProgram, pos int) int {
	if pos < 0 || pos >= len(p.toks) {
		return 0
	}
	start := p.toks[pos].start
	end := p.toks[pos].end
	if end-start == 1 {
		c := p.src[start]
		if c == '<' || c == '>' {
			return 3
		}
		if c == '+' || c == '-' || c == '|' || c == '^' {
			return 4
		}
		if c == '*' || c == '/' || c == '%' || c == '&' {
			return 5
		}
		return 0
	}
	if end-start == 2 {
		c0 := p.src[start]
		c1 := p.src[start+1]
		if c0 == '|' && c1 == '|' {
			return 1
		}
		if c0 == '&' && c1 == '&' {
			return 2
		}
		if (c0 == '=' || c0 == '!' || c0 == '<' || c0 == '>') && c1 == '=' {
			return 3
		}
		if (c0 == '<' && c1 == '<') || (c0 == '>' && c1 == '>') || (c0 == '&' && c1 == '^') {
			return 5
		}
	}
	return 0
}

func rtgFindExprBoundary(p *rtgProgram, start int, end int) int {
	i := start
	paren := 0
	brack := 0
	brace := 0
	for i < end {
		if paren == 0 && brack == 0 && brace == 0 && rtgTokCharIs(p, i, '{') {
			closeTok := rtgSkipBalanced(p, i, '{', '}')
			if closeTok > i {
				i = closeTok
				continue
			}
		}
		if paren == 0 && brack == 0 && brace == 0 && (rtgTokCharIs(p, i, ',') || rtgTokCharIs(p, i, ')') || rtgTokCharIs(p, i, ']') || rtgTokCharIs(p, i, '}')) {
			return i
		}
		if rtgTokCharIs(p, i, '(') {
			paren++
		} else if rtgTokCharIs(p, i, ')') {
			if paren == 0 {
				return i
			}
			paren--
		} else if rtgTokCharIs(p, i, '[') {
			brack++
		} else if rtgTokCharIs(p, i, ']') {
			if brack == 0 {
				return i
			}
			brack--
		} else if rtgTokCharIs(p, i, '{') {
			brace++
		} else if rtgTokCharIs(p, i, '}') {
			if brace == 0 {
				return i
			}
			brace--
		}
		i++
	}
	return i
}

func rtgFindMatchingExprClose(p *rtgProgram, start int, end int, open byte, close byte) int {
	depth := 0
	i := start
	for i < end {
		if rtgTokCharIs(p, i, open) {
			depth++
		} else if rtgTokCharIs(p, i, close) {
			if depth == 0 {
				return i
			}
			depth--
		}
		i++
	}
	return start
}

func rtgParseOneStatement(bp *rtgBodyParse, start int, end int) int {
	p := bp.prog
	if start >= end {
		return end
	}
	if rtgTokIsKind(p, start, rtgTokReturn) {
		exprEnd := rtgStatementLineEnd(p, start+1, end)
		rtgAddStmt(bp, rtgStmtReturn, start, exprEnd, start+1, exprEnd, 0, 0, 0, 0, 0, 0)
		return exprEnd
	}
	if rtgTokIsKind(p, start, rtgTokIf) {
		bodyStart := rtgFindStatementBodyOpen(p, start+1, end)
		if bodyStart <= start {
			return start
		}
		bodyEnd := rtgFindMatchingBrace(p, bodyStart, end)
		if bodyEnd <= bodyStart {
			return start
		}
		stmt := rtgStmt{kind: rtgStmtIf, startTok: start, endTok: bodyEnd + 1, exprStart: start + 1, exprEnd: bodyStart, bodyStart: bodyStart + 1, bodyEnd: bodyEnd}
		next := bodyEnd + 1
		if rtgTokIsKind(p, next, rtgTokElse) {
			if rtgTokIsKind(p, next+1, rtgTokIf) {
				foundEnd := rtgFindIfStatementEnd(p, next+1, end)
				if foundEnd <= next+1 {
					return start
				}
				stmt.elseStart = next + 1
				stmt.elseEnd = foundEnd
				stmt.endTok = foundEnd
				next = foundEnd
			} else if rtgTokCharIs(p, next+1, '{') {
				elseBodyEnd := rtgFindMatchingBrace(p, next+1, end)
				if elseBodyEnd <= next+1 {
					return start
				}
				stmt.elseStart = next + 2
				stmt.elseEnd = elseBodyEnd
				stmt.endTok = elseBodyEnd + 1
				next = elseBodyEnd + 1
			}
		}
		bp.stmts = append(bp.stmts, stmt)
		return next
	}
	if rtgTokIsKind(p, start, rtgTokSwitch) {
		bodyStart := rtgFindStatementBodyOpen(p, start+1, end)
		if bodyStart <= start {
			return start
		}
		bodyEnd := rtgFindMatchingBrace(p, bodyStart, end)
		if bodyEnd <= bodyStart {
			return start
		}
		rtgAddStmt(bp, rtgStmtSwitch, start, bodyEnd+1, start+1, bodyStart, bodyStart+1, bodyEnd, 0, 0, 0, 0)
		return bodyEnd + 1
	}
	if rtgTokIsKind(p, start, rtgTokFor) {
		bodyStart := rtgFindStatementBodyOpen(p, start+1, end)
		if bodyStart <= start {
			return start
		}
		bodyEnd := rtgFindMatchingBrace(p, bodyStart, end)
		if bodyEnd <= bodyStart {
			return start
		}
		rtgAddStmt(bp, rtgStmtFor, start, bodyEnd+1, start+1, bodyStart, bodyStart+1, bodyEnd, 0, 0, 0, 0)
		return bodyEnd + 1
	}
	if rtgTokCharIs(p, start, '{') {
		bodyEnd := rtgFindMatchingBrace(p, start, end)
		if bodyEnd <= start {
			return start
		}
		rtgAddStmt(bp, rtgStmtBlock, start, bodyEnd+1, 0, 0, start+1, bodyEnd, 0, 0, 0, 0)
		return bodyEnd + 1
	}
	if rtgTokIsKind(p, start, rtgTokBreak) {
		endTok := rtgStatementLineEnd(p, start+1, end)
		rtgAddStmt(bp, rtgStmtBreak, start, endTok, 0, 0, 0, 0, 0, 0, 0, 0)
		return endTok
	}
	if rtgTokIsKind(p, start, rtgTokContinue) {
		endTok := rtgStatementLineEnd(p, start+1, end)
		rtgAddStmt(bp, rtgStmtContinue, start, endTok, 0, 0, 0, 0, 0, 0, 0, 0)
		return endTok
	}
	if rtgTokIsKind(p, start, rtgTokGoto) {
		endTok := rtgStatementLineEnd(p, start+1, end)
		nameStart := 0
		nameEnd := 0
		if rtgTokIsKind(p, start+1, rtgTokIdent) {
			nameStart = p.toks[start+1].start
			nameEnd = p.toks[start+1].end
		}
		rtgAddStmt(bp, rtgStmtGoto, start, endTok, 0, 0, 0, 0, 0, 0, nameStart, nameEnd)
		return endTok
	}
	if rtgTokIsKind(p, start, rtgTokIdent) && rtgTokCharIs(p, start+1, ':') {
		name := &p.toks[start]
		rtgAddStmt(bp, rtgStmtLabel, start, start+2, 0, 0, 0, 0, 0, 0, name.start, name.end)
		return start + 2
	}
	if rtgTokIsKind(p, start, rtgTokVar) || rtgTokIsKind(p, start, rtgTokConst) {
		endTok := rtgStatementLineEnd(p, start+1, end)
		nameStart := 0
		nameEnd := 0
		if rtgTokIsKind(p, start+1, rtgTokIdent) {
			nameStart = p.toks[start+1].start
			nameEnd = p.toks[start+1].end
		}
		rtgAddStmt(bp, rtgStmtVar, start, endTok, 0, 0, 0, 0, 0, 0, nameStart, nameEnd)
		return endTok
	}
	lineEnd := rtgStatementLineEnd(p, start, end)
	assignTok := rtgFindAssignmentToken(p, start, lineEnd)
	if assignTok > start {
		kind := rtgStmtAssign
		if rtgTok2Is(p, assignTok, ':', '=') {
			kind = rtgStmtShort
		}
		nameStart := 0
		nameEnd := 0
		if rtgTokIsKind(p, start, rtgTokIdent) {
			nameStart = p.toks[start].start
			nameEnd = p.toks[start].end
		}
		rtgAddStmt(bp, kind, start, lineEnd, assignTok+1, lineEnd, 0, 0, 0, 0, nameStart, nameEnd)
		return lineEnd
	}
	rtgAddStmt(bp, rtgStmtExpr, start, lineEnd, start, lineEnd, 0, 0, 0, 0, 0, 0)
	return lineEnd
}

func rtgAddStmt(bp *rtgBodyParse, kind int, startTok int, endTok int, exprStart int, exprEnd int, bodyStart int, bodyEnd int, elseStart int, elseEnd int, nameStart int, nameEnd int) {
	var stmt rtgStmt
	stmt.kind = kind
	stmt.startTok = startTok
	stmt.endTok = endTok
	stmt.exprStart = exprStart
	stmt.exprEnd = exprEnd
	stmt.bodyStart = bodyStart
	stmt.bodyEnd = bodyEnd
	stmt.elseStart = elseStart
	stmt.elseEnd = elseEnd
	stmt.nameStart = nameStart
	stmt.nameEnd = nameEnd
	bp.stmts = append(bp.stmts, stmt)
}

func rtgFindIfStatementEnd(p *rtgProgram, start int, end int) int {
	if !(rtgTokIsKind(p, start, rtgTokIf)) {
		return start
	}
	bodyStart := rtgFindStatementBodyOpen(p, start+1, end)
	if bodyStart <= start {
		return start
	}
	bodyEnd := rtgFindMatchingBrace(p, bodyStart, end)
	if bodyEnd <= bodyStart {
		return start
	}
	next := bodyEnd + 1
	if rtgTokIsKind(p, next, rtgTokElse) {
		if rtgTokIsKind(p, next+1, rtgTokIf) {
			return rtgFindIfStatementEnd(p, next+1, end)
		}
		if rtgTokCharIs(p, next+1, '{') {
			elseEnd := rtgFindMatchingBrace(p, next+1, end)
			if elseEnd <= next+1 {
				return start
			}
			return elseEnd + 1
		}
	}
	return next
}

func rtgStatementLineEnd(p *rtgProgram, start int, end int) int {
	if start >= end {
		return end
	}
	line := p.toks[start].line
	i := start
	paren := 0
	brack := 0
	brace := 0
	for i < end {
		if i > start && paren == 0 && brack == 0 && brace == 0 {
			if rtgTokIsKind(p, i, rtgTokEOF) {
				return i
			}
			if rtgTokCharIs(p, i, ';') {
				return i
			}
			if p.toks[i].line != line {
				if rtgTokCharIs(p, i, '{') {
					return i
				}
				if rtgTokIsKind(p, i, rtgTokReturn) || rtgTokIsKind(p, i, rtgTokIf) || rtgTokIsKind(p, i, rtgTokFor) || rtgTokIsKind(p, i, rtgTokSwitch) || rtgTokIsKind(p, i, rtgTokCase) || rtgTokIsKind(p, i, rtgTokDefault) || rtgTokIsKind(p, i, rtgTokVar) || rtgTokIsKind(p, i, rtgTokConst) || rtgTokIsKind(p, i, rtgTokBreak) || rtgTokIsKind(p, i, rtgTokContinue) || rtgTokIsKind(p, i, rtgTokGoto) {
					return i
				}
				if rtgLineContinuesAfterPrevToken(p, i) {
					line = p.toks[i].line
				} else {
					return i
				}
			}
		}
		closed := false
		if rtgTokCharIs(p, i, '(') {
			paren++
		} else if rtgTokCharIs(p, i, ')') {
			paren--
			closed = true
		} else if rtgTokCharIs(p, i, '[') {
			brack++
		} else if rtgTokCharIs(p, i, ']') {
			brack--
			closed = true
		} else if rtgTokCharIs(p, i, '{') {
			brace++
		} else if rtgTokCharIs(p, i, '}') {
			if brace == 0 {
				return i
			}
			brace--
			closed = true
		}
		if i > start && p.toks[i].line != line && paren == 0 && brack == 0 && brace == 0 {
			if rtgLineContinuesAfterPrevToken(p, i) {
				line = p.toks[i].line
			} else {
				if closed {
					return i + 1
				}
				return i
			}
		}
		i++
	}
	return i
}

func rtgLineContinuesAfterPrevToken(p *rtgProgram, i int) bool {
	if i <= 0 {
		return false
	}
	prev := i - 1
	tokStart := p.toks[prev].start
	tokEnd := p.toks[prev].end
	if tokEnd <= tokStart {
		return false
	}
	c := p.src[tokStart]
	if c == '*' || c == '&' {
		return true
	}
	if c == '+' {
		if tokEnd == tokStart+1 || p.src[tokStart+1] != '+' {
			return true
		}
	}
	return false
}

func rtgFindNextTokenText(p *rtgProgram, start int, end int, text byte) int {
	i := start
	for i < end {
		if rtgTokCharIs(p, i, text) {
			return i
		}
		i++
	}
	return start
}

func rtgFindStatementBodyOpen(p *rtgProgram, start int, end int) int {
	i := start
	paren := 0
	brack := 0
	for i < end {
		tok := &p.toks[i]
		if tok.end == tok.start+1 {
			c := p.src[tok.start]
			if c == '(' {
				paren++
			} else if c == ')' {
				if paren > 0 {
					paren--
				}
			} else if c == '[' {
				brack++
			} else if c == ']' {
				if brack > 0 {
					brack--
				}
			} else if c == '{' {
				if paren == 0 && brack == 0 {
					return i
				}
				closeTok := rtgSkipBalanced(p, i, '{', '}')
				if closeTok > i {
					i = closeTok
					continue
				}
			}
		}
		i++
	}
	return start
}

func rtgFindMatchingBrace(p *rtgProgram, openTok int, end int) int {
	if !rtgTokCharIs(p, openTok, '{') {
		return openTok
	}
	depth := 1
	i := openTok + 1
	for i < end {
		if rtgTokCharIs(p, i, '{') {
			depth++
		} else if rtgTokCharIs(p, i, '}') {
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return openTok
}

func rtgFindAssignmentToken(p *rtgProgram, start int, end int) int {
	i := start
	paren := 0
	brack := 0
	for i < end {
		if rtgTokCharIs(p, i, '(') {
			paren++
		} else if rtgTokCharIs(p, i, ')') {
			paren--
		} else if rtgTokCharIs(p, i, '[') {
			brack++
		} else if rtgTokCharIs(p, i, ']') {
			brack--
		} else if paren == 0 && brack == 0 {
			if rtgTokCharIs(p, i, '=') || rtgTok2Is(p, i, ':', '=') || rtgTok2Is(p, i, '+', '=') || rtgTok2Is(p, i, '-', '=') || rtgTok2Is(p, i, '*', '=') || rtgTok2Is(p, i, '/', '=') || rtgTok2Is(p, i, '%', '=') {
				return i
			}
		}
		i++
	}
	return start
}

func rtgBuildMeta(pp *rtgProgram) rtgMeta {
	p := pp
	var m rtgMeta
	m.prog = p
	m.ok = true
	rtgInitBuiltinTypes(&m)

	for i := 0; i < len(p.decls); i++ {
		decl := p.decls[i]
		if decl.kind != rtgTokType && decl.kind != rtgTokVar && decl.kind != rtgTokConst {
			continue
		}
		entryStart := decl.startTok + 1
		if rtgTokCharIs(p, entryStart, '(') {
			groupEnd := decl.endTok
			if decl.kind == rtgTokConst {
				rtgParseConstDecls(&m, p, entryStart+1, groupEnd-1)
				continue
			}
			j := entryStart + 1
			for j < groupEnd-1 {
				if rtgTokIsKind(p, j, rtgTokIdent) {
					entryEnd := rtgStatementLineEnd(p, j, groupEnd-1)
					rtgParseTopDeclEntry(&m, p, decl.kind, j, entryEnd)
					if entryEnd <= j {
						j++
					} else {
						j = entryEnd
					}
				} else {
					j++
				}
			}
			continue
		}
		if decl.kind == rtgTokConst {
			rtgParseConstDecls(&m, p, entryStart, decl.endTok)
		} else {
			rtgParseTopDeclEntry(&m, p, decl.kind, entryStart, decl.endTok)
		}
	}
	for i := 0; i < len(p.funcs); i++ {
		rtgParseFuncInfo(&m, i)
	}

	return m
}

func rtgInitBuiltinTypes(m *rtgMeta) {
	rtgAddBuiltinType(m, rtgTypeInvalid, 0)
	rtgAddBuiltinType(m, rtgTypeInt, rtgNativeIntSize)
	rtgAddBuiltinType(m, rtgTypeInt64, 8)
	rtgAddBuiltinType(m, rtgTypeByte, 1)
	rtgAddBuiltinType(m, rtgTypeBool, 1)
	rtgAddBuiltinType(m, rtgTypeString, 16)
	rtgAddBuiltinType(m, rtgTypeFloat64, 8)
	rtgAddBuiltinType(m, rtgTypeInt16, 2)
	rtgAddBuiltinType(m, rtgTypeInt32, 4)
}

func rtgAddBuiltinType(m *rtgMeta, kind int, size int) {
	m.types = append(m.types, rtgTypeInfo{kind: kind, size: size})
}

func rtgParseConstDecls(m *rtgMeta, p *rtgProgram, start int, end int) {
	prevTypeStart := 0
	prevTypeEnd := 0
	var prevValues []int
	iotaValue := 0
	j := start
	for j < end {
		if !rtgTokIsKind(p, j, rtgTokIdent) {
			j++
			continue
		}
		specEnd := rtgStatementLineEnd(p, j, end)
		if specEnd <= j {
			rtgMetaError(m, rtgDiagMetaConstDecl)
			return
		}
		eq := rtgFindConstSpecEqual(p, j, specEnd)
		headEnd := specEnd
		if eq > j {
			headEnd = eq
		}
		var names []int
		k := j
		for k < headEnd {
			if !rtgTokIsKind(p, k, rtgTokIdent) {
				break
			}
			names = append(names, k)
			k++
			if rtgTokCharIs(p, k, ',') {
				k++
				continue
			}
			break
		}
		if len(names) == 0 {
			rtgMetaError(m, rtgDiagMetaConstDecl)
			return
		}
		if eq > j {
			prevTypeStart = k
			prevTypeEnd = headEnd
			var newValues []int
			newValues = rtgSplitTopLevelComma(p, eq+1, specEnd, newValues)
			prevValues = newValues
		}
		valueCount := len(prevValues) / 2
		if valueCount == 0 {
			rtgMetaError(m, rtgDiagMetaConstDecl)
			return
		}
		if valueCount != len(names) {
			rtgMetaError(m, rtgDiagMetaConstDecl)
			return
		}
		typ := 0
		if prevTypeStart < prevTypeEnd {
			typeResult := rtgParseType(m, p, prevTypeStart, prevTypeEnd)
			typ = typeResult.typ
		}
		for i := 0; i < len(names); i++ {
			nameTok := names[i]
			name := &p.toks[nameTok]
			if rtgBytesEqualText(p.src, name.start, name.end, "_") {
				continue
			}
			initStart := prevValues[i*2]
			initEnd := prevValues[i*2+1]
			constType := typ
			if constType == 0 {
				constType = rtgInferTopLiteralType(m, p, initStart, initEnd)
			}
			if constType == 0 {
				constType = rtgTypeInt
			}
			var sym rtgSymbolInfo
			sym.nameStart = name.start
			sym.nameEnd = name.end
			sym.kind = rtgTokConst
			sym.typ = constType
			sym.initStart = initStart
			sym.initEnd = initEnd
			sym.iotaValue = iotaValue
			constResult := rtgEvalMetaConstExpr(m, p, initStart, initEnd, iotaValue)
			if constResult.ok {
				sym.constValue = constResult.value
				sym.constValueOK = 1
			}
			m.globals = append(m.globals, sym)
		}
		iotaValue++
		j = specEnd
	}
}

func rtgEvalMetaConstExpr(m *rtgMeta, p *rtgProgram, start int, end int, iotaValue int) rtgConstResult {
	ep := rtgParseExpression(p, start, end)
	if !ep.ok || len(ep.exprs) == 0 {
		var r rtgConstResult
		return r
	}
	rootIndex := len(ep.exprs) - 1
	return rtgEvalMetaParsedConstExpr(m, p, &ep, rootIndex, iotaValue)
}

func rtgEvalMetaParsedConstExpr(m *rtgMeta, p *rtgProgram, ep *rtgExprParse, idx int, iotaValue int) rtgConstResult {
	e := &ep.exprs[idx]
	if e.kind == rtgExprInt {
		return rtgConstResultOk(rtgParseIntToken(p, e.tok))
	}
	if e.kind == rtgExprFloat {
		return rtgConstResultOk(rtgParseFloatTokenScaled(p, e.tok))
	}
	if e.kind == rtgExprChar {
		return rtgConstResultOk(rtgParseCharToken(p, e.tok))
	}
	if e.kind == rtgExprBool {
		return rtgConstResultOk(rtgBoolTokenValue(p, e.tok))
	}
	if e.kind == rtgExprIdent {
		if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "iota") {
			return rtgConstResultOk(iotaValue)
		}
		for i := 0; i < len(m.globals); i++ {
			s := &m.globals[i]
			if s.kind == rtgTokConst && s.constValueOK != 0 && rtgBytesEqualRange(p.src, s.nameStart, s.nameEnd, e.nameStart, e.nameEnd) {
				return rtgConstResultOk(s.constValue)
			}
		}
		var r rtgConstResult
		return r
	}
	if e.kind == rtgExprCall {
		if e.argCount == 1 {
			result := rtgEvalMetaParsedConstExpr(m, p, ep, ep.args[e.firstArg], iotaValue)
			if result.ok {
				callee := rtgExprIdentCode(p, ep, e.left)
				if callee == rtgIdentByte {
					result.value = rtgConvertConstInt(result.value, rtgTypeByte)
				}
				if callee == rtgIdentInt16 {
					result.value = rtgConvertConstInt(result.value, rtgTypeInt16)
				}
				if callee == rtgIdentInt32 {
					result.value = rtgConvertConstInt(result.value, rtgTypeInt32)
				}
			}
			return result
		}
		var r rtgConstResult
		return r
	}
	if e.kind == rtgExprUnary {
		inner := rtgEvalMetaParsedConstExpr(m, p, ep, e.left, iotaValue)
		if !inner.ok {
			var r rtgConstResult
			return r
		}
		if rtgTokCharIs(p, e.tok, '-') {
			return rtgConstResultOk(-inner.value)
		}
		if rtgTokCharIs(p, e.tok, '+') {
			return rtgConstResultOk(inner.value)
		}
		if rtgTokCharIs(p, e.tok, '!') {
			if inner.value == 0 {
				return rtgConstResultOk(1)
			}
			return rtgConstResultOk(0)
		}
		var r rtgConstResult
		return r
	}
	if e.kind == rtgExprBinary {
		left := rtgEvalMetaParsedConstExpr(m, p, ep, e.left, iotaValue)
		if !left.ok {
			var r rtgConstResult
			return r
		}
		right := rtgEvalMetaParsedConstExpr(m, p, ep, e.right, iotaValue)
		if !right.ok {
			var r rtgConstResult
			return r
		}
		var g rtgLinearGen
		g.prog = p
		return rtgEvalConstBinary(&g, e.tok, left.value, right.value)
	}
	var r rtgConstResult
	return r
}

func rtgFindConstSpecEqual(p *rtgProgram, start int, end int) int {
	paren := 0
	brack := 0
	brace := 0
	i := start
	for i < end {
		if rtgTokCharIs(p, i, '(') {
			paren++
		} else if rtgTokCharIs(p, i, ')') {
			if paren > 0 {
				paren--
			}
		} else if rtgTokCharIs(p, i, '[') {
			brack++
		} else if rtgTokCharIs(p, i, ']') {
			if brack > 0 {
				brack--
			}
		} else if rtgTokCharIs(p, i, '{') {
			brace++
		} else if rtgTokCharIs(p, i, '}') {
			if brace > 0 {
				brace--
			}
		} else if paren == 0 && brack == 0 && brace == 0 && rtgTokCharIs(p, i, '=') {
			return i
		}
		i++
	}
	return start
}

func rtgParseTopDeclEntry(m *rtgMeta, p *rtgProgram, kind int, start int, end int) {
	if start >= end || !rtgTokIsKind(p, start, rtgTokIdent) {
		rtgMetaError(m, rtgDiagMetaTopDecl)
		return
	}
	name := &p.toks[start]
	if kind == rtgTokType {
		typeStart := start + 1
		if rtgTokCharIs(p, typeStart, '=') {
			typeStart++
		}
		typeResult := rtgParseType(m, p, typeStart, end)
		if typeResult.typ == 0 || typeResult.next > end {
			rtgMetaError(m, rtgDiagMetaTopDecl)
			return
		}
		if m.types[typeResult.typ].kind == rtgTypeStruct || m.types[typeResult.typ].kind == rtgTypePointer || m.types[typeResult.typ].kind == rtgTypeSlice {
			m.types[typeResult.typ].nameStart = name.start
			m.types[typeResult.typ].nameEnd = name.end
		} else {
			size := rtgTypeSize(m, typeResult.typ)
			rtgAddType(m, rtgTypeNamed, typeResult.typ, 0, 0, size, name.start, name.end)
		}
		return
	}
	eq := start
	j := start + 1
	for j < end {
		if j >= 0 && j < len(p.toks) {
			tok := &p.toks[j]
			if tok.kind == rtgTokOp && tok.end-tok.start == 1 && p.src[tok.start] == '=' {
				eq = j
				j = end
				continue
			}
		}
		j++
	}
	typeEnd := end
	initStart := end
	initEnd := end
	if eq > start {
		typeEnd = eq
		initStart = eq + 1
		initEnd = end
	}
	typ := 0
	if start+1 < typeEnd {
		typeResult := rtgParseType(m, p, start+1, typeEnd)
		typ = typeResult.typ
	}
	if typ == 0 && initStart < initEnd {
		typ = rtgInferTopLiteralType(m, p, initStart, initEnd)
	}
	m.globals = append(m.globals, rtgSymbolInfo{nameStart: name.start, nameEnd: name.end, kind: kind, typ: typ, initStart: initStart, initEnd: initEnd})
}

func rtgInferTopLiteralType(m *rtgMeta, p *rtgProgram, start int, end int) int {
	if start+1 == end && rtgTokIsKind(p, start, rtgTokString) {
		return rtgTypeString
	}
	if start+1 == end && rtgTokIsKind(p, start, rtgTokFloat) {
		return rtgTypeFloat64
	}
	open := start
	depth := 0
	for open < end {
		if depth == 0 && rtgTokCharIs(p, open, '{') {
			typeResult := rtgParseType(m, p, start, open)
			if typeResult.typ != 0 {
				return typeResult.typ
			}
			return 0
		}
		if rtgTokCharIs(p, open, '(') || rtgTokCharIs(p, open, '[') {
			depth++
		} else if rtgTokCharIs(p, open, ')') || rtgTokCharIs(p, open, ']') {
			if depth > 0 {
				depth--
			}
		}
		open++
	}
	return 0
}

func rtgParseFuncInfo(m *rtgMeta, fnIndex int) {
	p := m.prog
	fn := p.funcs[fnIndex]
	nameStart := fn.nameStart
	nameEnd := fn.nameEnd
	nameTok := fn.nameTok
	if nameTok <= fn.startTok {
		rtgMetaError(m, rtgDiagMetaFuncDecl)
		return
	}
	lparen := rtgFindNextTokenText(p, nameTok+1, fn.bodyStart, '(')
	if lparen <= nameTok {
		rtgMetaError(m, rtgDiagMetaFuncDecl)
		return
	}
	rparen := rtgFindMatchingExprClose(p, lparen+1, fn.bodyStart, '(', ')')
	if rparen <= lparen {
		rtgMetaError(m, rtgDiagMetaFuncDecl)
		return
	}
	firstParam := len(m.params)
	paramCount := 0
	receiverType := 0
	if fn.receiverStart < fn.receiverEnd {
		beforeReceiver := len(m.params)
		rtgParseParamList(m, p, fn.receiverStart, fn.receiverEnd, &paramCount)
		if len(m.params) <= beforeReceiver {
			rtgMetaError(m, rtgDiagMetaFuncDecl)
			return
		}
		receiverType = m.params[beforeReceiver].typ
	}
	rtgParseParamList(m, p, lparen+1, rparen, &paramCount)
	resultType := 0
	if rparen+1 < fn.bodyStart {
		resultType = rtgParseFuncResultType(m, p, rparen+1, fn.bodyStart)
	}
	m.funcs = append(m.funcs, rtgFuncInfo{declIndex: fnIndex, nameStart: nameStart, nameEnd: nameEnd, firstParam: firstParam, paramCount: paramCount, resultType: resultType, receiverType: receiverType, bodyStart: fn.bodyStart + 1, bodyEnd: fn.bodyEnd})
}

func rtgParseFuncResultType(m *rtgMeta, p *rtgProgram, start int, end int) int {
	if rtgTokCharIs(p, start, '(') {
		closeTok := rtgFindMatchingExprClose(p, start+1, end, '(', ')')
		if closeTok > start && closeTok <= end {
			var parts []int
			parts = rtgSplitTopLevelComma(p, start+1, closeTok, parts)
			count := len(parts) / 2
			if count > 1 {
				return rtgBuildTupleType(m, p, parts)
			}
			if count == 1 {
				typeResult := rtgParseType(m, p, parts[0], parts[1])
				return typeResult.typ
			}
		}
	}
	typeResult := rtgParseType(m, p, start, end)
	return typeResult.typ
}

func rtgBuildTupleType(m *rtgMeta, p *rtgProgram, parts []int) int {
	firstField := len(m.fields)
	count := len(parts) / 2
	offset := 0
	for i := 0; i < count; i++ {
		typeStart := parts[i*2]
		typeEnd := parts[i*2+1]
		typeResult := rtgParseType(m, p, typeStart, typeEnd)
		if typeResult.typ == 0 {
			rtgMetaError(m, rtgDiagMetaResultType)
			return 0
		}
		offset = rtgAlignTo8(offset)
		m.fields = append(m.fields, rtgFieldInfo{typ: typeResult.typ, offset: offset})
		fieldSize := rtgTypeSize(m, typeResult.typ)
		if fieldSize < 8 {
			fieldSize = 8
		}
		offset += fieldSize
	}
	size := rtgAlignTo8(offset)
	return rtgAddType(m, rtgTypeStruct, 0, firstField, count, size, 0, 0)
}

func rtgParseParamList(m *rtgMeta, p *rtgProgram, start int, end int, count *int) {
	i := start
	for i < end {
		for i < end && rtgTokCharIs(p, i, ',') {
			i++
		}
		if i >= end {
			return
		}
		if !rtgTokIsKind(p, i, rtgTokIdent) {
			rtgMetaError(m, rtgDiagMetaParamList)
			return
		}
		name := &p.toks[i]
		typeStart := i + 1
		entryEnd := typeStart
		depth := 0
		for entryEnd < end {
			if depth == 0 && rtgTokCharIs(p, entryEnd, ',') {
				break
			}
			if rtgTokCharIs(p, entryEnd, '[') || rtgTokCharIs(p, entryEnd, '{') || rtgTokCharIs(p, entryEnd, '(') {
				depth++
			} else if rtgTokCharIs(p, entryEnd, ']') || rtgTokCharIs(p, entryEnd, '}') || rtgTokCharIs(p, entryEnd, ')') {
				depth--
			}
			entryEnd++
		}
		variadic := 0
		if rtgTokCharIs(p, typeStart, '.') && rtgTokCharIs(p, typeStart+1, '.') && rtgTokCharIs(p, typeStart+2, '.') {
			variadic = 1
		}
		typeResult := rtgParseType(m, p, typeStart, entryEnd)
		if typeResult.typ == 0 {
			rtgMetaError(m, rtgDiagMetaParamList)
			return
		}
		m.params = append(m.params, rtgSymbolInfo{nameStart: name.start, nameEnd: name.end, typ: typeResult.typ, initStart: variadic})
		*count = *count + 1
		i = entryEnd
		if rtgTokCharIs(p, i, ',') {
			i++
		}
	}
}

func rtgParseType(m *rtgMeta, p *rtgProgram, start int, end int) rtgTypeResult {
	if start >= end {
		return rtgTypeResult{next: start}
	}
	if rtgTokCharIs(p, start, '.') && rtgTokCharIs(p, start+1, '.') && rtgTokCharIs(p, start+2, '.') {
		elem := rtgParseType(m, p, start+3, end)
		if elem.typ == 0 {
			return rtgTypeResult{next: start}
		}
		typ := rtgAddType(m, rtgTypeSlice, elem.typ, 0, 0, 24, 0, 0)
		return rtgTypeResult{typ: typ, next: elem.next}
	}
	if rtgTokCharIs(p, start, '*') {
		elem := rtgParseType(m, p, start+1, end)
		if elem.typ == 0 {
			return rtgTypeResult{next: start}
		}
		typ := rtgAddType(m, rtgTypePointer, elem.typ, 0, 0, 8, 0, 0)
		return rtgTypeResult{typ: typ, next: elem.next}
	}
	if rtgTokCharIs(p, start, '[') && rtgTokCharIs(p, start+1, ']') {
		elem := rtgParseType(m, p, start+2, end)
		if elem.typ == 0 {
			return rtgTypeResult{next: start}
		}
		typ := rtgAddType(m, rtgTypeSlice, elem.typ, 0, 0, 24, 0, 0)
		return rtgTypeResult{typ: typ, next: elem.next}
	}
	if rtgTokIsKind(p, start, rtgTokStruct) && rtgTokCharIs(p, start+1, '{') {
		closeTok := rtgFindMatchingBrace(p, start+1, end)
		if closeTok <= start+1 {
			return rtgTypeResult{next: start}
		}
		firstField := len(m.fields)
		count := 0
		offset := 0
		i := start + 2
		for i < closeTok {
			if rtgTokIsKind(p, i, rtgTokIdent) {
				name := &p.toks[i]
				lineEnd := rtgStatementLineEnd(p, i, closeTok)
				fieldType := rtgParseType(m, p, i+1, lineEnd)
				if fieldType.typ == 0 {
					return rtgTypeResult{next: start}
				}
				offset = rtgAlignTo8(offset)
				m.fields = append(m.fields, rtgFieldInfo{nameStart: name.start, nameEnd: name.end, typ: fieldType.typ, offset: offset})
				offset += rtgTypeSize(m, fieldType.typ)
				count++
				i = lineEnd
			} else {
				i++
			}
		}
		size := rtgAlignTo8(offset)
		typ := rtgAddType(m, rtgTypeStruct, 0, firstField, count, size, 0, 0)
		return rtgTypeResult{typ: typ, next: closeTok + 1}
	}
	if rtgTokIsKind(p, start, rtgTokIdent) {
		builtin := rtgBuiltinTypeFromToken(p, start)
		if builtin != 0 {
			return rtgTypeResult{typ: builtin, next: start + 1}
		}
		return rtgTypeResult{typ: rtgNamedTypeFromToken(m, p, start), next: start + 1}
	}
	return rtgTypeResult{next: start}
}

func rtgBuiltinTypeFromToken(p *rtgProgram, tokIndex int) int {
	tok := &p.toks[tokIndex]
	if rtgBytesEqualText(p.src, tok.start, tok.end, "int") {
		return rtgTypeInt
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "int16") {
		return rtgTypeInt16
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "int32") {
		return rtgTypeInt32
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "int64") {
		return rtgTypeInt64
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "byte") {
		return rtgTypeByte
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "bool") {
		return rtgTypeBool
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "string") {
		return rtgTypeString
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "float64") {
		return rtgTypeFloat64
	}
	return 0
}

func rtgNamedTypeFromToken(m *rtgMeta, p *rtgProgram, tokIndex int) int {
	tok := &p.toks[tokIndex]
	for i := 0; i < len(m.types); i++ {
		if m.types[i].nameEnd > m.types[i].nameStart && rtgBytesEqualRange(p.src, m.types[i].nameStart, m.types[i].nameEnd, tok.start, tok.end) {
			return i
		}
	}
	return rtgAddType(m, rtgTypeNamed, 0, 0, 0, 8, tok.start, tok.end)
}

func rtgAddType(m *rtgMeta, kind int, elem int, first int, count int, size int, nameStart int, nameEnd int) int {
	m.types = append(m.types, rtgTypeInfo{kind: kind, elem: elem, first: first, count: count, size: size, nameStart: nameStart, nameEnd: nameEnd})
	index := len(m.types) - 1
	return index
}

func rtgTypeSize(m *rtgMeta, typ int) int {
	t := rtgResolveType(m, typ)
	if t.size > 0 {
		return t.size
	}
	return 8
}

func rtgResolveType(m *rtgMeta, typ int) rtgTypeInfo {
	if typ >= 0 && typ < len(m.types) {
		t := m.types[typ]
		if t.kind == rtgTypeNamed && t.elem > 0 && t.elem < len(m.types) {
			return m.types[t.elem]
		}
		if t.kind == rtgTypeNamed && t.elem == 0 && t.nameEnd > t.nameStart {
			for i := 0; i < len(m.types); i++ {
				other := m.types[i]
				if i != typ && other.nameEnd > other.nameStart && rtgBytesEqualRange(m.prog.src, other.nameStart, other.nameEnd, t.nameStart, t.nameEnd) {
					if other.kind != rtgTypeNamed || other.elem > 0 {
						return rtgResolveType(m, i)
					}
				}
			}
		}
		return t
	}
	var t rtgTypeInfo
	return t
}

func rtgTypeIsSlice(m *rtgMeta, typ int) bool {
	t := rtgResolveType(m, typ)
	return t.kind == rtgTypeSlice
}

func rtgTypeIsStringSlice(m *rtgMeta, typ int) bool {
	t := rtgResolveType(m, typ)
	if t.kind != rtgTypeSlice {
		return false
	}
	return rtgTypeIsString(m, t.elem)
}

func rtgTypeIsString(m *rtgMeta, typ int) bool {
	t := rtgResolveType(m, typ)
	return t.kind == rtgTypeString
}

func rtgTypeIsInt(m *rtgMeta, typ int) bool {
	t := rtgResolveType(m, typ)
	return t.kind == rtgTypeInt
}

func rtgTypeKindIsScalarInt(kind int) bool {
	if kind > rtgTypeInvalid && kind < rtgTypeString {
		return true
	}
	return kind == rtgTypeInt16 || kind == rtgTypeInt32
}

func rtgScalarKindSize(kind int) int {
	if kind == rtgTypeByte || kind == rtgTypeBool {
		return 1
	}
	if kind == rtgTypeInt {
		return rtgNativeIntSize
	}
	if kind >= rtgTypeInt16 {
		return (kind - 6) * 2
	}
	return 8
}

func rtgTypeIsStruct(m *rtgMeta, typ int) bool {
	t := rtgResolveType(m, typ)
	return t.kind == rtgTypeStruct
}

func rtgTypeIsTuple(m *rtgMeta, typ int) bool {
	t := rtgResolveType(m, typ)
	if t.kind != rtgTypeStruct || t.count <= 1 {
		return false
	}
	for i := 0; i < t.count; i++ {
		field := m.fields[t.first+i]
		if field.nameEnd > field.nameStart {
			return false
		}
	}
	return true
}

func rtgAlignTo8(v int) int {
	rem := v % 8
	if rem == 0 {
		return v
	}
	return v + 8 - rem
}

func rtgFindTokenTextInRange(p *rtgProgram, start int, end int, text byte) int {
	i := start
	for i < end {
		if rtgTokCharIs(p, i, text) {
			return i
		}
		i++
	}
	return start - 1
}

func rtgBytesEqualRange(src []byte, aStart int, aEnd int, bStart int, bEnd int) bool {
	if aEnd-aStart != bEnd-bStart {
		return false
	}
	for i := 0; i < aEnd-aStart; i++ {
		if src[aStart+i] != src[bStart+i] {
			return false
		}
	}
	return true
}
