package main

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
const rtgTypePointer = 7
const rtgTypeSlice = 8
const rtgTypeStruct = 9
const rtgTypeNamed = 10

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

const rtgRel32 = 1

type rtgLabelRef struct {
	at    int
	label int
	kind  int
}

type rtgDataRef struct {
	at  int
	off int
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
	dataRelocs []rtgDataRef
	absRelocs  []rtgAbsRef
	data       []byte
	bssSize    int
	codeOffset int
	dataOffset int
}

type rtgLocalInfo struct {
	nameStart int
	nameEnd   int
	offset    int
	typ       int
	size      int
}

type rtgGlobalInfo struct {
	nameStart int
	nameEnd   int
	offset    int
}

type rtgSliceLocation struct {
	offset int
	typ    int
	expr   int
	mem    bool
	global bool
	ok     bool
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

type rtgLinearGen struct {
	prog               *rtgProgram
	meta               *rtgMeta
	asm                rtgAsm
	funcLabels         []int
	currentFunc        int
	returnStruct       int
	locals             []rtgLocalInfo
	stackUsed          int
	globals            []rtgGlobalInfo
	gotoLabels         []rtgGlobalInfo
	breakLabels        []int
	continueLabels     []int
	breakDepth         int
	continueDepth      int
	envDataOff         int
	envLenOff          int
	streqLabel         int
	streqEmitted       bool
	append8Label       int
	append8Emitted     bool
	append64Label      int
	append64Emitted    bool
	appendAddrLabel    int
	appendAddrEmitted  bool
	appendBytesLabel   int
	appendBytesEmitted bool
	copyWordsLabel     int
	copyWordsEmitted   bool
	lastRangeReturns   bool
	scopeBase          int
	constEvalIota      int
	constEvalIotaValid int
}

const rtgIdentAppend = 1
const rtgIdentByteSlice = 2
const rtgIdentMake = 3
const rtgIdentRtgEnv = 4
const rtgIdentRtgParseProgram = 5
const rtgIdentInt = 6
const rtgIdentInt64 = 7
const rtgIdentByte = 8
const rtgIdentLen = 9
const rtgIdentOpen = 10
const rtgIdentClose = 11
const rtgIdentRead = 12
const rtgIdentWrite = 13
const rtgIdentChmod = 14
const rtgIdentCopy = 15

func compileLinuxAmd64(input []int, output int) int {
	var src []byte
	for i := 0; i < len(input); i++ {
		src = rtgReadAll(input[i], src)
		src = append(src, '\n')
	}
	var prog rtgProgram
	prog = rtgParseProgram(src)
	if !prog.ok {
		return 1
	}
	var meta rtgMeta
	meta = rtgBuildMeta(&prog)
	if !meta.ok {
		return 1
	}
	var result rtgCompileResult
	result = rtgTryCompileScalarProgram(&prog, &meta)
	if result.ok {
		write(output, result.data, 0)
		return 0
	}
	return 1
}

func rtgReadAll(fd int, out []byte) []byte {
	buf := make([]byte, 32768)
	for {
		n := read(fd, buf, -1)
		if n <= 0 {
			return out
		}
		out = append(out, buf[:n]...)
	}
}

func rtgParseProgram(src []byte) rtgProgram {
	var p rtgProgram
	p.src = src
	p.toks = rtgScan(src)
	p.ok = true

	i := 0
	if !rtgTokIsKind(&p, i, rtgTokPackage) {
		p.ok = false
		return p
	}
	i++
	if !rtgTokIsKind(&p, i, rtgTokIdent) {
		p.ok = false
		return p
	}
	i++

	for i < len(p.toks) && p.toks[i].kind != rtgTokEOF {
		if rtgTokIsKind(&p, i, rtgTokPackage) {
			i++
			if !rtgTokIsKind(&p, i, rtgTokIdent) {
				p.ok = false
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
					p.ok = false
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
				p.ok = false
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
				p.ok = false
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
	if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "rtgEnv") {
		return rtgIdentRtgEnv
	}
	if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "rtgParseProgram") {
		return rtgIdentRtgParseProgram
	}
	if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "int") {
		return rtgIdentInt
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

func rtgAddStringData(g *rtgLinearGen, msg []byte) int {
	msgOff := len(g.asm.data)
	for i := 0; i < len(msg); i++ {
		g.asm.data = append(g.asm.data, msg[i])
	}
	g.asm.data = append(g.asm.data, 0)
	return msgOff
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

func rtgTryCompileScalarProgram(p *rtgProgram, meta *rtgMeta) rtgCompileResult {
	appIndex := -1
	for i := 0; i < len(meta.funcs); i++ {
		if rtgBytesEqualText(meta.prog.src, meta.funcs[i].nameStart, meta.funcs[i].nameEnd, "appMain") {
			appIndex = i
		}
	}
	if appIndex < 0 {
		var result rtgCompileResult
		return result
	}
	var g rtgLinearGen
	g.prog = p
	g.meta = meta
	a := &g.asm
	rtgAsmInit(a)
	for i := 0; i < len(meta.funcs); i++ {
		g.funcLabels = append(g.funcLabels, rtgAsmNewLabel(a))
	}
	if !rtgLinearInitGlobals(&g) {
		var result rtgCompileResult
		return result
	}
	if !rtgEmitProgramEntryArgs(&g, appIndex) {
		var result rtgCompileResult
		return result
	}
	rtgAsmCallLabel(a, g.funcLabels[appIndex])
	rtgAsmMovRdiRax(a)
	rtgAsmMovRaxImm(a, 60)
	rtgAsmSyscall(a)
	for i := 0; i < len(meta.funcs); i++ {
		if rtgBytesEqualText(meta.prog.src, meta.funcs[i].nameStart, meta.funcs[i].nameEnd, "rtgEnv") {
			continue
		}
		if !rtgEmitScalarFunction(&g, i) {
			var result rtgCompileResult
			return result
		}
	}
	data := rtgAsmImage(a)
	var result rtgCompileResult
	result.data = data
	result.ok = true
	return result
}

func rtgEmitProgramEntryArgs(g *rtgLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	argsOff := g.asm.bssSize
	g.asm.bssSize += 32768
	g.envDataOff = g.asm.bssSize
	g.asm.bssSize += 32768
	g.envLenOff = g.asm.bssSize
	g.asm.bssSize += 8
	rtgAsmBuildArgvEnvSlices(&g.asm, argsOff, g.envDataOff, g.envLenOff)
	if app.paramCount == 0 {
		return true
	}
	first := &g.meta.params[app.firstParam]
	if !rtgTypeIsSlice(g.meta, first.typ) {
		return false
	}
	return true
}

func rtgEmitScalarFunction(g *rtgLinearGen, fnInfoIndex int) bool {
	a := &g.asm
	metaFn := &g.meta.funcs[fnInfoIndex]
	fn := &g.prog.funcs[metaFn.declIndex]
	oldLocals := g.locals
	oldBreak := g.breakDepth
	oldContinue := g.continueDepth
	oldCurrent := g.currentFunc
	oldReturnStruct := g.returnStruct
	oldStackUsed := g.stackUsed
	oldGotoLabels := g.gotoLabels
	oldLastRangeReturns := g.lastRangeReturns
	var locals []rtgLocalInfo
	var gotoLabels []rtgGlobalInfo
	g.locals = locals
	g.gotoLabels = gotoLabels
	g.breakDepth = 0
	g.continueDepth = 0
	g.currentFunc = fnInfoIndex
	g.returnStruct = 0
	g.stackUsed = 0
	rtgAsmMarkLabel(a, g.funcLabels[fnInfoIndex])
	rtgAsmEmit32(a, 8388808)
	if rtgTypeIsStruct(g.meta, metaFn.resultType) {
		g.returnStruct = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAsmStackMem(a, g.returnStruct, 35144, 0x7d, 0xbd)
	}
	if !rtgBindFunctionParams(g, fnInfoIndex) {
		return false
	}
	if !rtgEmitLinearRange(g, fn.bodyStart+1, fn.bodyEnd) {
		return false
	}
	if !g.lastRangeReturns {
		rtgAsmMovRaxImm(a, 0)
		rtgAsmLeave(a)
		rtgAsmRet(a)
	}
	g.locals = oldLocals
	g.breakDepth = oldBreak
	g.continueDepth = oldContinue
	g.currentFunc = oldCurrent
	g.returnStruct = oldReturnStruct
	g.stackUsed = oldStackUsed
	g.gotoLabels = oldGotoLabels
	g.lastRangeReturns = oldLastRangeReturns
	return true
}

func rtgBindFunctionParams(g *rtgLinearGen, fnIndex int) bool {
	meta := g.meta
	fn := &meta.funcs[fnIndex]
	reg := 0
	if rtgTypeIsStruct(meta, fn.resultType) {
		reg = 1
	}
	for i := 0; i < fn.paramCount; i++ {
		param := &meta.params[fn.firstParam+i]
		offset := rtgAddTypedLocal(g, param.nameStart, param.nameEnd, param.typ)
		if rtgTypeIsSlice(meta, param.typ) {
			if !rtgStoreParamWord(g, reg, offset) || !rtgStoreParamWord(g, reg+1, offset-8) || !rtgStoreParamWord(g, reg+2, offset-16) {
				return false
			}
			reg += 3
			continue
		}
		if rtgTypeIsString(meta, param.typ) {
			if !rtgStoreParamWord(g, reg, offset) || !rtgStoreParamWord(g, reg+1, offset-8) {
				return false
			}
			reg += 2
			continue
		}
		if rtgTypeIsStruct(meta, param.typ) {
			size := rtgTypeSize(meta, param.typ)
			for at := 0; at < size; at += 8 {
				if !rtgStoreParamWord(g, reg, offset-at) {
					return false
				}
				reg++
			}
			continue
		}
		if !rtgStoreParamWord(g, reg, offset) {
			return false
		}
		reg++
	}
	return true
}

func rtgStoreParamWord(g *rtgLinearGen, reg int, offset int) bool {
	a := &g.asm
	if reg == 0 {
		rtgAsmStackMem(a, offset, 35144, 0x7d, 0xbd)
		return true
	}
	if reg == 1 {
		rtgAsmStackMem(a, offset, 35144, 0x75, 0xb5)
		return true
	}
	if reg == 2 {
		rtgAsmStoreRdxStack(a, offset)
		return true
	}
	if reg == 3 {
		rtgAsmStackMem(a, offset, 35144, 0x4d, 0x8d)
		return true
	}
	if reg == 4 {
		rtgAsmStackMem(a, offset, 35148, 0x45, 0x85)
		return true
	}
	if reg == 5 {
		rtgAsmStackMem(a, offset, 35148, 0x4d, 0x8d)
		return true
	}
	rtgAsmEmit24(a, 8751944)
	rtgAsmEmit32(a, 16+(reg-6)*8)
	rtgAsmStoreRaxStack(a, offset)
	return true
}

func rtgEmitLinearRange(g *rtgLinearGen, start int, end int) bool {
	var bp rtgBodyParse
	var stmts []rtgStmt
	bp.prog = g.prog
	bp.stmts = stmts
	bp.ok = true
	i := start
	lastKind := 0
	for bp.ok && i < end {
		if i < 0 || i >= len(bp.prog.toks) {
			return true
		}
		if rtgTokCharIs(bp.prog, i, ';') {
			i++
			continue
		}
		if bp.prog.toks[i].start < 0 || bp.prog.toks[i].start >= len(bp.prog.src) {
			return true
		}
		if rtgTokIsKind(bp.prog, i, rtgTokEOF) {
			return true
		}
		if rtgTokCharIs(bp.prog, i, '}') {
			return true
		}
		before := len(bp.stmts)
		next := rtgParseOneStatement(&bp, i, end)
		if !bp.ok || next <= i || len(bp.stmts) <= before {
			return false
		}
		stmt := bp.stmts[len(bp.stmts)-1]
		lastKind = stmt.kind
		i = next
		if !rtgEmitLinearStmt(g, &stmt) {
			return false
		}
	}
	g.lastRangeReturns = lastKind == rtgStmtReturn
	if !bp.ok {
		return false
	}
	return true
}

func rtgEmitScopedRange(g *rtgLinearGen, start int, end int) bool {
	oldLocals := g.locals
	oldScopeBase := g.scopeBase
	g.scopeBase = len(oldLocals)
	if !rtgEmitLinearRange(g, start, end) {
		return false
	}
	g.locals = oldLocals
	g.scopeBase = oldScopeBase
	return true
}

func rtgEmitLinearStmt(g *rtgLinearGen, stmt *rtgStmt) bool {
	a := &g.asm
	p := g.prog
	if stmt.kind == rtgStmtExpr {
		if rtgEmitLinearPrintStmt(g, stmt) {
			return true
		}
		if rtgEmitLinearIncDec(g, stmt.startTok, stmt.endTok) {
			return true
		}
		ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		root := &ep.exprs[rootIndex]
		if root.kind != rtgExprCall {
			return false
		}
		if !rtgEmitIntExpr(g, &ep, rootIndex) {
			return false
		}
		return true
	}
	if stmt.kind == rtgStmtVar || stmt.kind == rtgStmtShort || stmt.kind == rtgStmtAssign {
		if !rtgEmitLinearAssign(g, stmt) {
			return false
		}
		return true
	}
	if stmt.kind == rtgStmtReturn {
		if stmt.exprStart == stmt.exprEnd {
			rtgAsmMovRaxImm(a, 0)
			rtgAsmLeave(a)
			rtgAsmRet(a)
			return true
		}
		resultType := g.meta.funcs[g.currentFunc].resultType
		if rtgTypeIsTuple(g.meta, resultType) {
			if !rtgEmitTupleReturn(g, stmt.exprStart, stmt.exprEnd) {
				return false
			}
			rtgAsmLeave(a)
			rtgAsmRet(a)
			return true
		}
		ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if rtgTypeIsStruct(g.meta, resultType) {
			if !rtgEmitStructReturnExpr(g, &ep, rootIndex) {
				return false
			}
		} else if rtgTypeIsSlice(g.meta, resultType) {
			if !rtgEmitSliceValueRegs(g, &ep, rootIndex) {
				return false
			}
		} else if rtgTypeIsString(g.meta, resultType) {
			if !rtgEmitStringValueRegs(g, &ep, rootIndex) {
				return false
			}
		} else {
			if !rtgEmitIntExpr(g, &ep, rootIndex) {
				return false
			}
		}
		rtgAsmLeave(a)
		rtgAsmRet(a)
		return true
	}
	if stmt.kind == rtgStmtIf {
		return rtgEmitLinearIf(g, stmt)
	}
	if stmt.kind == rtgStmtFor {
		return rtgEmitLinearFor(g, stmt)
	}
	if stmt.kind == rtgStmtSwitch {
		return rtgEmitLinearSwitch(g, stmt)
	}
	if stmt.kind == rtgStmtBlock {
		if !rtgEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
			return false
		}
		return true
	}
	if stmt.kind == rtgStmtGoto {
		label := rtgFindOrCreateGotoLabel(g, stmt.nameStart, stmt.nameEnd)
		rtgAsmJmpLabel(a, label)
		return true
	}
	if stmt.kind == rtgStmtLabel {
		label := rtgFindOrCreateGotoLabel(g, stmt.nameStart, stmt.nameEnd)
		rtgAsmMarkLabel(a, label)
		return true
	}
	if stmt.kind == rtgStmtBreak {
		if g.breakDepth == 0 {
			return false
		}
		rtgAsmJmpLabel(a, g.breakLabels[g.breakDepth-1])
		return true
	}
	if stmt.kind == rtgStmtContinue {
		if g.continueDepth == 0 {
			return false
		}
		rtgAsmJmpLabel(a, g.continueLabels[g.continueDepth-1])
		return true
	}
	return false
}

func rtgFindOrCreateGotoLabel(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.gotoLabels); i++ {
		info := g.gotoLabels[i]
		if rtgBytesEqualRange(g.prog.src, info.nameStart, info.nameEnd, nameStart, nameEnd) {
			return info.offset
		}
	}
	label := rtgAsmNewLabel(&g.asm)
	g.gotoLabels = append(g.gotoLabels, rtgGlobalInfo{nameStart: nameStart, nameEnd: nameEnd, offset: label})
	return label
}

func rtgEmitLinearIf(g *rtgLinearGen, stmt *rtgStmt) bool {
	a := &g.asm
	p := g.prog
	ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	endLabel := rtgAsmNewLabel(a)
	elseLabel := endLabel
	if stmt.elseStart > 0 {
		elseLabel = rtgAsmNewLabel(a)
	}
	if !rtgEmitJumpIfFalse(g, &ep, rootIndex, elseLabel) {
		return false
	}
	if !rtgEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
		return false
	}
	thenReturns := g.lastRangeReturns
	if stmt.elseStart <= 0 {
		rtgAsmMarkLabel(a, endLabel)
		return true
	}
	if !thenReturns {
		rtgAsmJmpLabel(a, endLabel)
	}
	rtgAsmMarkLabel(a, elseLabel)
	if rtgTokIsKind(p, stmt.elseStart, rtgTokIf) && rtgTokIsKind(p, stmt.elseStart-1, rtgTokElse) {
		var nested rtgBodyParse
		var stmts []rtgStmt
		nested.prog = p
		nested.stmts = stmts
		nested.ok = true
		next := rtgParseOneStatement(&nested, stmt.elseStart, stmt.elseEnd)
		if !nested.ok || next != stmt.elseEnd || len(nested.stmts) != 1 {
			return false
		}
		nestedStmt := nested.stmts[0]
		if !rtgEmitLinearStmt(g, &nestedStmt) {
			return false
		}
	} else if !rtgEmitScopedRange(g, stmt.elseStart, stmt.elseEnd) {
		return false
	}
	rtgAsmMarkLabel(a, endLabel)
	return true
}

func rtgEmitLinearFor(g *rtgLinearGen, stmt *rtgStmt) bool {
	a := &g.asm
	p := g.prog
	semi1 := rtgFindTokenTextInRange(p, stmt.exprStart, stmt.exprEnd, ';')
	if semi1 >= stmt.exprStart {
		return rtgEmitLinearClassicFor(g, stmt, semi1)
	}
	startLabel := rtgAsmNewLabel(a)
	endLabel := rtgAsmNewLabel(a)
	oldBreakDepth := g.breakDepth
	oldContinueDepth := g.continueDepth
	g.breakLabels = append(g.breakLabels, endLabel)
	g.continueLabels = append(g.continueLabels, startLabel)
	g.breakDepth = len(g.breakLabels)
	g.continueDepth = len(g.continueLabels)
	rtgAsmMarkLabel(a, startLabel)
	if stmt.exprStart < stmt.exprEnd {
		ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if !rtgEmitJumpIfFalse(g, &ep, rootIndex, endLabel) {
			return false
		}
	}
	if !rtgEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
		return false
	}
	rtgAsmJmpLabel(a, startLabel)
	rtgAsmMarkLabel(a, endLabel)
	g.breakDepth = oldBreakDepth
	g.continueDepth = oldContinueDepth
	return true
}

func rtgEmitLinearSwitch(g *rtgLinearGen, stmt *rtgStmt) bool {
	a := &g.asm
	p := g.prog
	if stmt.exprStart >= stmt.exprEnd {
		return false
	}
	ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	switchType := rtgInferParsedExprType(g, &ep, rootIndex)
	stringSwitch := rtgTypeIsString(g.meta, switchType)
	valueOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	lenOffset := 0
	if stringSwitch {
		lenOffset = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		if !rtgEmitStringValueRegs(g, &ep, rootIndex) {
			return false
		}
		rtgAsmStoreRaxStack(a, valueOffset)
		rtgAsmStoreRdxStack(a, lenOffset)
	} else {
		if !rtgEmitIntExpr(g, &ep, rootIndex) {
			return false
		}
		rtgAsmStoreRaxStack(a, valueOffset)
	}

	endLabel := rtgAsmNewLabel(a)
	oldBreakDepth := g.breakDepth
	g.breakLabels = append(g.breakLabels, endLabel)
	g.breakDepth = len(g.breakLabels)

	var clauseStarts []int
	var clauseLabels []int
	defaultLabel := endLabel
	hasDefault := false
	i := stmt.bodyStart
	for i < stmt.bodyEnd {
		clause := rtgFindNextSwitchClause(p, i, stmt.bodyEnd)
		if clause >= stmt.bodyEnd {
			break
		}
		label := rtgAsmNewLabel(a)
		clauseStarts = append(clauseStarts, clause)
		clauseLabels = append(clauseLabels, label)
		if rtgTokIsKind(p, clause, rtgTokDefault) {
			defaultLabel = label
			hasDefault = true
		}
		i = clause + 1
	}
	for i := 0; i < len(clauseStarts); i++ {
		clause := clauseStarts[i]
		if rtgTokIsKind(p, clause, rtgTokCase) {
			if !rtgEmitSwitchCaseTests(g, stmt, clause, valueOffset, lenOffset, stringSwitch, clauseLabels[i]) {
				return false
			}
		}
	}
	if hasDefault {
		rtgAsmJmpLabel(a, defaultLabel)
	} else {
		rtgAsmJmpLabel(a, endLabel)
	}
	for i := 0; i < len(clauseStarts); i++ {
		clause := clauseStarts[i]
		colon := rtgFindSwitchClauseColon(p, clause+1, stmt.bodyEnd)
		if colon <= clause {
			return false
		}
		bodyEnd := rtgFindNextSwitchClause(p, colon+1, stmt.bodyEnd)
		rtgAsmMarkLabel(a, clauseLabels[i])
		if !rtgEmitScopedRange(g, colon+1, bodyEnd) {
			return false
		}
		rtgAsmJmpLabel(a, endLabel)
	}
	rtgAsmMarkLabel(a, endLabel)
	g.breakDepth = oldBreakDepth
	return true
}

func rtgEmitSwitchCaseTests(g *rtgLinearGen, stmt *rtgStmt, clause int, valueOffset int, lenOffset int, stringSwitch bool, matchLabel int) bool {
	a := &g.asm
	p := g.prog
	colon := rtgFindSwitchClauseColon(p, clause+1, stmt.bodyEnd)
	if colon <= clause+1 {
		return false
	}
	i := clause + 1
	for i < colon {
		valueEnd := rtgFindExprBoundary(p, i, colon)
		if valueEnd <= i {
			return false
		}
		ep := rtgParseExpression(p, i, valueEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if stringSwitch {
			if !rtgEmitSwitchStringCaseTest(g, valueOffset, lenOffset, &ep, rootIndex, matchLabel) {
				return false
			}
		} else {
			rtgAsmLoadRaxStack(a, valueOffset)
			rtgAsmPushRax(a)
			if !rtgEmitIntExpr(g, &ep, rootIndex) {
				return false
			}
			rtgAsmPopRcx(a)
			rtgAsmCmpRcxRaxSet(a, 0x94)
			rtgAsmCmpRaxImm8(a, 0)
			rtgAsmJnzLabel(a, matchLabel)
		}
		i = valueEnd
		if rtgTokCharIs(p, i, ',') {
			i++
		}
	}
	return true
}

func rtgEmitSwitchStringCaseTest(g *rtgLinearGen, valueOffset int, lenOffset int, ep *rtgExprParse, idx int, matchLabel int) bool {
	a := &g.asm
	label := rtgEnsureStringEqualHelper(g)
	if !rtgEmitStringValueRegs(g, ep, idx) {
		return false
	}
	rtgAsmMovRcxRdx(a)
	rtgAsmMovRdxRax(a)
	rtgAsmLoadRaxStack(a, valueOffset)
	rtgAsmMovRdiRax(a)
	rtgAsmLoadRaxStack(a, lenOffset)
	rtgAsmMovRsiRax(a)
	rtgAsmCallLabel(a, label)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, matchLabel)
	return true
}

func rtgFindNextSwitchClause(p *rtgProgram, start int, end int) int {
	depth := 0
	i := start
	for i < end {
		if depth == 0 && (rtgTokIsKind(p, i, rtgTokCase) || rtgTokIsKind(p, i, rtgTokDefault)) {
			return i
		}
		if rtgTokCharIs(p, i, '{') {
			depth++
		} else if rtgTokCharIs(p, i, '}') {
			if depth > 0 {
				depth--
			}
		}
		i++
	}
	return end
}

func rtgFindSwitchClauseColon(p *rtgProgram, start int, end int) int {
	paren := 0
	brack := 0
	brace := 0
	i := start
	for i < end {
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
			if brace == 0 {
				return end
			}
			brace--
		}
		i++
	}
	return end
}

func rtgEmitLinearClassicFor(g *rtgLinearGen, stmt *rtgStmt, semi1 int) bool {
	a := &g.asm
	p := g.prog
	semi2 := rtgFindTokenTextInRange(p, semi1+1, stmt.exprEnd, ';')
	if semi2 <= semi1 {
		return false
	}
	if !rtgEmitLinearSimpleRange(g, stmt.exprStart, semi1) {
		return false
	}
	startLabel := rtgAsmNewLabel(a)
	postLabel := rtgAsmNewLabel(a)
	endLabel := rtgAsmNewLabel(a)
	oldBreakDepth := g.breakDepth
	oldContinueDepth := g.continueDepth
	g.breakLabels = append(g.breakLabels, endLabel)
	g.continueLabels = append(g.continueLabels, postLabel)
	g.breakDepth = len(g.breakLabels)
	g.continueDepth = len(g.continueLabels)
	rtgAsmMarkLabel(a, startLabel)
	if semi1+1 < semi2 {
		ep := rtgParseExpression(p, semi1+1, semi2)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if !rtgEmitJumpIfFalse(g, &ep, rootIndex, endLabel) {
			return false
		}
	}
	if !rtgEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
		return false
	}
	rtgAsmMarkLabel(a, postLabel)
	if !rtgEmitLinearSimpleRange(g, semi2+1, stmt.exprEnd) {
		return false
	}
	rtgAsmJmpLabel(a, startLabel)
	rtgAsmMarkLabel(a, endLabel)
	g.breakDepth = oldBreakDepth
	g.continueDepth = oldContinueDepth
	return true
}

func rtgEmitLinearSimpleRange(g *rtgLinearGen, start int, end int) bool {
	p := g.prog
	if start >= end {
		return true
	}
	if rtgEmitLinearIncDec(g, start, end) {
		return true
	}
	assignTok := rtgFindAssignmentToken(p, start, end)
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
		stmt := rtgStmt{kind: kind, startTok: start, endTok: end, exprStart: assignTok + 1, exprEnd: end, nameStart: nameStart, nameEnd: nameEnd}
		return rtgEmitLinearAssign(g, &stmt)
	}
	return false
}

func rtgEmitLinearIncDec(g *rtgLinearGen, start int, end int) bool {
	a := &g.asm
	p := g.prog
	if start+2 > end {
		return false
	}
	opTok := end - 1
	if !rtgTok2Is(p, opTok, '+', '+') && !rtgTok2Is(p, opTok, '-', '-') {
		return false
	}
	ep := rtgParseExpression(p, start, opTok)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	root := &ep.exprs[rootIndex]
	inc := rtgTok2Is(p, opTok, '+', '+')
	if root.kind == rtgExprIdent {
		localOffset := rtgFindLocalOffset(g, root.nameStart, root.nameEnd)
		if localOffset >= 0 {
			rtgAsmEmit16(a, 65352)
			if localOffset >= 0 && localOffset <= 128 {
				if inc {
					rtgAsmEmit8(a, 0x45)
				} else {
					rtgAsmEmit8(a, 0x4d)
				}
				rtgAsmEmit8(a, -localOffset)
			} else {
				if inc {
					rtgAsmEmit8(a, 0x85)
				} else {
					rtgAsmEmit8(a, 0x8d)
				}
				rtgAsmEmit32(a, -localOffset)
			}
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, root.nameStart, root.nameEnd)
		if globalOffset < 0 {
			return false
		}
		if inc {
			rtgAsmEmit24(a, 393032)
		} else {
			rtgAsmEmit24(a, 917320)
		}
		at := len(a.code)
		rtgAsmEmit32(a, 0)
		rtgAsmAddAbsReloc(a, at, globalOffset, 4)
		return true
	}
	if root.kind == rtgExprSelector {
		if !rtgEmitSelectorAddressRdx(g, &ep, rootIndex) {
			return false
		}
		if inc {
			rtgAsmIncMemRdx(a)
		} else {
			rtgAsmDecMemRdx(a)
		}
		return true
	}
	if root.kind == rtgExprIndex {
		if !rtgEmitIndexAddressRax(g, &ep, rootIndex) {
			return false
		}
		rtgAsmMovRdxRax(a)
		if inc {
			rtgAsmIncMemRdx(a)
		} else {
			rtgAsmDecMemRdx(a)
		}
		return true
	}
	if root.kind == rtgExprUnary && rtgTokCharIs(p, root.tok, '*') {
		if !rtgEmitIntExpr(g, &ep, root.left) {
			return false
		}
		rtgAsmMovRdxRax(a)
		if inc {
			rtgAsmIncMemRdx(a)
		} else {
			rtgAsmDecMemRdx(a)
		}
		return true
	}
	return false
}

func rtgEmitLinearPrintStmt(g *rtgLinearGen, stmt *rtgStmt) bool {
	p := g.prog
	a := &g.asm
	if stmt.exprStart < 0 || stmt.exprStart >= len(p.toks) || !rtgBytesEqualText(p.src, p.toks[stmt.exprStart].start, p.toks[stmt.exprStart].end, "print") {
		return false
	}
	ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	root := &ep.exprs[len(ep.exprs)-1]
	if root.kind != rtgExprCall || root.argCount != 1 || !rtgExprIsIdentText(p, &ep, root.left, "print") {
		return false
	}
	if !rtgEmitStringValueRegs(g, &ep, ep.args[root.firstArg]) {
		return false
	}
	rtgAsmEmit2(a, 0x6a, 1)
	rtgAsmPopRdi(a)
	rtgAsmMovRsiRax(a)
	rtgAsmMovRaxImm(a, 1)
	rtgAsmSyscall(a)
	return true
}

func rtgEmitRaxRcxOp(g *rtgLinearGen, tok int) bool {
	a := &g.asm
	p := g.prog
	if tok < 0 || tok >= len(p.toks) {
		return false
	}
	start := p.toks[tok].start
	end := p.toks[tok].end
	if start >= end {
		return false
	}
	c0 := p.src[start]
	c1 := byte(0)
	if start+1 < end {
		c1 = p.src[start+1]
	}
	if c0 == '+' {
		rtgAsmAddRaxRcx(a)
		return true
	}
	if c0 == '-' {
		rtgAsmEmit32(a, 1220618568)
		rtgAsmEmit16(a, 51337)
		return true
	}
	if c0 == '*' {
		rtgAsmEmit32(a, -1045491896)
		return true
	}
	if c0 == '/' {
		rtgAsmDivLeftRcxRightRax(a, false)
		return true
	}
	if c0 == '%' {
		rtgAsmDivLeftRcxRightRax(a, true)
		return true
	}
	if c0 == '&' {
		if c1 == '^' {
			rtgAsmEmit32(a, 1221654344)
			rtgAsmEmit16(a, 51233)
		} else {
			rtgAsmEmit24(a, 13115720)
		}
		return true
	}
	if c0 == '|' {
		rtgAsmEmit24(a, 13109576)
		return true
	}
	if c0 == '^' {
		rtgAsmEmit24(a, 13119816)
		return true
	}
	if c0 == '<' {
		if c1 == '<' {
			rtgAsmEmit32(a, 1221232968)
			rtgAsmEmit32(a, -1991720567)
			rtgAsmEmit32(a, -523024176)
		} else if c1 == '=' {
			rtgAsmCmpRcxRaxSet(a, 0x9e)
		} else {
			rtgAsmCmpRcxRaxSet(a, 0x9c)
		}
		return true
	}
	if c0 == '>' {
		if c1 == '>' {
			rtgAsmEmit32(a, 1221232968)
			rtgAsmEmit32(a, -1991720567)
			rtgAsmEmit32(a, -120370992)
		} else if c1 == '=' {
			rtgAsmCmpRcxRaxSet(a, 0x9d)
		} else {
			rtgAsmCmpRcxRaxSet(a, 0x9f)
		}
		return true
	}
	if c0 == '=' && c1 == '=' {
		rtgAsmCmpRcxRaxSet(a, 0x94)
		return true
	}
	if c0 == '!' && c1 == '=' {
		rtgAsmCmpRcxRaxSet(a, 0x95)
		return true
	}
	return false
}

func rtgEmitJumpIfFalse(g *rtgLinearGen, ep *rtgExprParse, idx int, falseLabel int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprBinary {
		if rtgTok2Is(p, e.tok, '&', '&') {
			if !rtgEmitJumpIfFalse(g, ep, e.left, falseLabel) {
				return false
			}
			return rtgEmitJumpIfFalse(g, ep, e.right, falseLabel)
		}
		if rtgTok2Is(p, e.tok, '|', '|') {
			trueLabel := rtgAsmNewLabel(a)
			if !rtgEmitJumpIfTrue(g, ep, e.left, trueLabel) {
				return false
			}
			if !rtgEmitJumpIfFalse(g, ep, e.right, falseLabel) {
				return false
			}
			rtgAsmMarkLabel(a, trueLabel)
			return true
		}
		if rtgEmitCompareJump(g, ep, e, falseLabel, false) {
			return true
		}
	}
	if e.kind == rtgExprUnary && rtgTokCharIs(p, e.tok, '!') {
		return rtgEmitJumpIfTrue(g, ep, e.left, falseLabel)
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJzLabel(a, falseLabel)
	return true
}

func rtgEmitJumpIfTrue(g *rtgLinearGen, ep *rtgExprParse, idx int, trueLabel int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprBinary {
		if rtgTok2Is(p, e.tok, '|', '|') {
			if !rtgEmitJumpIfTrue(g, ep, e.left, trueLabel) {
				return false
			}
			return rtgEmitJumpIfTrue(g, ep, e.right, trueLabel)
		}
		if rtgTok2Is(p, e.tok, '&', '&') {
			falseLabel := rtgAsmNewLabel(a)
			if !rtgEmitJumpIfFalse(g, ep, e.left, falseLabel) {
				return false
			}
			if !rtgEmitJumpIfTrue(g, ep, e.right, trueLabel) {
				return false
			}
			rtgAsmMarkLabel(a, falseLabel)
			return true
		}
		if rtgEmitCompareJump(g, ep, e, trueLabel, true) {
			return true
		}
	}
	if e.kind == rtgExprUnary && rtgTokCharIs(p, e.tok, '!') {
		return rtgEmitJumpIfFalse(g, ep, e.left, trueLabel)
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, trueLabel)
	return true
}

func rtgEmitCompareJump(g *rtgLinearGen, ep *rtgExprParse, e *rtgExpr, label int, jumpIfTrue bool) bool {
	p := g.prog
	if e.tok < 0 || e.tok >= len(p.toks) {
		return false
	}
	start := p.toks[e.tok].start
	end := p.toks[e.tok].end
	if start >= end {
		return false
	}
	c0 := p.src[start]
	c1 := byte(0)
	if start+1 < end {
		c1 = p.src[start+1]
	}
	isCompare := false
	if c0 == '=' && c1 == '=' {
		isCompare = true
	} else if c0 == '!' && c1 == '=' {
		isCompare = true
	} else if c0 == '<' && c1 != '<' {
		isCompare = true
	} else if c0 == '>' && c1 != '>' {
		isCompare = true
	}
	if !isCompare {
		return false
	}
	leftIndex := e.left
	rightIndex := e.right
	right := &ep.exprs[rightIndex]
	rightValue := 0
	rightOK := false
	if right.kind == rtgExprInt {
		rightValue = rtgParseIntToken(p, right.tok)
		rightOK = true
	} else if right.kind == rtgExprChar {
		rightValue = rtgParseCharToken(p, right.tok)
		rightOK = true
	} else if right.kind == rtgExprBool {
		rightValue = rtgBoolTokenValue(p, right.tok)
		rightOK = true
	} else if right.kind == rtgExprIdent {
		rightValue = rtgFindSmallConstByName(g, right.nameStart, right.nameEnd)
		rightOK = rightValue >= -128
	}
	if rightOK && rtgAsmImmFits8Signed(rightValue) {
		if !rtgEmitIntExpr(g, ep, leftIndex) {
			return false
		}
		rtgAsmCmpRaxImm8(&g.asm, rightValue)
		rtgEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue)
		return true
	}
	if c0 == '=' || c0 == '!' {
		if right.kind == rtgExprString {
			return false
		}
		if right.kind == rtgExprIdent {
			localIndex := rtgFindLocalIndex(g, right.nameStart, right.nameEnd)
			if localIndex >= 0 {
				if rtgTypeIsString(g.meta, g.locals[localIndex].typ) {
					return false
				}
			}
		}
	}
	if !rtgEmitIntExpr(g, ep, rightIndex) {
		return false
	}
	rtgAsmPushRax(&g.asm)
	if !rtgEmitIntExpr(g, ep, leftIndex) {
		return false
	}
	rtgAsmPopRcx(&g.asm)
	rtgAsmEmit24(&g.asm, 12663112)
	if c0 == '<' {
		c0 = '>'
	} else if c0 == '>' {
		c0 = '<'
	}
	rtgEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue)
	return true
}

func rtgEmitCompareJumpOp(a *rtgAsm, c0 byte, c1 byte, label int, jumpIfTrue bool) {
	op := 0
	if c0 == '=' {
		if jumpIfTrue {
			op = 33807
		} else {
			op = 34063
		}
	} else if c0 == '!' {
		if jumpIfTrue {
			op = 34063
		} else {
			op = 33807
		}
	} else if c0 == '<' {
		if c1 == '=' {
			if jumpIfTrue {
				op = 36367
			} else {
				op = 36623
			}
		} else {
			if jumpIfTrue {
				op = 35855
			} else {
				op = 36111
			}
		}
	} else if c1 == '=' {
		if jumpIfTrue {
			op = 36111
		} else {
			op = 35855
		}
	} else {
		if jumpIfTrue {
			op = 36623
		} else {
			op = 36367
		}
	}
	rtgAsmEmit16(a, op)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label, rtgRel32)
}

func rtgLinearInitGlobals(g *rtgLinearGen) bool {
	meta := g.meta
	a := &g.asm
	for i := 0; i < len(meta.globals); i++ {
		s := meta.globals[i]
		if s.kind != rtgTokVar {
			continue
		}
		off := g.asm.bssSize
		g.globals = append(g.globals, rtgGlobalInfo{nameStart: s.nameStart, nameEnd: s.nameEnd, offset: off})
		size := rtgTypeSize(meta, s.typ)
		if size < 8 {
			size = 8
		}
		g.asm.bssSize += rtgAlignTo8(size)
		if s.initStart < s.initEnd {
			ep := rtgParseExpression(g.prog, s.initStart, s.initEnd)
			if !ep.ok || len(ep.exprs) == 0 {
				return false
			}
			rootIndex := len(ep.exprs) - 1
			if rtgTypeIsString(meta, s.typ) {
				if !rtgEmitStringValueRegs(g, &ep, rootIndex) {
					return false
				}
				rtgAsmPushRdx(a)
				rtgAsmStoreRaxBss(a, off)
				rtgAsmPopRax(a)
				rtgAsmStoreRaxBss(a, off+8)
				continue
			}
			if rtgTypeIsSlice(meta, s.typ) {
				root := &ep.exprs[rootIndex]
				if root.kind != rtgExprComposite {
					return false
				}
				if !rtgEmitSliceLiteralRegs(g, &ep, rootIndex, s.typ) {
					return false
				}
				rtgAsmPushRcx(a)
				rtgAsmPushRdx(a)
				rtgAsmStoreRaxBss(a, off)
				rtgAsmPopRax(a)
				rtgAsmStoreRaxBss(a, off+8)
				rtgAsmPopRax(a)
				rtgAsmStoreRaxBss(a, off+16)
				continue
			}
			if rtgTypeIsStruct(meta, s.typ) {
				if !rtgEmitGlobalStructInit(g, &ep, rootIndex, s.typ, off) {
					return false
				}
				continue
			}
			resolved := rtgResolveType(meta, s.typ)
			root := &ep.exprs[rootIndex]
			if resolved.kind == rtgTypePointer && root.kind == rtgExprUnary && rtgTokCharIs(g.prog, root.tok, '&') {
				inner := &ep.exprs[root.left]
				if inner.kind != rtgExprIdent {
					return false
				}
				targetOff := rtgFindGlobalOffset(g, inner.nameStart, inner.nameEnd)
				if targetOff < 0 {
					return false
				}
				rtgAsmMovRaxBssAddr(a, targetOff)
				rtgAsmStoreRaxBss(a, off)
				continue
			}
			constResult := rtgEvalConstExpr(g, &ep, rootIndex)
			if !constResult.ok {
				return false
			}
			rtgAsmMovRaxImm(a, constResult.value)
			rtgAsmStoreRaxBss(a, off)
		} else if rtgTypeIsSlice(meta, s.typ) {
			rtgEmitInitEmptySliceBss(g, s.typ, off)
		}
	}
	return true
}

func rtgEmitGlobalStructInit(g *rtgLinearGen, ep *rtgExprParse, rootIndex int, typ int, off int) bool {
	root := &ep.exprs[rootIndex]
	if root.kind != rtgExprComposite {
		return false
	}
	for i := 0; i < root.argCount; i++ {
		field := ep.fields[root.firstArg+i]
		fieldIndex := rtgCompositeStructFieldIndex(g, typ, &field, i)
		if fieldIndex < 0 {
			return false
		}
		fieldOffset := g.meta.fields[fieldIndex].offset
		fieldType := g.meta.fields[fieldIndex].typ
		if fieldType == 0 {
			return false
		}
		fieldResolved := rtgResolveType(g.meta, fieldType)
		if fieldResolved.kind == rtgTypeString {
			if !rtgEmitStringValueRegs(g, ep, field.expr) {
				return false
			}
			rtgAsmPushRdx(&g.asm)
			rtgAsmStoreRaxBss(&g.asm, off+fieldOffset)
			rtgAsmPopRax(&g.asm)
			rtgAsmStoreRaxBss(&g.asm, off+fieldOffset+8)
		} else if fieldResolved.kind == rtgTypeStruct || fieldResolved.kind == rtgTypeSlice {
			return false
		} else {
			constResult := rtgEvalConstExpr(g, ep, field.expr)
			if !constResult.ok {
				return false
			}
			rtgAsmMovRaxImm(&g.asm, constResult.value)
			rtgAsmStoreRaxBss(&g.asm, off+fieldOffset)
		}
	}
	return true
}

func rtgEmitInitEmptySliceBss(g *rtgLinearGen, sliceType int, off int) {
	a := &g.asm
	t := rtgResolveType(g.meta, sliceType)
	elemSize := rtgTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	backingSize := 32768
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmStoreRaxBss(a, off)
	rtgAsmMovRaxImm(a, 0)
	rtgAsmStoreRaxBss(a, off+8)
	rtgAsmMovRaxImm(a, backingSize/elemSize)
	rtgAsmStoreRaxBss(a, off+16)
}

func rtgEmitLinearAssign(g *rtgLinearGen, stmt *rtgStmt) bool {
	meta := g.meta
	p := g.prog
	a := &g.asm
	nameStart := stmt.nameStart
	nameEnd := stmt.nameEnd
	if (stmt.kind == rtgStmtVar || rtgTokIsKind(p, stmt.startTok, rtgTokVar)) && rtgTokIsKind(p, stmt.startTok+1, rtgTokIdent) {
		nameStart = p.toks[stmt.startTok+1].start
		nameEnd = p.toks[stmt.startTok+1].end
	} else if rtgTokIsKind(p, stmt.startTok, rtgTokIdent) {
		nameStart = p.toks[stmt.startTok].start
		nameEnd = p.toks[stmt.startTok].end
	}
	assignTok := rtgFindAssignmentToken(p, stmt.startTok, stmt.endTok)
	if assignTok > stmt.startTok && rtgEmitMultiAssign(g, stmt, assignTok) {
		return true
	}
	if assignTok > stmt.startTok && rtgTokIsCompoundAssign(p, assignTok) {
		lhs := rtgParseExpression(p, stmt.startTok, assignTok)
		if lhs.ok && len(lhs.exprs) > 0 {
			lhsIndex := len(lhs.exprs) - 1
			lhsRoot := &lhs.exprs[lhsIndex]
			if lhsRoot.kind == rtgExprIndex {
				baseEnd := rtgFindTokenTextInRange(p, stmt.startTok, assignTok, '[')
				if baseEnd <= stmt.startTok {
					return false
				}
				baseEp := rtgParseExpression(p, stmt.startTok, baseEnd)
				if !baseEp.ok || len(baseEp.exprs) == 0 {
					return false
				}
				baseIndex := len(baseEp.exprs) - 1
				leftType := rtgInferParsedExprType(g, &baseEp, baseIndex)
				sliceType := rtgResolveType(meta, leftType)
				elemType := rtgResolveType(meta, sliceType.elem)
				if sliceType.kind != rtgTypeSlice || (elemType.kind != rtgTypeInt && elemType.kind != rtgTypeByte && elemType.kind != rtgTypeBool) {
					return false
				}
				indexOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				ptrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				if !rtgEmitIntExpr(g, &lhs, lhsRoot.right) {
					return false
				}
				rtgAsmStoreRaxStack(a, indexOffset)
				if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, &baseEp, baseIndex) {
					return false
				}
				rtgAsmStoreRaxStack(a, ptrOffset)
				rtgAsmLoadRaxStack(a, ptrOffset)
				rtgAsmMovRdxRax(a)
				rtgAsmLoadRcxStack(a, indexOffset)
				if elemType.kind == rtgTypeByte || elemType.kind == rtgTypeBool {
					rtgAsmLoadByteRaxIndexRcx(a)
				} else {
					rtgAsmLoadQwordRaxIndexRcx8(a)
				}
				rtgAsmPushRax(a)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rhsIndex := len(rhs.exprs) - 1
				if !rtgEmitIntExpr(g, &rhs, rhsIndex) {
					return false
				}
				rtgAsmPopRcx(a)
				if !rtgEmitRaxRcxOp(g, assignTok) {
					return false
				}
				rtgAsmLoadRdxStack(a, ptrOffset)
				rtgAsmLoadRcxStack(a, indexOffset)
				if elemType.kind == rtgTypeByte || elemType.kind == rtgTypeBool {
					rtgAsmStoreAlMemRdxRcx1(a)
				} else {
					rtgAsmStoreRaxMemRdxRcx8(a)
				}
				return true
			}
			if lhsRoot.kind == rtgExprSelector {
				if !rtgEmitSelectorAddressRdx(g, &lhs, lhsIndex) {
					return false
				}
				addrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				rtgAsmStoreRdxStack(a, addrOffset)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rtgAsmLoadRdxStack(a, addrOffset)
				rtgAsmLoadRaxMemRdxDisp(a, 0)
				rtgAsmPushRax(a)
				rhsIndex := len(rhs.exprs) - 1
				if !rtgEmitIntExpr(g, &rhs, rhsIndex) {
					return false
				}
				rtgAsmPopRcx(a)
				if !rtgEmitRaxRcxOp(g, assignTok) {
					return false
				}
				rtgAsmLoadRdxStack(a, addrOffset)
				rtgAsmStoreRaxMemRdxDisp(a, 0)
				return true
			}
		}
	}
	if assignTok > stmt.startTok && rtgTokCharIs(p, assignTok, '=') {
		lhs := rtgParseExpression(p, stmt.startTok, assignTok)
		if lhs.ok && len(lhs.exprs) > 0 {
			lhsIndex := len(lhs.exprs) - 1
			lhsRoot := &lhs.exprs[lhsIndex]
			lhsType := rtgInferParsedExprType(g, &lhs, lhsIndex)
			if lhsRoot.kind == rtgExprIndex {
				baseEnd := rtgFindTokenTextInRange(p, stmt.startTok, assignTok, '[')
				if baseEnd <= stmt.startTok {
					return false
				}
				baseEp := rtgParseExpression(p, stmt.startTok, baseEnd)
				if !baseEp.ok || len(baseEp.exprs) == 0 {
					return false
				}
				baseIndex := len(baseEp.exprs) - 1
				leftType := rtgInferParsedExprType(g, &baseEp, baseIndex)
				sliceType := rtgResolveType(meta, leftType)
				elemType := rtgResolveType(meta, sliceType.elem)
				if sliceType.kind != rtgTypeSlice {
					return false
				}
				scalarElem := elemType.kind == rtgTypeInt || elemType.kind == rtgTypeByte || elemType.kind == rtgTypeBool
				indexOffset := 0
				if scalarElem {
					indexOffset = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					if !rtgEmitIntExpr(g, &lhs, lhsRoot.right) {
						return false
					}
					rtgAsmStoreRaxStack(a, indexOffset)
				}
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rhsIndex := len(rhs.exprs) - 1
				if scalarElem {
					rhsRoot := rhs.exprs[rhsIndex]
					if rhsRoot.kind == rtgExprCall && rtgExprIdentCode(p, &rhs, rhsRoot.left) == rtgIdentByte && rhsRoot.argCount == 1 {
						argStart := rhsRoot.tok + 1
						argEnd := rtgFindExprBoundary(p, argStart, stmt.endTok)
						argEp := rtgParseExpression(p, argStart, argEnd)
						if !argEp.ok || len(argEp.exprs) == 0 {
							return false
						}
						argIndex := len(argEp.exprs) - 1
						if !rtgEmitIntExpr(g, &argEp, argIndex) {
							return false
						}
						rtgAsmEmit8(a, 0x25)
						rtgAsmEmit32(a, 255)
					} else {
						if !rtgEmitIntExpr(g, &rhs, rhsIndex) {
							return false
						}
					}
					rtgAsmPushRax(a)
					rtgAsmLoadRaxStack(a, indexOffset)
					rtgAsmPushRax(a)
					if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, &baseEp, baseIndex) {
						return false
					}
					rtgAsmMovRdxRax(a)
					rtgAsmPopRcx(a)
					rtgAsmPopRax(a)
					if elemType.kind == rtgTypeByte || elemType.kind == rtgTypeBool {
						rtgAsmStoreAlMemRdxRcx1(a)
					} else {
						rtgAsmStoreRaxMemRdxRcx8(a)
					}
					return true
				}
				if elemType.kind == rtgTypeString {
					indexOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					ptrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					if !rtgEmitIntExpr(g, &lhs, lhsRoot.right) {
						return false
					}
					rtgAsmStoreRaxStack(a, indexOffset)
					if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, &baseEp, baseIndex) {
						return false
					}
					rtgAsmStoreRaxStack(a, ptrOffset)
					if !rtgEmitStringValueRegs(g, &rhs, rhsIndex) {
						return false
					}
					rtgAsmPushStringRegs(a)
					rtgAsmLoadRaxStack(a, ptrOffset)
					rtgAsmMovRdxRax(a)
					rtgAsmLoadRcxStack(a, indexOffset)
					rtgAsmShlRcxImm(a, 4)
					rtgAsmAddRdxRcx(a)
					rtgAsmPopStoreStringMemRdx(a, 0)
					return true
				}
				if elemType.kind == rtgTypeStruct {
					indexOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					ptrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					destOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					indexEnd := rtgFindMatchingExprClose(p, baseEnd+1, assignTok, '[', ']')
					if indexEnd <= baseEnd+1 {
						return false
					}
					indexEp := rtgParseExpression(p, baseEnd+1, indexEnd)
					if !indexEp.ok || len(indexEp.exprs) == 0 {
						return false
					}
					indexRoot := len(indexEp.exprs) - 1
					if !rtgEmitIntExpr(g, &indexEp, indexRoot) {
						return false
					}
					rtgAsmStoreRaxStack(a, indexOffset)
					if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, &baseEp, baseIndex) {
						return false
					}
					rtgAsmStoreRaxStack(a, ptrOffset)
					rtgAsmLoadRaxStack(a, ptrOffset)
					rtgAsmMovRdxRax(a)
					rtgAsmLoadRcxStack(a, indexOffset)
					elemSize := rtgTypeSize(meta, sliceType.elem)
					rtgAsmImulRcxImm(a, elemSize)
					rtgAsmAddRdxRcx(a)
					rtgAsmStoreRdxStack(a, destOffset)
					if !rtgEmitCompositeFieldToMem(g, &rhs, rhsIndex, sliceType.elem, destOffset, 0) {
						return false
					}
					return true
				}
				return false
			}
			if lhsRoot.kind == rtgExprSelector && rtgTypeIsSlice(meta, lhsType) {
				if !rtgEmitSelectorAddressRdx(g, &lhs, lhsIndex) {
					return false
				}
				addrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				rtgAsmStoreRdxStack(a, addrOffset)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				if rtgEmitAppendAssignGeneral(g, stmt, &rhs) {
					return true
				}
				rhsIndex := len(rhs.exprs) - 1
				if !rtgEmitSliceValueRegs(g, &rhs, rhsIndex) {
					return false
				}
				rtgAsmPushSliceRegs(a)
				rtgAsmLoadRdxStack(a, addrOffset)
				rtgAsmPopStoreSliceMemRdx(a, 0)
				return true
			}
			if lhsRoot.kind == rtgExprSelector && rtgTypeIsStruct(meta, lhsType) {
				if !rtgEmitSelectorAddressRdx(g, &lhs, lhsIndex) {
					return false
				}
				addrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				rtgAsmStoreRdxStack(a, addrOffset)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rhsIndex := len(rhs.exprs) - 1
				size := rtgTypeSize(meta, lhsType)
				tempOffset := rtgAddTypedLocal(g, 0, 0, lhsType)
				if !rtgEmitTypedAssign(g, &rhs, rhsIndex, tempOffset) {
					return false
				}
				rtgAsmLoadRdxStack(a, addrOffset)
				rtgEmitCopyStackToMemRdx(g, tempOffset, 0, size)
				return true
			}
			if lhsRoot.kind == rtgExprSelector {
				if !rtgEmitSelectorAddressRdx(g, &lhs, lhsIndex) {
					return false
				}
				rtgAsmPushRdx(a)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rhsIndex := len(rhs.exprs) - 1
				if !rtgEmitIntExpr(g, &rhs, rhsIndex) {
					return false
				}
				rtgAsmPopRdx(a)
				rtgAsmStoreRaxMemRdxDisp(a, 0)
				return true
			}
		}
	}
	if nameEnd <= nameStart {
		if rtgTokCharIs(p, stmt.startTok, '*') && assignTok > stmt.startTok && rtgTokIsCompoundAssign(p, assignTok) {
			left := rtgParseExpression(p, stmt.startTok+1, assignTok)
			if !left.ok || len(left.exprs) == 0 {
				return false
			}
			leftIndex := len(left.exprs) - 1
			if !rtgEmitIntExpr(g, &left, leftIndex) {
				return false
			}
			rtgAsmPushRax(a)
			rtgAsmMovRdxRax(a)
			rtgAsmLoadRaxMemRdxDisp(a, 0)
			rtgAsmPushRax(a)
			right := rtgParseExpression(p, assignTok+1, stmt.endTok)
			if !right.ok || len(right.exprs) == 0 {
				return false
			}
			rightIndex := len(right.exprs) - 1
			if !rtgEmitIntExpr(g, &right, rightIndex) {
				return false
			}
			rtgAsmPopRcx(a)
			if !rtgEmitRaxRcxOp(g, assignTok) {
				return false
			}
			rtgAsmPopRdx(a)
			rtgAsmStoreRaxMemRdxDisp(a, 0)
			return true
		}
		if rtgTokCharIs(p, stmt.startTok, '*') && assignTok > stmt.startTok && rtgTokCharIs(p, assignTok, '=') {
			left := rtgParseExpression(p, stmt.startTok+1, assignTok)
			if !left.ok || len(left.exprs) == 0 {
				return false
			}
			leftIndex := len(left.exprs) - 1
			if !rtgEmitIntExpr(g, &left, leftIndex) {
				return false
			}
			rtgAsmPushRax(a)
			right := rtgParseExpression(p, assignTok+1, stmt.endTok)
			if !right.ok || len(right.exprs) == 0 {
				return false
			}
			rightIndex := len(right.exprs) - 1
			if !rtgEmitIntExpr(g, &right, rightIndex) {
				return false
			}
			rtgAsmPopRdx(a)
			rtgAsmStoreRaxMemRdxDisp(a, 0)
			return true
		}
		return false
	}
	if nameEnd == nameStart+1 && p.src[nameStart] == '_' {
		if assignTok <= stmt.startTok || !rtgTokCharIs(p, assignTok, '=') {
			return true
		}
		ep := rtgParseExpression(p, assignTok+1, stmt.endTok)
		return ep.ok
	}
	offset := rtgFindLocalOffset(g, nameStart, nameEnd)
	if stmt.kind == rtgStmtVar || rtgTokIsKind(p, stmt.startTok, rtgTokVar) || stmt.kind == rtgStmtShort {
		offset = -1
	}
	globalOffset := -1
	fieldStackOffset := -1
	if rtgTokIsKind(p, stmt.startTok, rtgTokIdent) && rtgTokCharIs(p, stmt.startTok+1, '.') && rtgTokIsKind(p, stmt.startTok+2, rtgTokIdent) {
		localIndex := rtgFindLocalIndex(g, p.toks[stmt.startTok].start, p.toks[stmt.startTok].end)
		if localIndex < 0 {
			return false
		}
		fieldOffset := rtgStructFieldOffset(g, g.locals[localIndex].typ, p.toks[stmt.startTok+2].start, p.toks[stmt.startTok+2].end)
		if fieldOffset < 0 {
			return false
		}
		fieldStackOffset = g.locals[localIndex].offset - fieldOffset
		offset = fieldStackOffset
	}
	if offset < 0 {
		if stmt.kind == rtgStmtAssign && !rtgTokIsKind(p, stmt.startTok, rtgTokVar) {
			globalOffset = rtgFindGlobalOffset(g, nameStart, nameEnd)
			if globalOffset < 0 {
				return false
			}
		} else {
			localType := rtgTypeInt
			if stmt.kind == rtgStmtVar || rtgTokIsKind(p, stmt.startTok, rtgTokVar) {
				typeEnd := assignTok
				if assignTok <= stmt.startTok {
					typeEnd = stmt.endTok
				}
				if stmt.startTok+2 < typeEnd {
					typeResult := rtgParseType(meta, g.prog, stmt.startTok+2, typeEnd)
					if typeResult.typ != 0 {
						localType = typeResult.typ
					}
				}
			}
			if stmt.kind == rtgStmtShort {
				inferredType := rtgInferExprType(g, assignTok+1, stmt.endTok)
				if assignTok+2 < stmt.endTok && rtgTokIsKind(p, assignTok+1, rtgTokIdent) && rtgTokCharIs(p, assignTok+2, '(') {
					fnIndex := -1
					for i := 0; i < len(g.meta.funcs); i++ {
						f := &g.meta.funcs[i]
						if rtgBytesEqualRange(g.prog.src, f.nameStart, f.nameEnd, p.toks[assignTok+1].start, p.toks[assignTok+1].end) {
							fnIndex = i
						}
					}
					if fnIndex >= 0 {
						inferredType = meta.funcs[fnIndex].resultType
					}
				}
				if inferredType != 0 {
					localType = inferredType
				}
			}
			offset = rtgAddTypedLocal(g, nameStart, nameEnd, localType)
		}
	}
	if assignTok <= stmt.startTok {
		if globalOffset >= 0 {
			rtgAsmMovRaxImm(a, 0)
			rtgAsmStoreRaxBss(a, globalOffset)
		} else {
			rtgZeroLocalAtOffset(g, offset)
		}
		return true
	}
	ep := rtgParseExpression(p, assignTok+1, stmt.endTok)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	if stmt.kind == rtgStmtShort {
		root := &ep.exprs[rootIndex]
		if root.kind == rtgExprCall && root.argCount == 2 && rtgExprIdentCode(p, &ep, root.left) == rtgIdentAppend {
			if !rtgEmitSliceValueRegs(g, &ep, ep.args[root.firstArg]) {
				return false
			}
			rtgAsmStoreSliceStack(a, offset)
		}
	}
	if rtgEmitAppendAssignGeneral(g, stmt, &ep) {
		return true
	}
	if rtgTokIsCompoundAssign(p, assignTok) {
		if globalOffset >= 0 {
			rtgAsmLoadRaxBss(a, globalOffset)
		} else {
			rtgAsmLoadRaxStack(a, offset)
		}
		rtgAsmPushRax(a)
		if !rtgEmitIntExpr(g, &ep, rootIndex) {
			return false
		}
		rtgAsmPopRcx(a)
		if !rtgEmitRaxRcxOp(g, assignTok) {
			return false
		}
		if globalOffset >= 0 {
			rtgAsmStoreRaxBss(a, globalOffset)
		} else {
			rtgAsmStoreRaxStack(a, offset)
		}
		return true
	}
	if globalOffset < 0 && rtgEmitTypedAssign(g, &ep, rootIndex, offset) {
		return true
	}
	if !rtgEmitIntExpr(g, &ep, rootIndex) {
		return false
	}
	if globalOffset >= 0 {
		rtgAsmStoreRaxBss(a, globalOffset)
	} else {
		rtgAsmStoreRaxStack(a, offset)
	}
	return true
}

func rtgEmitMultiAssign(g *rtgLinearGen, stmt *rtgStmt, assignTok int) bool {
	p := g.prog
	var lhs []int
	var rhs []int
	lhs = rtgSplitTopLevelComma(p, stmt.startTok, assignTok, lhs)
	rhs = rtgSplitTopLevelComma(p, assignTok+1, stmt.endTok, rhs)
	lhsCount := len(lhs) / 2
	rhsCount := len(rhs) / 2
	if lhsCount <= 1 && rhsCount <= 1 {
		return false
	}
	if lhsCount > 1 && rhsCount == 1 {
		if rtgEmitTupleCallAssign(g, stmt.kind, lhs, lhsCount, rhs[0], rhs[1]) {
			return true
		}
	}
	if lhsCount != rhsCount {
		return false
	}
	var tempOffsets []int
	var tempTypes []int
	for i := 0; i < rhsCount; i++ {
		rhsStart := rhs[i*2]
		rhsEnd := rhs[i*2+1]
		ep := rtgParseExpression(p, rhsStart, rhsEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		typ := rtgInferParsedExprType(g, &ep, rootIndex)
		if typ == 0 {
			typ = rtgTypeInt
		}
		offset := rtgAddTypedLocal(g, 0, 0, typ)
		if !rtgEmitExprToLocal(g, &ep, rootIndex, offset) {
			return false
		}
		tempOffsets = append(tempOffsets, offset)
		tempTypes = append(tempTypes, typ)
	}
	for i := 0; i < lhsCount; i++ {
		lhsStart := lhs[i*2]
		lhsEnd := lhs[i*2+1]
		if !rtgEmitTempToTarget(g, stmt.kind, lhsStart, lhsEnd, tempOffsets[i], tempTypes[i]) {
			return false
		}
	}
	return true
}

func rtgEmitTupleCallAssign(g *rtgLinearGen, kind int, lhs []int, lhsCount int, rhsStart int, rhsEnd int) bool {
	p := g.prog
	ep := rtgParseExpression(p, rhsStart, rhsEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	root := &ep.exprs[rootIndex]
	if root.kind != rtgExprCall {
		return false
	}
	fnIndex := rtgFuncInfoFromCall(g, &ep, root.left)
	if fnIndex < 0 {
		return false
	}
	resultType := g.meta.funcs[fnIndex].resultType
	if !rtgTypeIsTuple(g.meta, resultType) {
		return false
	}
	tuple := rtgResolveType(g.meta, resultType)
	if tuple.count != lhsCount {
		return false
	}
	offset := rtgAddTypedLocal(g, 0, 0, resultType)
	if !rtgEmitStructCallToLocal(g, &ep, rootIndex, resultType, offset) {
		return false
	}
	for i := 0; i < lhsCount; i++ {
		field := g.meta.fields[tuple.first+i]
		lhsStart := lhs[i*2]
		lhsEnd := lhs[i*2+1]
		if !rtgEmitTempToTarget(g, kind, lhsStart, lhsEnd, offset-field.offset, field.typ) {
			return false
		}
	}
	return true
}

func rtgEmitExprToLocal(g *rtgLinearGen, ep *rtgExprParse, idx int, offset int) bool {
	if rtgEmitTypedAssign(g, ep, idx, offset) {
		return true
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	rtgAsmStoreRaxStack(&g.asm, offset)
	return true
}

func rtgEmitTempToTarget(g *rtgLinearGen, kind int, targetStart int, targetEnd int, tempOffset int, tempType int) bool {
	p := g.prog
	ep := rtgParseExpression(p, targetStart, targetEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	root := &ep.exprs[rootIndex]
	size := rtgTypeSize(g.meta, tempType)
	if size < 8 {
		size = 8
	}
	if root.kind == rtgExprIdent {
		if root.nameEnd == root.nameStart+1 && p.src[root.nameStart] == '_' {
			return true
		}
		localIndex := rtgFindLocalIndex(g, root.nameStart, root.nameEnd)
		if kind == rtgStmtShort {
			localIndex = rtgFindLocalIndexInCurrentScope(g, root.nameStart, root.nameEnd)
			if localIndex < 0 {
				offset := rtgAddTypedLocal(g, root.nameStart, root.nameEnd, tempType)
				rtgEmitCopyStackToStack(g, tempOffset, offset, size)
				return true
			}
		}
		if localIndex >= 0 {
			rtgEmitCopyStackToStack(g, tempOffset, g.locals[localIndex].offset, size)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, root.nameStart, root.nameEnd)
		if globalOffset < 0 {
			return false
		}
		rtgEmitCopyStackToBss(g, tempOffset, globalOffset, size)
		return true
	}
	if kind == rtgStmtShort {
		return false
	}
	if root.kind == rtgExprSelector {
		if !rtgEmitSelectorAddressRdx(g, &ep, rootIndex) {
			return false
		}
		targetType := rtgInferParsedExprType(g, &ep, rootIndex)
		targetSize := rtgTypeSize(g.meta, targetType)
		if targetSize < 8 {
			targetSize = 8
		}
		rtgEmitCopyStackToMemRdx(g, tempOffset, 0, targetSize)
		return true
	}
	if root.kind == rtgExprIndex {
		if !rtgEmitIndexAddressRax(g, &ep, rootIndex) {
			return false
		}
		rtgAsmMovRdxRax(&g.asm)
		targetType := rtgInferParsedExprType(g, &ep, rootIndex)
		targetSize := rtgTypeSize(g.meta, targetType)
		if targetSize < 8 {
			targetSize = 8
		}
		rtgEmitCopyStackToMemRdx(g, tempOffset, 0, targetSize)
		return true
	}
	if root.kind == rtgExprUnary && rtgTokCharIs(p, root.tok, '*') {
		if !rtgEmitIntExpr(g, &ep, root.left) {
			return false
		}
		rtgAsmMovRdxRax(&g.asm)
		targetType := rtgInferParsedExprType(g, &ep, rootIndex)
		targetSize := rtgTypeSize(g.meta, targetType)
		if targetSize < 8 {
			targetSize = 8
		}
		rtgEmitCopyStackToMemRdx(g, tempOffset, 0, targetSize)
		return true
	}
	return false
}

func rtgEmitCopyStackToBss(g *rtgLinearGen, srcOffset int, bssOffset int, size int) {
	if size < 8 {
		size = 8
	}
	for at := 0; at < size; at += 8 {
		rtgAsmLoadRaxStack(&g.asm, srcOffset-at)
		rtgAsmStoreRaxBss(&g.asm, bssOffset+at)
	}
}

func rtgFindLocalIndexInCurrentScope(g *rtgLinearGen, nameStart int, nameEnd int) int {
	start := g.scopeBase
	if start < 0 {
		start = 0
	}
	for i := len(g.locals) - 1; i >= start; i-- {
		if rtgBytesEqualRange(g.prog.src, g.locals[i].nameStart, g.locals[i].nameEnd, nameStart, nameEnd) {
			return i
		}
	}
	return -1
}

func rtgSplitTopLevelComma(p *rtgProgram, start int, end int, ranges []int) []int {
	partStart := start
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
		} else if paren == 0 && brack == 0 && brace == 0 && rtgTokCharIs(p, i, ',') {
			ranges = append(ranges, partStart)
			ranges = append(ranges, i)
			partStart = i + 1
		}
		i++
	}
	if partStart < end {
		ranges = append(ranges, partStart)
		ranges = append(ranges, end)
	}
	return ranges
}

func rtgEmitTupleReturn(g *rtgLinearGen, start int, end int) bool {
	resultType := g.meta.funcs[g.currentFunc].resultType
	tuple := rtgResolveType(g.meta, resultType)
	var parts []int
	parts = rtgSplitTopLevelComma(g.prog, start, end, parts)
	count := len(parts) / 2
	if count == tuple.count {
		for i := 0; i < count; i++ {
			partStart := parts[i*2]
			partEnd := parts[i*2+1]
			field := g.meta.fields[tuple.first+i]
			if !rtgEmitTupleReturnField(g, partStart, partEnd, field.typ, field.offset) {
				return false
			}
		}
		return true
	}
	if count == 1 {
		ep := rtgParseExpression(g.prog, start, end)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		return rtgEmitStructReturnExpr(g, &ep, rootIndex)
	}
	return false
}

func rtgEmitTupleReturnField(g *rtgLinearGen, start int, end int, typ int, fieldOffset int) bool {
	ep := rtgParseExpression(g.prog, start, end)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	tempOffset := rtgAddTypedLocal(g, 0, 0, typ)
	if !rtgEmitExprToLocal(g, &ep, rootIndex, tempOffset) {
		return false
	}
	size := rtgTypeSize(g.meta, typ)
	if size < 8 {
		size = 8
	}
	rtgAsmLoadRdxStack(&g.asm, g.returnStruct)
	rtgEmitCopyStackToMemRdx(g, tempOffset, fieldOffset, size)
	return true
}

func rtgInferExprType(g *rtgLinearGen, start int, end int) int {
	ep := rtgParseExpression(g.prog, start, end)
	if !ep.ok || len(ep.exprs) == 0 {
		return 0
	}
	rootIndex := len(ep.exprs) - 1
	return rtgInferParsedExprType(g, &ep, rootIndex)
}

func rtgInferParsedExprType(g *rtgLinearGen, ep *rtgExprParse, idx int) int {
	p := g.prog
	meta := g.meta
	e := ep.exprs[idx]
	if e.kind == rtgExprInt || e.kind == rtgExprChar || e.kind == rtgExprBool {
		return rtgTypeInt
	}
	if e.kind == rtgExprFloat {
		return rtgTypeFloat64
	}
	if e.kind == rtgExprString {
		return rtgTypeString
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			return g.locals[localIndex].typ
		}
		for i := 0; i < len(meta.globals); i++ {
			s := &meta.globals[i]
			if rtgBytesEqualRange(p.src, s.nameStart, s.nameEnd, e.nameStart, e.nameEnd) {
				return s.typ
			}
		}
		constStringTok := rtgFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constStringTok >= 0 {
			return rtgTypeString
		}
		return rtgTypeInt
	}
	if e.kind == rtgExprCall {
		callee := rtgExprIdentCode(p, ep, e.left)
		if callee == rtgIdentAppend && e.argCount == 2 {
			return rtgInferParsedExprType(g, ep, ep.args[e.firstArg])
		}
		if callee == rtgIdentByteSlice && e.argCount == 1 {
			return rtgAddType(meta, rtgTypeSlice, rtgTypeByte, 0, 0, 24, 0, 0)
		}
		if e.argCount == 2 || e.argCount == 3 {
			if callee == rtgIdentMake {
				return rtgTypeFromExpr(g, ep, ep.args[e.firstArg])
			}
		}
		if callee == rtgIdentRtgEnv && e.argCount == 0 {
			fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
			if fnIndex >= 0 {
				return meta.funcs[fnIndex].resultType
			}
		}
		if callee == rtgIdentRtgParseProgram {
			named := rtgFindTypeByText(g, "rtgProgram")
			if named > 0 {
				return named
			}
		}
		if callee == rtgIdentInt || callee == rtgIdentInt64 || callee == rtgIdentByte || callee == rtgIdentLen || callee == rtgIdentOpen || callee == rtgIdentClose || callee == rtgIdentRead || callee == rtgIdentWrite || callee == rtgIdentChmod || callee == rtgIdentCopy {
			return rtgTypeInt
		}
		fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
		if fnIndex >= 0 {
			if rtgBytesEqualText(p.src, meta.funcs[fnIndex].nameStart, meta.funcs[fnIndex].nameEnd, "rtgParseProgram") {
				named := rtgFindTypeByText(g, "rtgProgram")
				if named > 0 {
					return named
				}
			}
			return meta.funcs[fnIndex].resultType
		}
		if e.argCount == 1 {
			calleeExpr := &ep.exprs[e.left]
			if calleeExpr.kind == rtgExprIdent {
				namedType := rtgFindTypeByRange(g, calleeExpr.nameStart, calleeExpr.nameEnd)
				if namedType > 0 {
					return namedType
				}
			}
		}
	}
	if e.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, e.left)
		t := rtgResolveType(meta, leftType)
		if t.kind == rtgTypeSlice {
			return t.elem
		}
		if t.kind == rtgTypeString {
			return rtgTypeByte
		}
	}
	if e.kind == rtgExprSlice {
		return rtgInferParsedExprType(g, ep, e.left)
	}
	if e.kind == rtgExprSelector {
		baseType := rtgInferParsedExprType(g, ep, e.left)
		t := rtgResolveType(meta, baseType)
		if t.kind == rtgTypePointer {
			t = rtgResolveType(meta, t.elem)
		}
		if t.kind == rtgTypeStruct {
			for i := 0; i < t.count; i++ {
				field := &meta.fields[t.first+i]
				if rtgBytesEqualRange(p.src, field.nameStart, field.nameEnd, e.nameStart, e.nameEnd) {
					return field.typ
				}
			}
		}
	}
	if e.kind == rtgExprComposite {
		return rtgTypeFromExpr(g, ep, idx)
	}
	if e.kind == rtgExprUnary {
		if rtgTokCharIs(p, e.tok, '+') || rtgTokCharIs(p, e.tok, '-') {
			return rtgInferParsedExprType(g, ep, e.left)
		}
		if rtgTokCharIs(p, e.tok, '&') {
			elemType := rtgInferParsedExprType(g, ep, e.left)
			if elemType == 0 {
				return 0
			}
			return rtgAddType(meta, rtgTypePointer, elemType, 0, 0, 8, 0, 0)
		}
		if rtgTokCharIs(p, e.tok, '*') {
			innerType := rtgInferParsedExprType(g, ep, e.left)
			inner := rtgResolveType(meta, innerType)
			if inner.kind == rtgTypePointer {
				return inner.elem
			}
		}
	}
	if e.kind == rtgExprBinary {
		if rtgTok2Is(p, e.tok, '=', '=') || rtgTok2Is(p, e.tok, '!', '=') || rtgTokCharIs(p, e.tok, '<') || rtgTokCharIs(p, e.tok, '>') {
			return rtgTypeInt
		}
		leftType := rtgResolveType(meta, rtgInferParsedExprType(g, ep, e.left))
		rightType := rtgResolveType(meta, rtgInferParsedExprType(g, ep, e.right))
		if leftType.kind == rtgTypeFloat64 || rightType.kind == rtgTypeFloat64 {
			return rtgTypeFloat64
		}
	}
	return rtgTypeInt
}

func rtgTypeFromExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) int {
	p := g.prog
	e := &ep.exprs[idx]
	if e.tok < 0 || e.tok >= len(p.toks) {
		return 0
	}
	endTok := e.tok
	for endTok < len(p.toks) && p.toks[endTok].end <= e.nameEnd {
		endTok++
	}
	typeResult := rtgParseType(g.meta, p, e.tok, endTok)
	return typeResult.typ
}

func rtgFindTypeByText(g *rtgLinearGen, name string) int {
	for i := 0; i < len(g.meta.types); i++ {
		t := &g.meta.types[i]
		if t.nameEnd > t.nameStart && rtgBytesEqualText(g.prog.src, t.nameStart, t.nameEnd, name) {
			return i
		}
	}
	return 0
}

func rtgFindTypeByRange(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.meta.types); i++ {
		t := &g.meta.types[i]
		if t.nameEnd > t.nameStart && rtgBytesEqualRange(g.prog.src, t.nameStart, t.nameEnd, nameStart, nameEnd) {
			return i
		}
	}
	return 0
}

func rtgLocalTypeAtOffset(g *rtgLinearGen, offset int) int {
	for i := 0; i < len(g.locals); i++ {
		if g.locals[i].offset == offset {
			return g.locals[i].typ
		}
	}
	for i := 0; i < len(g.locals); i++ {
		t := rtgResolveType(g.meta, g.locals[i].typ)
		if t.kind == rtgTypeStruct {
			for j := 0; j < t.count; j++ {
				field := g.meta.fields[t.first+j]
				if g.locals[i].offset-field.offset == offset {
					return field.typ
				}
			}
		}
	}
	return 0
}

func rtgEmitTypedAssign(g *rtgLinearGen, ep *rtgExprParse, idx int, offset int) bool {
	meta := g.meta
	destType := rtgLocalTypeAtOffset(g, offset)
	e := &ep.exprs[idx]
	destResolved := rtgResolveType(meta, destType)
	if destResolved.kind == rtgTypeStruct {
		if e.kind == rtgExprIdent {
			localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
			if localIndex < 0 || rtgTypeSize(meta, g.locals[localIndex].typ) != rtgTypeSize(meta, destType) {
				return false
			}
			size := rtgTypeSize(meta, destType)
			rtgEmitCopyStackToStack(g, g.locals[localIndex].offset, offset, size)
			return true
		}
		if e.kind == rtgExprCall {
			return rtgEmitStructCallToLocal(g, ep, idx, destType, offset)
		}
		if e.kind == rtgExprIndex {
			return rtgEmitIndexedStructToLocal(g, ep, idx, destType, offset)
		}
		if e.kind == rtgExprSelector {
			fieldType := rtgInferParsedExprType(g, ep, idx)
			if !rtgTypeIsStruct(meta, fieldType) || rtgTypeSize(meta, fieldType) != rtgTypeSize(meta, destType) {
				return false
			}
			if !rtgEmitSelectorAddressRdx(g, ep, idx) {
				return false
			}
			size := rtgTypeSize(meta, destType)
			rtgEmitCopyMemRdxToStack(g, offset, size)
			return true
		}
		if e.kind == rtgExprComposite {
			rtgZeroLocalAtOffset(g, offset)
			for i := 0; i < e.argCount; i++ {
				field := ep.fields[e.firstArg+i]
				fieldIndex := rtgCompositeStructFieldIndex(g, destType, &field, i)
				if fieldIndex < 0 {
					return false
				}
				fieldOffset := g.meta.fields[fieldIndex].offset
				fieldType := g.meta.fields[fieldIndex].typ
				if fieldType == 0 {
					return false
				}
				if !rtgEmitCompositeFieldToStack(g, ep, field.expr, fieldType, offset-fieldOffset) {
					return false
				}
			}
			return true
		}
		return false
	}
	if destResolved.kind == rtgTypeString {
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmStoreRaxStack(&g.asm, offset)
		rtgAsmStoreRdxStack(&g.asm, offset-8)
		return true
	}
	if !rtgTypeIsSlice(meta, destType) {
		return false
	}
	if !rtgEmitSliceValueRegs(g, ep, idx) {
		return false
	}
	rtgAsmStoreSliceStack(&g.asm, offset)
	return true
}

func rtgEmitSliceValueRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprSlice {
		if e.firstArg >= 0 {
			return false
		}
		if !rtgEmitSliceValueRegs(g, ep, e.left) {
			return false
		}
		if e.right >= 0 {
			rtgAsmPushRax(a)
			rtgAsmPushRcx(a)
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmMovRdxRax(a)
			rtgAsmPopRcx(a)
			rtgAsmPopRax(a)
		}
		return true
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
			globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalOffset < 0 || !rtgTypeIsSlice(meta, globalType) {
				return false
			}
			rtgAsmLoadRaxBss(a, globalOffset+16)
			rtgAsmPushRax(a)
			rtgAsmLoadRaxBss(a, globalOffset+8)
			rtgAsmPushRax(a)
			rtgAsmLoadRaxBss(a, globalOffset)
			rtgAsmPopRdx(a)
			rtgAsmPopRcx(a)
			return true
		}
		if !rtgTypeIsSlice(meta, g.locals[localIndex].typ) {
			return false
		}
		rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
		rtgAsmLoadRdxStack(a, g.locals[localIndex].offset-8)
		rtgAsmLoadRcxStack(a, g.locals[localIndex].offset-16)
		return true
	}
	if e.kind == rtgExprSelector {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(meta, valueType) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return false
		}
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		rtgAsmPushRax(a)
		rtgAsmLoadRaxMemRdxDisp(a, 8)
		rtgAsmPushRax(a)
		rtgAsmLoadRaxMemRdxDisp(a, 16)
		rtgAsmMovRcxRax(a)
		rtgAsmPopRdx(a)
		rtgAsmPopRax(a)
		return true
	}
	if e.kind == rtgExprComposite {
		sliceType := rtgTypeFromExpr(g, ep, idx)
		if !rtgTypeIsSlice(meta, sliceType) {
			return false
		}
		return rtgEmitSliceLiteralRegs(g, ep, idx, sliceType)
	}
	if e.kind == rtgExprCall {
		callee := rtgExprIdentCode(g.prog, ep, e.left)
		if e.argCount == 2 && callee == rtgIdentAppend {
			var stmt rtgStmt
			if !rtgEmitAppendAssignGeneral(g, &stmt, ep) {
				return false
			}
			return rtgEmitSliceValueRegs(g, ep, ep.args[e.firstArg])
		}
		if e.argCount == 2 || e.argCount == 3 {
			if callee == rtgIdentMake {
				return rtgEmitMakeSliceRegs(g, ep, idx)
			}
		}
		if e.argCount == 1 && callee == rtgIdentByteSlice {
			return rtgEmitByteSliceConversionRegs(g, ep, idx)
		}
		if e.argCount == 0 && callee == rtgIdentRtgEnv {
			if g.envDataOff == 0 && g.envLenOff == 0 {
				return false
			}
			rtgAsmLoadRaxBss(a, g.envLenOff)
			rtgAsmMovRdxRax(a)
			rtgAsmPushRdx(a)
			rtgAsmPopRcx(a)
			rtgAsmMovRaxBssAddr(a, g.envDataOff)
			return true
		}
		callType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(meta, callType) {
			return false
		}
		if !rtgEmitIntExpr(g, ep, idx) {
			return false
		}
		return true
	}
	return false
}

func rtgEmitSliceLiteralRegs(g *rtgLinearGen, ep *rtgExprParse, idx int, sliceType int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	t := rtgResolveType(g.meta, sliceType)
	if t.kind != rtgTypeSlice {
		return false
	}
	elemSize := rtgTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	backingSize := 32768
	needSize := e.argCount * elemSize
	if needSize > backingSize {
		backingSize = rtgAlignTo8(needSize)
	}
	if backingSize < elemSize {
		backingSize = elemSize
	}
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	if !rtgEmitSliceLiteralBacking(g, ep, idx, sliceType, backingOff) {
		return false
	}
	capacity := backingSize / elemSize
	rtgAsmMovRaxImm(a, capacity)
	rtgAsmPushRax(a)
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmMovRdxImm(a, e.argCount)
	rtgAsmPopRcx(a)
	return true
}

func rtgEmitSliceLiteralBacking(g *rtgLinearGen, ep *rtgExprParse, idx int, sliceType int, backingOff int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	t := rtgResolveType(g.meta, sliceType)
	if t.kind != rtgTypeSlice {
		return false
	}
	elemType := t.elem
	elemResolved := rtgResolveType(g.meta, elemType)
	elemSize := rtgTypeSize(g.meta, elemType)
	if elemSize < 1 {
		elemSize = 8
	}
	for i := 0; i < e.argCount; i++ {
		field := ep.fields[e.firstArg+i]
		if field.nameEnd > field.nameStart {
			return false
		}
		disp := i * elemSize
		if elemResolved.kind == rtgTypeString {
			if !rtgEmitStringValueRegs(g, ep, field.expr) {
				return false
			}
			rtgAsmPushStringRegs(a)
			rtgAsmMovRaxBssAddr(a, backingOff)
			rtgAsmMovRdxRax(a)
			rtgAsmPopStoreStringMemRdx(a, disp)
			continue
		}
		if elemResolved.kind == rtgTypeStruct {
			addrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
			rtgAsmMovRaxBssAddr(a, backingOff)
			rtgAsmMovRdxRax(a)
			if disp != 0 {
				rtgAsmAddRdxImm(a, disp)
			}
			rtgAsmStoreRdxStack(a, addrOffset)
			if !rtgEmitCompositeFieldToMem(g, ep, field.expr, elemType, addrOffset, 0) {
				return false
			}
			continue
		}
		if elemResolved.kind != rtgTypeInt && elemResolved.kind != rtgTypeInt64 && elemResolved.kind != rtgTypeByte && elemResolved.kind != rtgTypeBool {
			return false
		}
		if !rtgEmitIntExpr(g, ep, field.expr) {
			return false
		}
		rtgAsmPushRax(a)
		rtgAsmMovRaxBssAddr(a, backingOff)
		rtgAsmMovRdxRax(a)
		rtgAsmPopRax(a)
		if elemSize == 1 {
			if disp == 0 {
				rtgAsmEmit2(a, 0x88, 0x02)
			} else if rtgAsmImmFits8Signed(disp) {
				rtgAsmEmit3(a, 0x88, 0x42, disp)
			} else {
				rtgAsmEmit16(a, 33416)
				rtgAsmEmit32(a, disp)
			}
		} else {
			rtgAsmStoreRaxMemRdxDisp(a, disp)
		}
	}
	return true
}

func rtgEmitMakeSliceRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	if e.argCount != 2 && e.argCount != 3 {
		return false
	}
	sliceType := rtgTypeFromExpr(g, ep, ep.args[e.firstArg])
	t := rtgResolveType(g.meta, sliceType)
	if t.kind != rtgTypeSlice {
		return false
	}
	elemSize := rtgTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	lenOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	capOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
		return false
	}
	rtgAsmStoreRaxStack(a, lenOffset)
	if e.argCount == 3 {
		if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+2]) {
			return false
		}
		rtgAsmStoreRaxStack(a, capOffset)
	} else {
		rtgAsmLoadRaxStack(a, lenOffset)
		rtgAsmStoreRaxStack(a, capOffset)
	}
	backingSize := 32768
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmLoadRdxStack(a, lenOffset)
	rtgAsmLoadRcxStack(a, capOffset)
	return true
}

func rtgEmitByteSliceConversionRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	srcOff := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	lenOff := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	idxOff := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	backingOff := g.asm.bssSize
	backingSize := 32768
	g.asm.bssSize += backingSize
	argIndex := ep.args[e.firstArg]
	if !rtgEmitStringValueRegs(g, ep, argIndex) {
		return false
	}
	rtgAsmStoreRaxStack(a, srcOff)
	rtgAsmStoreRdxStack(a, lenOff)
	rtgAsmMovRaxImm(a, 0)
	rtgAsmStoreRaxStack(a, idxOff)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmLoadRaxStack(a, idxOff)
	rtgAsmPushRax(a)
	rtgAsmLoadRaxStack(a, lenOff)
	rtgAsmPopRcx(a)
	rtgAsmCmpRcxRaxSet(a, 0x9d)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, doneLabel)
	rtgAsmLoadRaxStack(a, idxOff)
	rtgAsmPushRax(a)
	rtgAsmLoadRaxStack(a, srcOff)
	rtgAsmPopRcx(a)
	rtgAsmLoadByteRaxIndexRcx(a)
	rtgAsmPushRax(a)
	rtgAsmLoadRaxStack(a, idxOff)
	rtgAsmPushRax(a)
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmMovRdxRax(a)
	rtgAsmPopRcx(a)
	rtgAsmPopRax(a)
	rtgAsmStoreAlMemRdxRcx1(a)
	rtgAsmLoadRaxStack(a, idxOff)
	rtgAsmIncRax(a)
	rtgAsmStoreRaxStack(a, idxOff)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmLoadRdxStack(a, lenOff)
	rtgAsmMovRcxRdx(a)
	return true
}

func rtgEmitStringValueRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprString {
		msg := rtgDecodeStringToken(g.prog, e.tok)
		msgOff := rtgAddStringData(g, msg)
		msgLen := len(msg)
		rtgAsmMovRaxDataAddr(a, msgOff)
		rtgAsmMovRdxImm(a, msgLen)
		return true
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if !rtgTypeIsString(meta, g.locals[localIndex].typ) {
				return false
			}
			rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
			rtgAsmLoadRdxStack(a, g.locals[localIndex].offset-8)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
		globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 && rtgTypeIsString(meta, globalType) {
			rtgAsmLoadRaxBss(a, globalOffset)
			rtgAsmPushRax(a)
			rtgAsmLoadRaxBss(a, globalOffset+8)
			rtgAsmMovRdxRax(a)
			rtgAsmPopRax(a)
			return true
		}
		constTok := rtgFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constTok >= 0 {
			msg := rtgDecodeStringToken(g.prog, constTok)
			msgOff := rtgAddStringData(g, msg)
			msgLen := len(msg)
			rtgAsmMovRaxDataAddr(a, msgOff)
			rtgAsmMovRdxImm(a, msgLen)
			return true
		}
		return false
	}
	if e.kind == rtgExprIndex {
		left := &ep.exprs[e.left]
		if left.kind != rtgExprIdent {
			return false
		}
		localIndex := rtgFindLocalIndex(g, left.nameStart, left.nameEnd)
		if localIndex < 0 {
			return false
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind != rtgTypeSlice {
			return false
		}
		elem := rtgResolveType(meta, t.elem)
		if elem.kind != rtgTypeString {
			return false
		}
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushRax(a)
		rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
		rtgAsmPopRcx(a)
		rtgAsmShlRcxImm(a, 4)
		rtgAsmMovRdxRax(a)
		rtgAsmEmit32(a, 134515528)
		rtgAsmAddRdxRcx(a)
		rtgAsmMemDisp(a, 8, 35656, 0x52, 0x92)
		return true
	}
	if e.kind == rtgExprSelector {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsString(meta, valueType) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return false
		}
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		rtgAsmPushRax(a)
		rtgAsmLoadRaxMemRdxDisp(a, 8)
		rtgAsmMovRdxRax(a)
		rtgAsmPopRax(a)
		return true
	}
	if e.kind == rtgExprCall {
		callType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsString(meta, callType) {
			return false
		}
		if !rtgEmitUserCall(g, ep, idx) {
			return false
		}
		return true
	}
	return false
}

func rtgEmitCompositeFieldToStack(g *rtgLinearGen, ep *rtgExprParse, idx int, fieldType int, destOffset int) bool {
	a := &g.asm
	fieldResolved := rtgResolveType(g.meta, fieldType)
	if fieldResolved.kind == rtgTypeSlice {
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmStoreSliceStack(a, destOffset)
		return true
	}
	if fieldResolved.kind == rtgTypeString {
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmStoreRaxStack(a, destOffset)
		rtgAsmStoreRdxStack(a, destOffset-8)
		return true
	}
	if fieldResolved.kind == rtgTypeStruct {
		tempOffset := rtgAddTypedLocal(g, 0, 0, fieldType)
		if !rtgEmitTypedAssign(g, ep, idx, tempOffset) {
			return false
		}
		size := rtgTypeSize(g.meta, fieldType)
		rtgEmitCopyStackToStack(g, tempOffset, destOffset, size)
		return true
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	rtgAsmStoreRaxStack(a, destOffset)
	return true
}

func rtgEmitCompositeFieldToMem(g *rtgLinearGen, ep *rtgExprParse, idx int, fieldType int, addrOffset int, fieldOffset int) bool {
	a := &g.asm
	fieldResolved := rtgResolveType(g.meta, fieldType)
	if fieldResolved.kind == rtgTypeSlice {
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmPushSliceRegs(a)
		rtgAsmLoadRdxStack(a, addrOffset)
		rtgAsmPopStoreSliceMemRdx(a, fieldOffset)
		return true
	}
	if fieldResolved.kind == rtgTypeString {
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmPushStringRegs(a)
		rtgAsmLoadRdxStack(a, addrOffset)
		rtgAsmPopStoreStringMemRdx(a, fieldOffset)
		return true
	}
	if fieldResolved.kind == rtgTypeStruct {
		tempOffset := rtgAddTypedLocal(g, 0, 0, fieldType)
		if !rtgEmitTypedAssign(g, ep, idx, tempOffset) {
			return false
		}
		size := rtgTypeSize(g.meta, fieldType)
		rtgAsmLoadRdxStack(a, addrOffset)
		rtgEmitCopyStackToMemRdx(g, tempOffset, fieldOffset, size)
		return true
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	rtgAsmLoadRdxStack(a, addrOffset)
	rtgAsmStoreRaxMemRdxDisp(a, fieldOffset)
	return true
}

func rtgEmitCopyStackToStack(g *rtgLinearGen, srcOffset int, destOffset int, size int) {
	a := &g.asm
	if size > 16 {
		label := rtgEnsureCopyWordsHelper(g)
		rtgAsmLeaRdiStack(a, destOffset)
		rtgAsmLeaRsiStack(a, srcOffset)
		rtgAsmMovRdxImm(a, size/8)
		rtgAsmCallLabel(a, label)
		return
	}
	for at := 0; at < size; at += 8 {
		rtgAsmLoadRaxStack(a, srcOffset-at)
		rtgAsmStoreRaxStack(a, destOffset-at)
	}
}

func rtgEmitCopyStackToMemRdx(g *rtgLinearGen, srcOffset int, destDisp int, size int) {
	a := &g.asm
	if size > 16 {
		label := rtgEnsureCopyWordsHelper(g)
		if destDisp != 0 {
			rtgAsmAddRdxImm(a, destDisp)
		}
		rtgAsmEmit16(a, 24402)
		rtgAsmLeaRsiStack(a, srcOffset)
		rtgAsmMovRdxImm(a, size/8)
		rtgAsmCallLabel(a, label)
		return
	}
	for at := 0; at < size; at += 8 {
		rtgAsmLoadRaxStack(a, srcOffset-at)
		rtgAsmStoreRaxMemRdxDisp(a, destDisp+at)
	}
}

func rtgEmitCopyMemRdxToStack(g *rtgLinearGen, destOffset int, size int) {
	a := &g.asm
	if size > 16 {
		label := rtgEnsureCopyWordsHelper(g)
		rtgAsmLeaRdiStack(a, destOffset)
		rtgAsmEmit16(a, 24146)
		rtgAsmMovRdxImm(a, size/8)
		rtgAsmCallLabel(a, label)
		return
	}
	for at := 0; at < size; at += 8 {
		rtgAsmLoadRaxMemRdxDisp(a, at)
		rtgAsmStoreRaxStack(a, destOffset-at)
	}
}

func rtgEmitPushStackWords(g *rtgLinearGen, offset int, size int) {
	for at := size - 8; at >= 0; at -= 8 {
		rtgAsmLoadRaxStack(&g.asm, offset-at)
		rtgAsmPushRax(&g.asm)
	}
}

func rtgEmitPushMemRdxWords(g *rtgLinearGen, size int) {
	for at := size - 8; at >= 0; at -= 8 {
		rtgAsmLoadRaxMemRdxDisp(&g.asm, at)
		rtgAsmPushRax(&g.asm)
	}
}

func rtgEmitIndexedStructToLocal(g *rtgLinearGen, ep *rtgExprParse, idx int, destType int, offset int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	leftType := rtgInferParsedExprType(g, ep, e.left)
	sliceType := rtgResolveType(meta, leftType)
	if sliceType.kind != rtgTypeSlice {
		return false
	}
	elemType := rtgResolveType(meta, sliceType.elem)
	destResolved := rtgResolveType(meta, destType)
	if elemType.kind != rtgTypeStruct || destResolved.kind != rtgTypeStruct {
		return false
	}
	elemSize := rtgTypeSize(meta, sliceType.elem)
	if rtgTypeSize(meta, destType) != elemSize {
		return false
	}
	if !rtgEmitIntExpr(g, ep, e.right) {
		return false
	}
	rtgAsmPushRax(a)
	if !rtgEmitSlicePtrLen(g, ep, e.left) {
		return false
	}
	rtgAsmPopRcx(a)
	rtgAsmImulRcxImm(a, elemSize)
	rtgAsmMovRdxRax(a)
	rtgAsmAddRdxRcx(a)
	rtgEmitCopyMemRdxToStack(g, offset, elemSize)
	return true
}

func rtgEmitStructCallToLocal(g *rtgLinearGen, ep *rtgExprParse, idx int, destType int, offset int) bool {
	e := &ep.exprs[idx]
	fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
	if fnIndex < 0 || !rtgTypeIsStruct(g.meta, g.meta.funcs[fnIndex].resultType) {
		return false
	}
	if rtgTypeSize(g.meta, destType) != rtgTypeSize(g.meta, g.meta.funcs[fnIndex].resultType) {
		return false
	}
	wordCount := 1
	for i := e.argCount - 1; i >= 0; i-- {
		words := rtgEmitCallArgReverse(g, ep, ep.args[e.firstArg+i])
		if words < 0 {
			return false
		}
		wordCount += words
	}
	rtgAsmStackMem(&g.asm, offset, 36168, 0x45, 0x85)
	rtgAsmPushRax(&g.asm)
	rtgEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
}

func rtgEmitStructReturnExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	if g.returnStruct <= 0 {
		return false
	}
	e := &ep.exprs[idx]
	resultType := g.meta.funcs[g.currentFunc].resultType
	size := rtgTypeSize(meta, resultType)
	rtgAsmLoadRdxStack(a, g.returnStruct)
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || rtgTypeSize(meta, g.locals[localIndex].typ) != size {
			return false
		}
		rtgEmitCopyStackToMemRdx(g, g.locals[localIndex].offset, 0, size)
		return true
	}
	if e.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, e.left)
		sliceType := rtgResolveType(meta, leftType)
		elemType := rtgResolveType(meta, sliceType.elem)
		if sliceType.kind != rtgTypeSlice || elemType.kind != rtgTypeStruct || rtgTypeSize(meta, sliceType.elem) != size {
			return false
		}
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushRax(a)
		if !rtgEmitSlicePtrLen(g, ep, e.left) {
			return false
		}
		rtgAsmPopRcx(a)
		rtgAsmImulRcxImm(a, size)
		rtgAsmMovRdxRax(a)
		rtgAsmAddRdxRcx(a)
		rtgAsmLoadRcxStack(a, g.returnStruct)
		for at := 0; at < size; at += 8 {
			rtgAsmLoadRaxMemRdxDisp(a, at)
			if at == 0 {
				rtgAsmEmit3(a, 0x48, 0x89, 0x01)
			} else {
				rtgAsmMemDisp(a, at, 35144, 0x41, 0x81)
			}
		}
		return true
	}
	if e.kind == rtgExprComposite {
		rtgAsmMovRaxImm(a, 0)
		for at := 0; at < size; at += 8 {
			rtgAsmStoreRaxMemRdxDisp(a, at)
		}
		for i := 0; i < e.argCount; i++ {
			field := ep.fields[e.firstArg+i]
			fieldIndex := rtgCompositeStructFieldIndex(g, resultType, &field, i)
			if fieldIndex < 0 {
				return false
			}
			fieldOffset := g.meta.fields[fieldIndex].offset
			fieldType := g.meta.fields[fieldIndex].typ
			if fieldType == 0 {
				return false
			}
			if !rtgEmitCompositeFieldToMem(g, ep, field.expr, fieldType, g.returnStruct, fieldOffset) {
				return false
			}
		}
		return true
	}
	if e.kind == rtgExprCall {
		fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
		if fnIndex < 0 || !rtgTypeIsStruct(meta, meta.funcs[fnIndex].resultType) {
			return false
		}
		if rtgTypeSize(meta, meta.funcs[fnIndex].resultType) != size {
			return false
		}
		wordCount := 1
		for i := e.argCount - 1; i >= 0; i-- {
			words := rtgEmitCallArgReverse(g, ep, ep.args[e.firstArg+i])
			if words < 0 {
				return false
			}
			wordCount += words
		}
		rtgAsmLoadRaxStack(a, g.returnStruct)
		rtgAsmPushRax(a)
		rtgEmitCallWithWordCount(g, fnIndex, wordCount)
		return true
	}
	return false
}

func rtgEmitUserCall(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := &ep.exprs[idx]
	fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
	if fnIndex < 0 {
		return rtgEmitNamedConversionCall(g, ep, idx)
	}
	if fnIndex >= len(g.funcLabels) {
		return false
	}
	firstArg := e.firstArg
	argCount := e.argCount
	expanded := e.nameStart
	wordCount := 0
	fn := &g.meta.funcs[fnIndex]
	receiverIndex := -1
	receiverDotTok := 0
	if fn.receiverType != 0 {
		callee := &ep.exprs[e.left]
		if callee.kind != rtgExprSelector {
			return false
		}
		receiverIndex = callee.left
		receiverDotTok = callee.tok
	}
	if expanded == 0 && fn.paramCount > 0 && g.meta.params[fn.firstParam+fn.paramCount-1].initStart == 1 {
		fixed := fn.paramCount - 1
		if receiverIndex >= 0 {
			fixed--
		}
		if argCount < fixed {
			return false
		}
		if receiverIndex >= 0 {
			if !rtgEmitVariadicArgSliceReverse(g, ep, firstArg+fixed, argCount-fixed, g.meta.params[fn.firstParam+fn.paramCount-1].typ) {
				return false
			}
		} else {
			if !rtgEmitVariadicArgSliceFromCallReverse(g, ep, idx, fixed, argCount-fixed, g.meta.params[fn.firstParam+fn.paramCount-1].typ) {
				return false
			}
		}
		wordCount = 3
		for i := fixed - 1; i >= 0; i-- {
			words := rtgEmitCallArgReverse(g, ep, ep.args[firstArg+i])
			if words < 0 {
				return false
			}
			wordCount += words
		}
	} else {
		for i := argCount - 1; i >= 0; i-- {
			words := rtgEmitCallArgReverse(g, ep, ep.args[firstArg+i])
			if words < 0 {
				return false
			}
			wordCount += words
		}
	}
	if receiverIndex >= 0 {
		words := rtgEmitMethodReceiverArgReverse(g, ep, receiverIndex, g.meta.params[fn.firstParam].typ)
		if words < 0 {
			words = rtgEmitMethodReceiverArgTokensReverse(g, receiverDotTok, g.meta.params[fn.firstParam].typ)
			if words < 0 {
				return false
			}
		}
		wordCount += words
	}
	rtgEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
}

func rtgEmitMethodReceiverArgReverse(g *rtgLinearGen, ep *rtgExprParse, idx int, receiverType int) int {
	meta := g.meta
	a := &g.asm
	receiver := rtgResolveType(meta, receiverType)
	exprType := rtgInferParsedExprType(g, ep, idx)
	exprResolved := rtgResolveType(meta, exprType)
	if receiver.kind == rtgTypePointer {
		if exprResolved.kind == rtgTypePointer {
			if !rtgEmitIntExpr(g, ep, idx) {
				return -1
			}
			rtgAsmPushRax(a)
			return 1
		}
		if !rtgEmitAddressRax(g, ep, idx) {
			return -1
		}
		rtgAsmPushRax(a)
		return 1
	}
	if receiver.kind == rtgTypeStruct && exprResolved.kind == rtgTypePointer {
		if !rtgEmitIntExpr(g, ep, idx) {
			return -1
		}
		rtgAsmMovRdxRax(a)
		size := rtgTypeSize(meta, receiverType)
		rtgEmitPushMemRdxWords(g, size)
		return size / 8
	}
	return rtgEmitCallArgReverse(g, ep, idx)
}

func rtgEmitMethodReceiverArgTokensReverse(g *rtgLinearGen, dotTok int, receiverType int) int {
	if dotTok <= 0 {
		return -1
	}
	start := dotTok - 1
	if !rtgTokIsKind(g.prog, start, rtgTokIdent) {
		return -1
	}
	receiverEp := rtgParseExpression(g.prog, start, dotTok)
	if !receiverEp.ok || len(receiverEp.exprs) == 0 {
		return -1
	}
	return rtgEmitMethodReceiverArgReverse(g, &receiverEp, len(receiverEp.exprs)-1, receiverType)
}

func rtgEmitAddressRax(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			rtgAsmLeaRaxStack(a, g.locals[localIndex].offset)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 {
			rtgAsmMovRaxBssAddr(a, globalOffset)
			return true
		}
	}
	if e.kind == rtgExprSelector {
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return false
		}
		rtgAsmMovRaxRdx(a)
		return true
	}
	if e.kind == rtgExprIndex {
		if !rtgEmitIndexAddressRax(g, ep, idx) {
			return false
		}
		return true
	}
	return false
}

func rtgEmitNamedConversionCall(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	calleeExpr := &ep.exprs[e.left]
	if calleeExpr.kind != rtgExprIdent {
		return false
	}
	namedType := rtgFindTypeByRange(g, calleeExpr.nameStart, calleeExpr.nameEnd)
	resolved := rtgResolveType(g.meta, namedType)
	if resolved.kind == rtgTypeString {
		return rtgEmitStringValueRegs(g, ep, ep.args[e.firstArg])
	}
	if resolved.kind == rtgTypeInt || resolved.kind == rtgTypeInt64 || resolved.kind == rtgTypeBool {
		return rtgEmitIntExpr(g, ep, ep.args[e.firstArg])
	}
	if resolved.kind == rtgTypeByte {
		if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
			return false
		}
		rtgAsmEmit8(&g.asm, 0x25)
		rtgAsmEmit32(&g.asm, 255)
		return true
	}
	return false
}

func rtgEmitVariadicArgSliceReverse(g *rtgLinearGen, ep *rtgExprParse, first int, count int, sliceType int) bool {
	fieldFirst := len(ep.fields)
	for i := 0; i < count; i++ {
		var field rtgCompositeField
		field.expr = ep.args[first+i]
		ep.fields = append(ep.fields, field)
	}
	idx := len(ep.exprs)
	var expr rtgExpr
	expr.kind = rtgExprComposite
	expr.firstArg = fieldFirst
	expr.argCount = count
	ep.exprs = append(ep.exprs, expr)
	if !rtgEmitSliceLiteralRegs(g, ep, idx, sliceType) {
		return false
	}
	rtgAsmPushSliceRegs(&g.asm)
	return true
}

func rtgEmitVariadicArgSliceFromCallReverse(g *rtgLinearGen, ep *rtgExprParse, callIdx int, skip int, count int, sliceType int) bool {
	a := &g.asm
	call := &ep.exprs[callIdx]
	t := rtgResolveType(g.meta, sliceType)
	if t.kind != rtgTypeSlice {
		return false
	}
	elem := rtgResolveType(g.meta, t.elem)
	if elem.kind != rtgTypeInt && elem.kind != rtgTypeInt64 && elem.kind != rtgTypeByte && elem.kind != rtgTypeBool {
		return false
	}
	elemSize := rtgTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	backingSize := 32768
	needSize := count * elemSize
	if needSize > backingSize {
		backingSize = rtgAlignTo8(needSize)
	}
	if backingSize < elemSize {
		backingSize = elemSize
	}
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	closeTok := rtgFindMatchingExprClose(g.prog, call.tok+1, ep.end, '(', ')')
	if closeTok <= call.tok {
		return false
	}
	pos := call.tok + 1
	argIndex := 0
	emitted := 0
	for pos < closeTok && emitted < count {
		argEnd := rtgFindExprBoundary(g.prog, pos, closeTok)
		if rtgTokCharIs(g.prog, argEnd, '{') {
			compositeEnd := rtgSkipBalanced(g.prog, argEnd, '{', '}')
			if compositeEnd > argEnd {
				argEnd = compositeEnd
			}
		}
		if argIndex >= skip {
			argEp := rtgParseExpression(g.prog, pos, argEnd)
			if !argEp.ok || len(argEp.exprs) == 0 {
				return false
			}
			rootIndex := len(argEp.exprs) - 1
			if !rtgEmitIntExpr(g, &argEp, rootIndex) {
				return false
			}
			disp := emitted * elemSize
			rtgAsmPushRax(a)
			rtgAsmMovRaxBssAddr(a, backingOff)
			rtgAsmMovRdxRax(a)
			rtgAsmPopRax(a)
			if elemSize == 1 {
				if disp == 0 {
					rtgAsmEmit2(a, 0x88, 0x02)
				} else if rtgAsmImmFits8Signed(disp) {
					rtgAsmEmit3(a, 0x88, 0x42, disp)
				} else {
					rtgAsmEmit16(a, 33416)
					rtgAsmEmit32(a, disp)
				}
			} else {
				rtgAsmStoreRaxMemRdxDisp(a, disp)
			}
			emitted++
		}
		pos = argEnd
		if rtgTokCharIs(g.prog, pos, ',') {
			pos++
		}
		argIndex++
	}
	if emitted != count {
		return false
	}
	capacity := backingSize / elemSize
	rtgAsmMovRaxImm(a, capacity)
	rtgAsmPushRax(a)
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmMovRdxImm(a, count)
	rtgAsmPopRcx(a)
	rtgAsmPushSliceRegs(a)
	return true
}

func rtgEmitCallWithWordCount(g *rtgLinearGen, fnIndex int, wordCount int) {
	a := &g.asm
	if wordCount > 0 {
		rtgAsmPopRdi(a)
	}
	if wordCount > 1 {
		rtgAsmEmit8(a, 0x5e)
	}
	if wordCount > 2 {
		rtgAsmPopRdx(a)
	}
	if wordCount > 3 {
		rtgAsmPopRcx(a)
	}
	if wordCount > 4 {
		rtgAsmEmit16(a, 22593)
	}
	if wordCount > 5 {
		rtgAsmEmit16(a, 22849)
	}
	rtgAsmCallLabel(a, g.funcLabels[fnIndex])
	if wordCount > 6 {
		imm := (wordCount - 6) * 8
		if rtgAsmImmFits8Signed(imm) {
			rtgAsmEmit4(a, 0x48, 0x83, 0xc4, imm)
		} else {
			rtgAsmEmit24(a, 12878152)
			rtgAsmEmit32(a, imm)
		}
	}
}

func rtgEmitCallArgReverse(g *rtgLinearGen, ep *rtgExprParse, idx int) int {
	p := g.prog
	a := &g.asm
	typ := rtgInferParsedExprType(g, ep, idx)
	if rtgTypeIsSlice(g.meta, typ) {
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return -1
		}
		rtgAsmPushSliceRegs(&g.asm)
		return 3
	}
	if rtgTypeIsString(g.meta, typ) {
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return -1
		}
		rtgAsmPushStringRegs(&g.asm)
		return 2
	}
	if rtgTypeIsTuple(g.meta, typ) {
		return rtgEmitTupleArgReverse(g, ep, idx, typ)
	}
	if rtgTypeIsStruct(g.meta, typ) {
		return rtgEmitStructArgReverse(g, ep, idx, typ)
	}
	e := &ep.exprs[idx]
	if e.kind == rtgExprInt {
		value := rtgParseIntToken(p, e.tok)
		rtgAsmPushImm(a, value)
		return 1
	}
	if e.kind == rtgExprChar {
		value := rtgParseCharToken(p, e.tok)
		rtgAsmPushImm(a, value)
		return 1
	}
	if e.kind == rtgExprBool {
		value := rtgBoolTokenValue(p, e.tok)
		rtgAsmPushImm(a, value)
		return 1
	}
	if e.kind == rtgExprIdent {
		constResult := rtgEvalConstByName(g, e.nameStart, e.nameEnd)
		if constResult.ok {
			rtgAsmPushImm(a, constResult.value)
			return 1
		}
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return -1
	}
	rtgAsmPushRax(a)
	return 1
}

func rtgEmitTupleArgReverse(g *rtgLinearGen, ep *rtgExprParse, idx int, typ int) int {
	e := &ep.exprs[idx]
	if e.kind != rtgExprCall {
		return -1
	}
	offset := rtgAddTypedLocal(g, 0, 0, typ)
	if !rtgEmitStructCallToLocal(g, ep, idx, typ, offset) {
		return -1
	}
	tuple := rtgResolveType(g.meta, typ)
	wordCount := 0
	for i := tuple.count - 1; i >= 0; i-- {
		field := g.meta.fields[tuple.first+i]
		size := rtgTypeSize(g.meta, field.typ)
		if size < 8 {
			size = 8
		}
		rtgEmitPushStackWords(g, offset-field.offset, size)
		wordCount += size / 8
	}
	return wordCount
}

func rtgEmitStructArgReverse(g *rtgLinearGen, ep *rtgExprParse, idx int, typ int) int {
	meta := g.meta
	a := &g.asm
	size := rtgTypeSize(meta, typ)
	if size <= 0 {
		return -1
	}
	e := &ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || rtgTypeSize(meta, g.locals[localIndex].typ) != size {
			return -1
		}
		rtgEmitPushStackWords(g, g.locals[localIndex].offset, size)
		return size / 8
	}
	if e.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, e.left)
		sliceType := rtgResolveType(meta, leftType)
		elemType := rtgResolveType(meta, sliceType.elem)
		if sliceType.kind != rtgTypeSlice || elemType.kind != rtgTypeStruct || rtgTypeSize(meta, sliceType.elem) != size {
			return -1
		}
		if !rtgEmitIntExpr(g, ep, e.right) {
			return -1
		}
		rtgAsmPushRax(a)
		if !rtgEmitSlicePtrLen(g, ep, e.left) {
			return -1
		}
		rtgAsmPopRcx(a)
		rtgAsmImulRcxImm(a, size)
		rtgAsmMovRdxRax(a)
		rtgAsmAddRdxRcx(a)
		rtgEmitPushMemRdxWords(g, size)
		return size / 8
	}
	if e.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsStruct(meta, fieldType) || rtgTypeSize(meta, fieldType) != size {
			return -1
		}
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return -1
		}
		rtgEmitPushMemRdxWords(g, size)
		return size / 8
	}
	if e.kind == rtgExprComposite {
		offset := rtgAddTypedLocal(g, 0, 0, typ)
		rtgZeroLocalAtOffset(g, offset)
		for i := 0; i < e.argCount; i++ {
			field := ep.fields[e.firstArg+i]
			fieldIndex := rtgCompositeStructFieldIndex(g, typ, &field, i)
			if fieldIndex < 0 {
				return -1
			}
			fieldOffset := g.meta.fields[fieldIndex].offset
			fieldType := g.meta.fields[fieldIndex].typ
			if !rtgEmitCompositeFieldToStack(g, ep, field.expr, fieldType, offset-fieldOffset) {
				return -1
			}
		}
		rtgEmitPushStackWords(g, offset, size)
		return size / 8
	}
	if e.kind == rtgExprCall {
		offset := rtgAddTypedLocal(g, 0, 0, typ)
		if !rtgEmitStructCallToLocal(g, ep, idx, typ, offset) {
			return -1
		}
		rtgEmitPushStackWords(g, offset, size)
		return size / 8
	}
	return -1
}

func rtgEmitIntExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if (e.kind == rtgExprUnary || e.kind == rtgExprBinary || e.kind == rtgExprCall) && rtgExprCanFoldConst(g, ep, idx) {
		constResult := rtgEvalConstExpr(g, ep, idx)
		if constResult.ok {
			rtgAsmMovRaxImm(a, constResult.value)
			return true
		}
	}
	if e.kind == rtgExprInt {
		rtgAsmMovRaxIntToken(a, p, e.tok)
		return true
	}
	if e.kind == rtgExprFloat {
		value := rtgParseFloatTokenScaled(p, e.tok)
		rtgAsmMovRaxImm(a, value)
		return true
	}
	if e.kind == rtgExprIdent {
		offset := rtgFindLocalOffset(g, e.nameStart, e.nameEnd)
		if offset < 0 {
			constResult := rtgEvalConstByName(g, e.nameStart, e.nameEnd)
			if !constResult.ok {
				globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
				if globalOffset < 0 {
					return false
				}
				rtgAsmLoadRaxBss(a, globalOffset)
				return true
			}
			rtgAsmMovRaxImm(a, constResult.value)
			return true
		}
		rtgAsmLoadRaxStack(a, offset)
		return true
	}
	if e.kind == rtgExprChar {
		value := rtgParseCharToken(p, e.tok)
		rtgAsmMovRaxImm(a, value)
		return true
	}
	if e.kind == rtgExprBool {
		value := rtgBoolTokenValue(p, e.tok)
		rtgAsmMovRaxImm(a, value)
		return true
	}
	if e.kind == rtgExprCall {
		callee := rtgExprIdentCode(p, ep, e.left)
		if e.argCount == 1 && (callee == rtgIdentInt || callee == rtgIdentInt64) {
			return rtgEmitIntExpr(g, ep, ep.args[e.firstArg])
		}
		if e.argCount == 1 && callee == rtgIdentByte {
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmEmit8(a, 0x25)
			rtgAsmEmit32(a, 255)
			return true
		}
		if e.argCount == 1 && callee == rtgIdentLen {
			arg := &ep.exprs[ep.args[e.firstArg]]
			if arg.kind == rtgExprString {
				msg := rtgDecodeStringToken(p, arg.tok)
				msgLen := len(msg)
				rtgAsmMovRaxImm(a, msgLen)
				return true
			}
			if arg.kind == rtgExprIdent {
				localIndex := rtgFindLocalIndex(g, arg.nameStart, arg.nameEnd)
				if localIndex >= 0 && (rtgTypeIsSlice(g.meta, g.locals[localIndex].typ) || rtgTypeIsString(g.meta, g.locals[localIndex].typ)) {
					rtgAsmLoadRaxStack(a, g.locals[localIndex].offset-8)
					return true
				}
				globalOffset := rtgFindGlobalOffset(g, arg.nameStart, arg.nameEnd)
				globalType := rtgFindGlobalType(g, arg.nameStart, arg.nameEnd)
				if globalOffset >= 0 && (rtgTypeIsString(g.meta, globalType) || rtgTypeIsSlice(g.meta, globalType)) {
					rtgAsmLoadRaxBss(a, globalOffset+8)
					return true
				}
				constTok := rtgFindConstStringToken(g, arg.nameStart, arg.nameEnd)
				if constTok >= 0 {
					msg := rtgDecodeStringToken(p, constTok)
					msgLen := len(msg)
					rtgAsmMovRaxImm(a, msgLen)
					return true
				}
			}
			if arg.kind == rtgExprSelector {
				if !rtgEmitSlicePtrLen(g, ep, ep.args[e.firstArg]) {
					return false
				}
				rtgAsmEmit16(a, 22609)
				return true
			}
			if arg.kind == rtgExprUnary && rtgTokCharIs(p, arg.tok, '*') {
				if !rtgEmitIntExpr(g, ep, arg.left) {
					return false
				}
				rtgAsmMovRdxRax(a)
				rtgAsmLoadRaxMemRdxDisp(a, 8)
				return true
			}
		}
		if callee == rtgIdentOpen {
			if e.argCount != 2 {
				return false
			}
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
				return false
			}
			rtgAsmMovRsiRax(a)
			if !rtgEmitStringPtrExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmMovRdiRax(a)
			rtgAsmMovRdxImm(a, 493)
			rtgAsmMovRaxImm(a, 2)
			rtgAsmSyscall(a)
			return true
		}
		if callee == rtgIdentClose {
			if e.argCount != 1 {
				return false
			}
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmMovRdiRax(a)
			rtgAsmMovRaxImm(a, 3)
			rtgAsmSyscall(a)
			return true
		}
		if callee == rtgIdentChmod {
			if e.argCount != 2 {
				return false
			}
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmPushRax(a)
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
				return false
			}
			rtgAsmMovRsiRax(a)
			rtgAsmPopRdi(a)
			rtgAsmMovRaxImm(a, 91)
			rtgAsmSyscall(a)
			return true
		}
		if callee == rtgIdentRead {
			return rtgEmitBuiltinReadWrite(g, ep, idx, 0, 17)
		}
		if callee == rtgIdentWrite {
			return rtgEmitBuiltinReadWrite(g, ep, idx, 1, 18)
		}
		if callee == rtgIdentCopy {
			return rtgEmitBuiltinCopy(g, ep, idx)
		}
		return rtgEmitUserCall(g, ep, idx)
	}
	if e.kind == rtgExprIndex {
		return rtgEmitIndexExpr(g, ep, idx)
	}
	if e.kind == rtgExprSelector {
		base := &ep.exprs[e.left]
		if base.kind == rtgExprCall {
			baseType := rtgInferParsedExprType(g, ep, e.left)
			if !rtgTypeIsStruct(g.meta, baseType) {
				return false
			}
			fieldOffset := rtgStructFieldOffset(g, baseType, e.nameStart, e.nameEnd)
			if fieldOffset < 0 {
				return false
			}
			offset := rtgAddTypedLocal(g, 0, 0, baseType)
			if !rtgEmitStructCallToLocal(g, ep, e.left, baseType, offset) {
				return false
			}
			rtgAsmLoadRaxStack(a, offset-fieldOffset)
			return true
		}
		if base.kind == rtgExprIndex {
			return rtgEmitIndexedStructField(g, ep, e.left, e.nameStart, e.nameEnd)
		}
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return false
		}
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		return true
	}
	if e.kind == rtgExprUnary {
		if rtgTokCharIs(p, e.tok, '&') {
			inner := &ep.exprs[e.left]
			if inner.kind == rtgExprIdent {
				localIndex := rtgFindLocalIndex(g, inner.nameStart, inner.nameEnd)
				if localIndex >= 0 {
					rtgAsmStackMem(a, g.locals[localIndex].offset, 36168, 0x45, 0x85)
					return true
				}
				globalOffset := rtgFindGlobalOffset(g, inner.nameStart, inner.nameEnd)
				if globalOffset >= 0 {
					rtgAsmMovRaxBssAddr(a, globalOffset)
					return true
				}
				return false
			}
			if inner.kind == rtgExprSelector {
				if !rtgEmitSelectorAddressRdx(g, ep, e.left) {
					return false
				}
				rtgAsmEmit16(a, 22610)
				return true
			}
			if inner.kind == rtgExprIndex {
				return rtgEmitIndexAddressRax(g, ep, e.left)
			}
			return false
		}
		if rtgTokCharIs(p, e.tok, '*') {
			leftType := rtgResolveType(g.meta, rtgInferParsedExprType(g, ep, e.left))
			if !rtgEmitIntExpr(g, ep, e.left) {
				return false
			}
			rtgAsmMovRdxRax(a)
			derefType := rtgResolveType(g.meta, rtgInferParsedExprType(g, ep, idx))
			if leftType.kind != rtgTypePointer || derefType.kind == rtgTypeByte || derefType.kind == rtgTypeBool {
				rtgAsmEmit4(a, 0x48, 0x0f, 0xb6, 0x02)
			} else {
				rtgAsmLoadRaxMemRdxDisp(a, 0)
			}
			return true
		}
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		if rtgTokCharIs(p, e.tok, '-') {
			rtgAsmEmit24(a, 14219080)
			return true
		}
		if rtgTokCharIs(p, e.tok, '+') {
			return true
		}
		if rtgTokCharIs(p, e.tok, '!') {
			rtgAsmBoolNotRax(a)
			return true
		}
		return false
	}
	if e.kind == rtgExprBinary {
		if rtgBinaryUsesFloat(g, ep, e) {
			return rtgEmitFloatBinaryExpr(g, ep, idx)
		}
		if rtgTok2Is(p, e.tok, '&', '&') || rtgTok2Is(p, e.tok, '|', '|') {
			falseLabel := rtgAsmNewLabel(a)
			endLabel := rtgAsmNewLabel(a)
			if !rtgEmitJumpIfFalse(g, ep, idx, falseLabel) {
				return false
			}
			rtgAsmMovRaxImm(a, 1)
			rtgAsmJmpLabel(a, endLabel)
			rtgAsmMarkLabel(a, falseLabel)
			rtgAsmMovRaxImm(a, 0)
			rtgAsmMarkLabel(a, endLabel)
			return true
		}
		if rtgTok2Is(p, e.tok, '=', '=') || rtgTok2Is(p, e.tok, '!', '=') {
			leftType := rtgInferParsedExprType(g, ep, e.left)
			rightType := rtgInferParsedExprType(g, ep, e.right)
			if rtgTypeIsString(g.meta, leftType) || rtgTypeIsString(g.meta, rightType) {
				notEqual := rtgTok2Is(p, e.tok, '!', '=')
				return rtgEmitStringCompare(g, ep, e.left, e.right, notEqual)
			}
		}
		rightExpr := &ep.exprs[e.right]
		rightKind := rightExpr.kind
		rightTok := rightExpr.tok
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		rtgAsmPushRax(a)
		if rightKind == rtgExprInt {
			rtgAsmMovRaxIntToken(a, p, rightTok)
		} else if rightKind == rtgExprChar {
			value := rtgParseCharToken(p, rightTok)
			rtgAsmMovRaxImm(a, value)
		} else if rightKind == rtgExprBool {
			value := rtgBoolTokenValue(p, rightTok)
			rtgAsmMovRaxImm(a, value)
		} else {
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
		}
		rtgAsmPopRcx(a)
		if !rtgEmitRaxRcxOp(g, e.tok) {
			return false
		}
		return true
	}
	return false
}

func rtgEmitAppendAssignGeneral(g *rtgLinearGen, stmt *rtgStmt, ep *rtgExprParse) bool {
	p := g.prog
	if len(ep.exprs) == 0 {
		return false
	}
	root := &ep.exprs[len(ep.exprs)-1]
	if root.kind != rtgExprCall || root.argCount != 2 || rtgExprIdentCode(p, ep, root.left) != rtgIdentAppend {
		return false
	}
	var loc rtgSliceLocation
	locEp := ep
	assignTok := rtgFindAssignmentToken(p, stmt.startTok, stmt.endTok)
	if assignTok > stmt.startTok {
		lhs := rtgParseExpression(p, stmt.startTok, assignTok)
		if lhs.ok && len(lhs.exprs) > 0 {
			lhsIndex := len(lhs.exprs) - 1
			rtgSetSliceLocationFromExpr(g, &lhs, lhsIndex, &loc)
			locEp = &lhs
		}
	}
	if !loc.ok {
		rtgSetSliceLocationFromExpr(g, ep, ep.args[root.firstArg], &loc)
		locEp = ep
	}
	if !loc.ok {
		return false
	}
	t := rtgResolveType(g.meta, loc.typ)
	if t.kind != rtgTypeSlice {
		return false
	}
	elem := rtgResolveType(g.meta, t.elem)
	valueIndex := ep.args[root.firstArg+1]
	if root.nameStart == 1 {
		return rtgEmitAppendExpansionToLocation(g, ep, locEp, &loc, t.elem, valueIndex)
	}
	if elem.kind == rtgTypeStruct {
		value := &ep.exprs[valueIndex]
		if value.kind != rtgExprComposite {
			if value.kind == rtgExprUnary && rtgTokCharIs(p, value.tok, '*') {
				return rtgEmitAppendStructDeref(g, ep, locEp, &loc, t.elem, valueIndex)
			}
			if value.kind == rtgExprIdent {
				typeTok := value.tok
				if !rtgTokCharIs(p, typeTok+1, '{') {
					typeTok = 0
					for i := 0; i < len(p.toks); i++ {
						if p.toks[i].start == value.nameStart {
							typeTok = i
						}
					}
				}
				if rtgTokCharIs(p, typeTok+1, '{') {
					return rtgEmitAppendStructCompositeTokens(g, locEp, &loc, t.elem, typeTok)
				}
				return rtgEmitAppendStructLocal(g, ep, locEp, &loc, t.elem, valueIndex)
			}
			typeTok := rtgFindAppendCompositeTypeToken(p, root.tok, stmt.endTok)
			if typeTok >= 0 {
				return rtgEmitAppendStructCompositeTokens(g, locEp, &loc, t.elem, typeTok)
			}
			return false
		}
		if !rtgEmitAppendStructComposite(g, ep, locEp, &loc, t.elem, valueIndex) {
			return false
		}
		return true
	}
	if elem.kind == rtgTypeInt || elem.kind == rtgTypeByte || elem.kind == rtgTypeBool {
		if !rtgEmitAppendScalarToLocation(g, ep, locEp, &loc, elem.kind, valueIndex) {
			return false
		}
		return true
	}
	if elem.kind == rtgTypeString {
		if !rtgEmitAppendStringToLocation(g, ep, locEp, &loc, valueIndex) {
			return false
		}
		return true
	}
	return false
}

func rtgBinaryUsesFloat(g *rtgLinearGen, ep *rtgExprParse, e *rtgExpr) bool {
	p := g.prog
	if rtgTok2Is(p, e.tok, '&', '&') || rtgTok2Is(p, e.tok, '|', '|') {
		return false
	}
	left := rtgInferParsedExprType(g, ep, e.left)
	if left == rtgTypeFloat64 {
		return true
	}
	right := rtgInferParsedExprType(g, ep, e.right)
	if right == rtgTypeFloat64 {
		return true
	}
	if !ep.hasFloat {
		return false
	}
	if rtgExprValueIsFloat(g, ep, e.left) {
		return true
	}
	return rtgExprValueIsFloat(g, ep, e.right)
}

func rtgExprValueIsFloat(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	p := g.prog
	e := &ep.exprs[idx]
	if e.kind == rtgExprFloat {
		return true
	}
	if e.kind == rtgExprUnary {
		if rtgTokCharIs(p, e.tok, '+') || rtgTokCharIs(p, e.tok, '-') {
			return rtgExprValueIsFloat(g, ep, e.left)
		}
		typ := rtgResolveType(g.meta, rtgInferParsedExprType(g, ep, idx))
		return typ.kind == rtgTypeFloat64
	}
	if e.kind == rtgExprBinary {
		if rtgTok2Is(p, e.tok, '=', '=') || rtgTok2Is(p, e.tok, '!', '=') || rtgTokCharIs(p, e.tok, '<') || rtgTokCharIs(p, e.tok, '>') || rtgTok2Is(p, e.tok, '&', '&') || rtgTok2Is(p, e.tok, '|', '|') {
			return false
		}
		if rtgExprValueIsFloat(g, ep, e.left) {
			return true
		}
		return rtgExprValueIsFloat(g, ep, e.right)
	}
	if e.kind == rtgExprIdent || e.kind == rtgExprCall || e.kind == rtgExprIndex || e.kind == rtgExprSelector {
		typ := rtgResolveType(g.meta, rtgInferParsedExprType(g, ep, idx))
		return typ.kind == rtgTypeFloat64
	}
	return false
}

func rtgEmitFloatBinaryExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if rtgTokCharIs(p, e.tok, '*') {
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		rtgAsmPushRax(a)
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPopRcx(a)
		rtgAsmEmit32(a, -1045491896)
		rtgAsmSarRaxImm(a, 2)
		return true
	}
	if rtgTokCharIs(p, e.tok, '/') {
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		rtgAsmShlRaxImm(a, 2)
		rtgAsmPushRax(a)
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPopRcx(a)
		rtgAsmDivLeftRcxRightRax(a, false)
		return true
	}
	if !rtgEmitIntExpr(g, ep, e.left) {
		return false
	}
	rtgAsmPushRax(a)
	if !rtgEmitIntExpr(g, ep, e.right) {
		return false
	}
	rtgAsmPopRcx(a)
	return rtgEmitRaxRcxOp(g, e.tok)
}

func rtgExprCanFoldConst(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := &ep.exprs[idx]
	if e.kind == rtgExprInt || e.kind == rtgExprChar || e.kind == rtgExprBool {
		return true
	}
	if e.kind == rtgExprIdent {
		if rtgFindLocalIndex(g, e.nameStart, e.nameEnd) >= 0 {
			return false
		}
		builtin := rtgEvalBuiltinConst(g, e.nameStart, e.nameEnd)
		if builtin.ok {
			return true
		}
		for i := 0; i < len(g.meta.globals); i++ {
			s := &g.meta.globals[i]
			if s.kind == rtgTokConst && rtgBytesEqualRange(g.prog.src, s.nameStart, s.nameEnd, e.nameStart, e.nameEnd) {
				return true
			}
		}
		return false
	}
	if e.kind == rtgExprUnary {
		return rtgExprCanFoldConst(g, ep, e.left)
	}
	if e.kind == rtgExprBinary {
		return rtgExprCanFoldConst(g, ep, e.left) && rtgExprCanFoldConst(g, ep, e.right)
	}
	if e.kind == rtgExprCall {
		if e.argCount != 1 || !rtgExprCanFoldConst(g, ep, ep.args[e.firstArg]) {
			return false
		}
		callee := rtgExprIdentCode(g.prog, ep, e.left)
		if callee == rtgIdentInt || callee == rtgIdentByte || callee == rtgIdentInt64 {
			return true
		}
		calleeExpr := &ep.exprs[e.left]
		if calleeExpr.kind == rtgExprIdent {
			return rtgFindTypeByRange(g, calleeExpr.nameStart, calleeExpr.nameEnd) > 0
		}
	}
	return false
}

func rtgEmitAppendExpansionToLocation(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, elemType int, valueIndex int) bool {
	a := &g.asm
	elemSize := rtgTypeSize(g.meta, elemType)
	if elemSize < 1 {
		elemSize = 8
	}
	if elemSize != 1 && elemSize%8 != 0 {
		return false
	}
	sourceType := rtgInferParsedExprType(g, ep, valueIndex)
	source := rtgResolveType(g.meta, sourceType)
	if source.kind != rtgTypeSlice {
		return false
	}
	if rtgTypeSize(g.meta, source.elem) != elemSize {
		return false
	}
	if elemSize == 1 {
		if !rtgEmitSliceValueRegs(g, ep, valueIndex) {
			return false
		}
		rtgAsmPushRax(a)
		rtgAsmPushRdx(a)
		if !rtgEmitSliceSlotAddrs(g, locEp, loc, elemSize) {
			return false
		}
		rtgAsmPopRdx(a)
		rtgAsmPopRax(a)
		label := rtgEnsureAppendBytesHelper(g)
		rtgAsmCallLabel(a, label)
		return true
	}
	srcPtr := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	srcLen := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	srcIndex := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	destPtr := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	destLen := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	headerOffset := 0
	if !rtgEmitSliceValueRegs(g, ep, valueIndex) {
		return false
	}
	rtgAsmStoreRaxStack(a, srcPtr)
	rtgAsmStoreRdxStack(a, srcLen)
	rtgAsmMovRaxImm(a, 0)
	rtgAsmStoreRaxStack(a, srcIndex)
	if loc.mem {
		if loc.expr < 0 || loc.expr >= len(locEp.exprs) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, locEp, loc.expr) {
			return false
		}
		headerOffset = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAsmStoreRdxStack(a, headerOffset)
		rtgEmitEnsureMemSlice(g, elemSize)
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		rtgAsmStoreRaxStack(a, destPtr)
		rtgAsmLoadRdxStack(a, headerOffset)
		rtgAsmLoadRaxMemRdxDisp(a, 8)
		rtgAsmStoreRaxStack(a, destLen)
	} else if loc.global {
		rtgAsmLoadRaxBss(a, loc.offset)
		rtgAsmStoreRaxStack(a, destPtr)
		rtgAsmLoadRaxBss(a, loc.offset+8)
		rtgAsmStoreRaxStack(a, destLen)
	} else {
		rtgAsmLoadRaxStack(a, loc.offset)
		rtgAsmStoreRaxStack(a, destPtr)
		rtgAsmLoadRaxStack(a, loc.offset-8)
		rtgAsmStoreRaxStack(a, destLen)
	}
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmLoadRaxStack(a, srcIndex)
	rtgAsmPushRax(a)
	rtgAsmLoadRaxStack(a, srcLen)
	rtgAsmPopRcx(a)
	rtgAsmCmpRcxRaxSet(a, 0x9d)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, doneLabel)
	rtgEmitAppendExpansionCopyElement(g, elemSize, srcPtr, srcIndex, destPtr, destLen)
	rtgAsmLoadRaxStack(a, srcIndex)
	rtgAsmIncRax(a)
	rtgAsmStoreRaxStack(a, srcIndex)
	rtgAsmLoadRaxStack(a, destLen)
	rtgAsmIncRax(a)
	rtgAsmStoreRaxStack(a, destLen)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmLoadRaxStack(a, destLen)
	if loc.mem {
		rtgAsmLoadRdxStack(a, headerOffset)
		rtgAsmStoreRaxMemRdxDisp(a, 8)
	} else if loc.global {
		rtgAsmStoreRaxBss(a, loc.offset+8)
	} else {
		rtgAsmStoreRaxStack(a, loc.offset-8)
	}
	return true
}

func rtgEmitAppendExpansionCopyElement(g *rtgLinearGen, elemSize int, srcPtr int, srcIndex int, destPtr int, destLen int) {
	a := &g.asm
	if elemSize == 1 {
		rtgAsmLoadRaxStack(a, srcPtr)
		rtgAsmLoadRcxStack(a, srcIndex)
		rtgAsmLoadByteRaxIndexRcx(a)
		rtgAsmPushRax(a)
		rtgAsmLoadRdxStack(a, destPtr)
		rtgAsmLoadRcxStack(a, destLen)
		rtgAsmPopRax(a)
		rtgAsmStoreAlMemRdxRcx1(a)
		return
	}
	if elemSize == 8 {
		rtgAsmLoadRaxStack(a, srcPtr)
		rtgAsmLoadRcxStack(a, srcIndex)
		rtgAsmLoadQwordRaxIndexRcx8(a)
		rtgAsmPushRax(a)
		rtgAsmLoadRdxStack(a, destPtr)
		rtgAsmLoadRcxStack(a, destLen)
		rtgAsmPopRax(a)
		rtgAsmStoreRaxMemRdxRcx8(a)
		return
	}
	for copyOff := 0; copyOff < elemSize; copyOff += 8 {
		rtgAsmLoadRaxStack(a, srcPtr)
		rtgAsmLoadRcxStack(a, srcIndex)
		rtgAsmImulRcxImm(a, elemSize)
		rtgAsmLoadQwordRaxIndexRcxDisp(a, copyOff)
		rtgAsmPushRax(a)
		rtgAsmLoadRdxStack(a, destPtr)
		rtgAsmLoadRcxStack(a, destLen)
		rtgAsmImulRcxImm(a, elemSize)
		rtgAsmAddRdxRcx(a)
		rtgAsmPopRax(a)
		rtgAsmStoreRaxMemRdxDisp(a, copyOff)
	}
}

func rtgFindAppendCompositeTypeToken(p *rtgProgram, openTok int, end int) int {
	if openTok < 0 || openTok >= end || !rtgTokCharIs(p, openTok, '(') {
		return -1
	}
	i := openTok + 1
	paren := 0
	brack := 0
	brace := 0
	for i < end {
		if paren == 0 && brack == 0 && brace == 0 && rtgTokCharIs(p, i, ',') {
			typeTok := i + 1
			if rtgTokIsKind(p, typeTok, rtgTokIdent) && rtgTokCharIs(p, typeTok+1, '{') {
				return typeTok
			}
			return -1
		}
		if rtgTokCharIs(p, i, '(') {
			paren++
		} else if rtgTokCharIs(p, i, ')') {
			if paren == 0 {
				return -1
			}
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
		i++
	}
	return -1
}

func rtgEmitAppendScalarToLocation(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, elemKind int, valueIndex int) bool {
	a := &g.asm
	label := rtgEnsureAppendScalarHelper(g, elemKind)
	elemSize := 8
	if elemKind == rtgTypeByte || elemKind == rtgTypeBool {
		elemSize = 1
	}
	if !rtgEmitIntExpr(g, ep, valueIndex) {
		return false
	}
	rtgAsmPushRax(a)
	if !rtgEmitSliceSlotAddrs(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmPopRdx(a)
	rtgAsmCallLabel(a, label)
	return true
}

func rtgEmitSliceSlotAddrs(g *rtgLinearGen, locEp *rtgExprParse, loc *rtgSliceLocation, elemSize int) bool {
	a := &g.asm
	if loc.mem {
		if loc.expr < 0 || loc.expr >= len(locEp.exprs) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, locEp, loc.expr) {
			return false
		}
		rtgEmitEnsureMemSlice(g, elemSize)
		rtgAsmEmit16(a, 24402)
		rtgAsmEmit24(a, 7507272)
		rtgAsmEmit8(a, 8)
		return true
	}
	if loc.global {
		rtgAsmEmit24(a, 4033864)
		at := len(a.code)
		rtgAsmEmit32(a, 0)
		rtgAsmAddAbsReloc(a, at, loc.offset, 4)
		rtgAsmEmit24(a, 3509576)
		at = len(a.code)
		rtgAsmEmit32(a, 0)
		rtgAsmAddAbsReloc(a, at, loc.offset+8, 4)
		return true
	}
	rtgAsmLeaRdiStack(a, loc.offset)
	rtgAsmLeaRsiStack(a, loc.offset-8)
	return true
}

func rtgEmitAppendDestRax(g *rtgLinearGen, locEp *rtgExprParse, loc *rtgSliceLocation, elemSize int) bool {
	label := rtgEnsureAppendAddrHelper(g)
	if !rtgEmitSliceSlotAddrs(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmMovRdxImm(&g.asm, elemSize)
	rtgAsmCallLabel(&g.asm, label)
	return true
}

func rtgEnsureAppendScalarHelper(g *rtgLinearGen, elemKind int) int {
	if elemKind == rtgTypeByte || elemKind == rtgTypeBool {
		return rtgEnsureAppend8Helper(g)
	}
	return rtgEnsureAppend64Helper(g)
}

func rtgEnsureAppendAddrHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.appendAddrEmitted {
		return g.appendAddrLabel
	}
	g.appendAddrEmitted = true
	g.appendAddrLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.appendAddrLabel)
	rtgAsmEmit24(a, 953160)
	rtgAsmEmit24(a, 494408)
	rtgAsmEmit32(a, -894496952)
	rtgAsmAddRaxRcx(a)
	rtgAsmEmit24(a, 953160)
	rtgAsmIncRcx(a)
	rtgAsmEmit24(a, 952648)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.appendAddrLabel
}

func rtgEnsureAppend8Helper(g *rtgLinearGen) int {
	a := &g.asm
	if g.append8Emitted {
		return g.append8Label
	}
	g.append8Emitted = true
	g.append8Label = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.append8Label)
	rtgAsmEmit24(a, 953160)
	rtgAsmEmit24(a, 494412)
	rtgAsmEmit32(a, 135563329)
	rtgAsmIncRcx(a)
	rtgAsmEmit24(a, 952648)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.append8Label
}

func rtgEnsureAppend64Helper(g *rtgLinearGen) int {
	a := &g.asm
	if g.append64Emitted {
		return g.append64Label
	}
	g.append64Emitted = true
	g.append64Label = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.append64Label)
	rtgAsmEmit24(a, 953160)
	rtgAsmEmit24(a, 494412)
	rtgAsmEmit32(a, -938178231)
	rtgAsmIncRcx(a)
	rtgAsmEmit24(a, 952648)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.append64Label
}

func rtgEnsureAppendBytesHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.appendBytesEmitted {
		return g.appendBytesLabel
	}
	g.appendBytesEmitted = true
	g.appendBytesLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.appendBytesLabel)
	rtgAsmEmit3(a, 0x48, 0x8b, 0x0e)
	rtgAsmEmit3(a, 0x48, 0x8b, 0x3f)
	rtgAsmEmit3(a, 0x48, 0x01, 0xcf)
	rtgAsmEmit3(a, 0x48, 0x01, 0x16)
	rtgAsmEmit3(a, 0x48, 0x89, 0xc6)
	rtgAsmEmit3(a, 0x48, 0x89, 0xd1)
	rtgAsmEmit2(a, 0xf3, 0xa4)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.appendBytesLabel
}

func rtgEnsureCopyWordsHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.copyWordsEmitted {
		return g.copyWordsLabel
	}
	g.copyWordsEmitted = true
	g.copyWordsLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.copyWordsLabel)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmEmit3(a, 0x48, 0x85, 0xd2)
	rtgAsmJzLabel(a, doneLabel)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmEmit3(a, 0x48, 0x8b, 0x06)
	rtgAsmEmit3(a, 0x48, 0x89, 0x07)
	rtgAsmEmit4(a, 0x48, 0x83, 0xc6, 8)
	rtgAsmEmit4(a, 0x48, 0x83, 0xc7, 8)
	rtgAsmEmit3(a, 0x48, 0xff, 0xca)
	rtgAsmJnzLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.copyWordsLabel
}

func rtgEmitAppendStringToLocation(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, valueIndex int) bool {
	a := &g.asm
	rtgEnsureAppendAddrHelper(g)
	if !rtgEmitStringValueRegs(g, ep, valueIndex) {
		return false
	}
	rtgAsmPushStringRegs(a)
	if !rtgEmitAppendDestRax(g, locEp, loc, 16) {
		return false
	}
	rtgAsmMovRdxRax(a)
	rtgAsmPopStoreStringMemRdx(a, 0)
	return true
}

func rtgSetSliceLocationFromExpr(g *rtgLinearGen, ep *rtgExprParse, idx int, loc *rtgSliceLocation) {
	e := &ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
			globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalOffset < 0 || !rtgTypeIsSlice(g.meta, globalType) {
				return
			}
			loc.offset = globalOffset
			loc.typ = globalType
			loc.global = true
			loc.ok = true
			return
		}
		if !rtgTypeIsSlice(g.meta, g.locals[localIndex].typ) {
			return
		}
		loc.offset = g.locals[localIndex].offset
		loc.typ = g.locals[localIndex].typ
		loc.ok = true
		return
	}
	if e.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(g.meta, fieldType) {
			return
		}
		loc.expr = idx
		loc.typ = fieldType
		loc.mem = true
		loc.ok = true
		return
	}
}

func rtgEmitEnsureMemSlice(g *rtgLinearGen, elemSize int) {
	a := &g.asm
	if elemSize < 1 {
		elemSize = 8
	}
	okLabel := rtgAsmNewLabel(a)
	rtgAsmLoadRaxMemRdxDisp(a, 0)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, okLabel)
	backingSize := 8388608
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmStoreRaxMemRdxDisp(a, 0)
	rtgAsmMovRaxImm(a, backingSize/elemSize)
	rtgAsmStoreRaxMemRdxDisp(a, 16)
	rtgAsmMarkLabel(a, okLabel)
}

func rtgEmitAppendStructCompositeTokens(g *rtgLinearGen, locEp *rtgExprParse, loc *rtgSliceLocation, elemType int, typeTok int) bool {
	p := g.prog
	openTok := typeTok + 1
	closeTok := rtgSkipBalanced(p, openTok, '{', '}')
	if closeTok <= openTok {
		return false
	}
	elemSize := rtgTypeSize(g.meta, elemType)
	if !rtgEmitAppendDestRax(g, locEp, loc, elemSize) {
		return false
	}
	destOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	rtgAsmStoreRaxStack(&g.asm, destOffset)
	i := openTok + 1
	for i < closeTok-1 {
		if !rtgTokIsKind(p, i, rtgTokIdent) || !rtgTokCharIs(p, i+1, ':') {
			return false
		}
		fieldTok := p.toks[i]
		exprStart := i + 2
		exprEnd := rtgFindExprBoundary(p, exprStart, closeTok-1)
		ep := rtgParseExpression(p, exprStart, exprEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		fieldOffset := rtgStructFieldOffset(g, elemType, fieldTok.start, fieldTok.end)
		if fieldOffset < 0 {
			return false
		}
		fieldType := rtgStructFieldType(g, elemType, fieldTok.start, fieldTok.end)
		if fieldType == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if !rtgEmitCompositeFieldToMem(g, &ep, rootIndex, fieldType, destOffset, fieldOffset) {
			return false
		}
		i = exprEnd
		if rtgTokCharIs(p, i, ',') {
			i++
		}
	}
	return true
}

func rtgEmitAppendStructDeref(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, elemType int, valueIndex int) bool {
	a := &g.asm
	value := &ep.exprs[valueIndex]
	valueType := rtgInferParsedExprType(g, ep, valueIndex)
	if !rtgTypeIsStruct(g.meta, valueType) || rtgTypeSize(g.meta, valueType) != rtgTypeSize(g.meta, elemType) {
		return false
	}
	elemSize := rtgTypeSize(g.meta, elemType)
	tempOffset := rtgAddTypedLocal(g, 0, 0, elemType)
	if !rtgEmitIntExpr(g, ep, value.left) {
		return false
	}
	rtgAsmMovRdxRax(a)
	rtgEmitCopyMemRdxToStack(g, tempOffset, elemSize)
	if !rtgEmitAppendDestRax(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmMovRdxRax(a)
	rtgEmitCopyStackToMemRdx(g, tempOffset, 0, elemSize)
	return true
}

func rtgEmitAppendStructLocal(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, elemType int, valueIndex int) bool {
	value := &ep.exprs[valueIndex]
	localIndex := rtgFindLocalIndex(g, value.nameStart, value.nameEnd)
	if localIndex < 0 {
		return false
	}
	elemSize := rtgTypeSize(g.meta, elemType)
	if rtgTypeSize(g.meta, g.locals[localIndex].typ) != elemSize {
		return false
	}
	if !rtgEmitAppendDestRax(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmMovRdxRax(&g.asm)
	rtgEmitCopyStackToMemRdx(g, g.locals[localIndex].offset, 0, elemSize)
	return true
}

func rtgEmitAppendStructComposite(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, elemType int, valueIndex int) bool {
	elemSize := rtgTypeSize(g.meta, elemType)
	tempOffset := rtgAddTypedLocal(g, 0, 0, elemType)
	if !rtgEmitTypedAssign(g, ep, valueIndex, tempOffset) {
		return false
	}
	if !rtgEmitAppendDestRax(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmMovRdxRax(&g.asm)
	rtgEmitCopyStackToMemRdx(g, tempOffset, 0, elemSize)
	return true
}

func rtgEmitStringCompare(g *rtgLinearGen, ep *rtgExprParse, left int, right int, notEqual bool) bool {
	a := &g.asm
	label := rtgEnsureStringEqualHelper(g)
	if !rtgEmitStringValueRegs(g, ep, left) {
		return false
	}
	rtgAsmPushStringRegs(a)
	if !rtgEmitStringValueRegs(g, ep, right) {
		return false
	}
	rtgAsmMovRcxRdx(a)
	rtgAsmMovRdxRax(a)
	rtgAsmPopRdi(a)
	rtgAsmEmit8(a, 0x5e)
	rtgAsmCallLabel(a, label)
	if notEqual {
		rtgAsmBoolNotRax(a)
	}
	return true
}

func rtgEnsureStringEqualHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.streqEmitted {
		return g.streqLabel
	}
	g.streqEmitted = true
	g.streqLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.streqLabel)
	notEqualLabel := rtgAsmNewLabel(a)
	equalLabel := rtgAsmNewLabel(a)
	loopLabel := rtgAsmNewLabel(a)
	rtgAsmMovRaxImm(a, 0)
	rtgAsmEmit24(a, 13515080)
	rtgAsmJnzLabel(a, notEqualLabel)
	rtgAsmEmit24(a, 16155976)
	rtgAsmJzLabel(a, equalLabel)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmEmit24(a, 494148)
	rtgAsmEmit24(a, 145476)
	rtgAsmJnzLabel(a, notEqualLabel)
	rtgAsmEmit24(a, 13107016)
	rtgAsmEmit24(a, 12779336)
	rtgAsmEmit24(a, 13565768)
	rtgAsmJnzLabel(a, loopLabel)
	rtgAsmMarkLabel(a, equalLabel)
	rtgAsmMovRaxImm(a, 1)
	rtgAsmMarkLabel(a, notEqualLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.streqLabel
}

func rtgEmitBuiltinReadWrite(g *rtgLinearGen, ep *rtgExprParse, idx int, seqSyscall int, offSyscall int) bool {
	a := &g.asm
	p := g.prog
	firstArg := ep.exprs[idx].firstArg
	argCount := ep.exprs[idx].argCount
	if argCount != 3 {
		return false
	}
	fdStart := ep.exprs[idx].tok + 1
	fdEnd := rtgFindExprBoundary(p, fdStart, ep.end)
	fdEp := rtgParseExpression(p, fdStart, fdEnd)
	if !fdEp.ok || len(fdEp.exprs) == 0 {
		return false
	}
	fdIndex := len(fdEp.exprs) - 1
	if !rtgEmitIntExpr(g, &fdEp, fdIndex) {
		return false
	}
	rtgAsmPushRax(a)
	offIndex := ep.args[firstArg+2]
	offConst := rtgEvalConstExpr(g, ep, offIndex)
	offsetRead := true
	if offConst.ok && offConst.value < 0 {
		offsetRead = false
	}
	if offsetRead {
		if offConst.ok {
			rtgAsmMovRaxImm(a, offConst.value)
		} else {
			if !rtgEmitIntExpr(g, ep, offIndex) {
				return false
			}
		}
		rtgAsmPushRax(a)
	}
	if !rtgEmitSlicePtrLen(g, ep, ep.args[firstArg+1]) {
		return false
	}
	rtgAsmMovRsiRax(a)
	rtgAsmEmit16(a, 23121)
	if offsetRead {
		rtgAsmPopRax(a)
		rtgAsmEmit3(a, 0x49, 0x89, 0xc2)
	}
	rtgAsmPopRdi(a)
	if offsetRead {
		rtgAsmMovRaxImm(a, offSyscall)
	} else {
		rtgAsmMovRaxImm(a, seqSyscall)
	}
	rtgAsmSyscall(a)
	return true
}

func rtgEmitBuiltinCopy(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.argCount != 2 {
		return false
	}
	destIndex := ep.args[e.firstArg]
	srcIndex := ep.args[e.firstArg+1]
	destType := rtgInferParsedExprType(g, ep, destIndex)
	srcType := rtgInferParsedExprType(g, ep, srcIndex)
	destSlice := rtgResolveType(meta, destType)
	srcSlice := rtgResolveType(meta, srcType)
	if destSlice.kind != rtgTypeSlice || srcSlice.kind != rtgTypeSlice {
		return false
	}
	elemSize := rtgTypeSize(meta, destSlice.elem)
	if elemSize != rtgTypeSize(meta, srcSlice.elem) {
		return false
	}
	if elemSize < 1 {
		elemSize = 8
	}
	destPtr := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	destLen := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	srcPtr := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	srcLen := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	count := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	if !rtgEmitSliceValueRegs(g, ep, destIndex) {
		return false
	}
	rtgAsmStoreRaxStack(a, destPtr)
	rtgAsmStoreRdxStack(a, destLen)
	if !rtgEmitSliceValueRegs(g, ep, srcIndex) {
		return false
	}
	rtgAsmStoreRaxStack(a, srcPtr)
	rtgAsmStoreRdxStack(a, srcLen)
	rtgAsmMovRaxImm(a, 0)
	rtgAsmStoreRaxStack(a, count)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmLoadRaxStack(a, count)
	rtgAsmPushRax(a)
	rtgAsmLoadRaxStack(a, destLen)
	rtgAsmPopRcx(a)
	rtgAsmCmpRcxRaxSet(a, 0x9d)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, doneLabel)
	rtgAsmLoadRaxStack(a, count)
	rtgAsmPushRax(a)
	rtgAsmLoadRaxStack(a, srcLen)
	rtgAsmPopRcx(a)
	rtgAsmCmpRcxRaxSet(a, 0x9d)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, doneLabel)
	rtgEmitAppendExpansionCopyElement(g, elemSize, srcPtr, count, destPtr, count)
	rtgAsmLoadRaxStack(a, count)
	rtgAsmIncRax(a)
	rtgAsmStoreRaxStack(a, count)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmLoadRaxStack(a, count)
	return true
}

func rtgEmitSliceBasePtrLenTokens(g *rtgLinearGen, p *rtgProgram, start int, end int, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	if start+1 == end && rtgTokIsKind(p, start, rtgTokIdent) {
		nameStart := p.toks[start].start
		nameEnd := p.toks[start].end
		localIndex := rtgFindLocalIndex(g, nameStart, nameEnd)
		if localIndex >= 0 {
			if !rtgTypeIsSlice(meta, g.locals[localIndex].typ) {
				return false
			}
			rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
			rtgAsmLoadRcxStack(a, g.locals[localIndex].offset-8)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, nameStart, nameEnd)
		globalType := rtgFindGlobalType(g, nameStart, nameEnd)
		if globalOffset >= 0 && rtgTypeIsSlice(meta, globalType) {
			rtgAsmLoadRaxBss(a, globalOffset+8)
			rtgAsmMovRcxRax(a)
			rtgAsmLoadRaxBss(a, globalOffset)
			return true
		}
		return false
	}
	if start+3 == end && rtgTokIsKind(p, start, rtgTokIdent) && rtgTokCharIs(p, start+1, '.') && rtgTokIsKind(p, start+2, rtgTokIdent) {
		localIndex := rtgFindLocalIndex(g, p.toks[start].start, p.toks[start].end)
		if localIndex < 0 {
			return false
		}
		fieldType := rtgStructFieldType(g, g.locals[localIndex].typ, p.toks[start+2].start, p.toks[start+2].end)
		if !rtgTypeIsSlice(meta, fieldType) {
			return false
		}
		fieldOffset := rtgStructFieldOffset(g, g.locals[localIndex].typ, p.toks[start+2].start, p.toks[start+2].end)
		if fieldOffset < 0 {
			return false
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind == rtgTypePointer {
			rtgAsmLoadRdxStack(a, g.locals[localIndex].offset)
			if fieldOffset != 0 {
				rtgAsmAddRdxImm(a, fieldOffset)
			}
		} else {
			rtgAsmStackMem(a, g.locals[localIndex].offset-fieldOffset, 36168, 0x55, 0x95)
		}
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		rtgAsmMemDisp(a, 8, 35656, 0x4a, 0x8a)
		return true
	}
	return rtgEmitSlicePtrLen(g, ep, idx)
}

func rtgEmitSlicePtrLen(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprSlice {
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmMovRcxRdx(a)
		return true
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
			globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalOffset < 0 || (!rtgTypeIsSlice(meta, globalType) && !rtgTypeIsString(meta, globalType)) {
				return false
			}
			rtgAsmLoadRaxBss(a, globalOffset+8)
			rtgAsmMovRcxRax(a)
			rtgAsmLoadRaxBss(a, globalOffset)
			return true
		}
		if !rtgTypeIsSlice(meta, g.locals[localIndex].typ) && !rtgTypeIsString(meta, g.locals[localIndex].typ) {
			return false
		}
		rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
		rtgAsmLoadRcxStack(a, g.locals[localIndex].offset-8)
		return true
	}
	if e.kind == rtgExprComposite {
		sliceType := rtgTypeFromExpr(g, ep, idx)
		if !rtgTypeIsSlice(meta, sliceType) {
			return false
		}
		if !rtgEmitSliceLiteralRegs(g, ep, idx, sliceType) {
			return false
		}
		rtgAsmMovRcxRdx(a)
		return true
	}
	if e.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(meta, fieldType) && !rtgTypeIsString(meta, fieldType) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return false
		}
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		rtgAsmMemDisp(a, 8, 35656, 0x4a, 0x8a)
		return true
	}
	return false
}

func rtgEmitIndexedStructField(g *rtgLinearGen, ep *rtgExprParse, indexIdx int, fieldStart int, fieldEnd int) bool {
	a := &g.asm
	indexExpr := &ep.exprs[indexIdx]
	leftIndex := indexExpr.left
	rightIndex := indexExpr.right
	leftType := rtgInferParsedExprType(g, ep, leftIndex)
	sliceType := rtgResolveType(g.meta, leftType)
	if sliceType.kind != rtgTypeSlice {
		return false
	}
	elemType := rtgResolveType(g.meta, sliceType.elem)
	if elemType.kind != rtgTypeStruct {
		return false
	}
	fieldOffset := rtgStructFieldOffset(g, sliceType.elem, fieldStart, fieldEnd)
	if fieldOffset < 0 {
		return false
	}
	if !rtgEmitIntExpr(g, ep, rightIndex) {
		return false
	}
	rtgAsmPushRax(a)
	if !rtgEmitSlicePtrLen(g, ep, leftIndex) {
		return false
	}
	rtgAsmPopRcx(a)
	elemSize := rtgTypeSize(g.meta, sliceType.elem)
	rtgAsmImulRcxImm(a, elemSize)
	rtgAsmLoadQwordRaxIndexRcxDisp(a, fieldOffset)
	return true
}

func rtgEmitIndexAddressRax(g *rtgLinearGen, ep *rtgExprParse, indexIdx int) bool {
	a := &g.asm
	indexExpr := &ep.exprs[indexIdx]
	leftType := rtgInferParsedExprType(g, ep, indexExpr.left)
	sliceType := rtgResolveType(g.meta, leftType)
	if sliceType.kind != rtgTypeSlice {
		return false
	}
	elemSize := rtgTypeSize(g.meta, sliceType.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	if !rtgEmitIntExpr(g, ep, indexExpr.right) {
		return false
	}
	rtgAsmPushRax(a)
	if !rtgEmitSlicePtrLen(g, ep, indexExpr.left) {
		return false
	}
	rtgAsmPopRcx(a)
	if elemSize != 1 {
		rtgAsmImulRcxImm(a, elemSize)
	}
	rtgAsmAddRaxRcx(a)
	return true
}

func rtgEmitIndexExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	left := &ep.exprs[e.left]
	if left.kind == rtgExprString {
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		msg := rtgDecodeStringToken(g.prog, left.tok)
		msgOff := rtgAddStringData(g, msg)
		rtgAsmPushRax(a)
		rtgAsmMovRaxDataAddr(a, msgOff)
		rtgAsmPopRcx(a)
		rtgAsmLoadByteRaxIndexRcx(a)
		return true
	}
	if left.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, left.nameStart, left.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, left.nameStart, left.nameEnd)
			globalType := rtgFindGlobalType(g, left.nameStart, left.nameEnd)
			if globalOffset >= 0 && rtgTypeIsString(meta, globalType) {
				if !rtgEmitIntExpr(g, ep, e.right) {
					return false
				}
				rtgAsmPushRax(a)
				rtgAsmLoadRaxBss(a, globalOffset)
				rtgAsmPopRcx(a)
				rtgAsmLoadByteRaxIndexRcx(a)
				return true
			}
			if globalOffset >= 0 && rtgTypeIsSlice(meta, globalType) {
				t := rtgResolveType(meta, globalType)
				elem := rtgResolveType(meta, t.elem)
				if elem.kind != rtgTypeInt && elem.kind != rtgTypeByte && elem.kind != rtgTypeBool {
					return false
				}
				if !rtgEmitIntExpr(g, ep, e.right) {
					return false
				}
				rtgAsmPushRax(a)
				rtgAsmLoadRaxBss(a, globalOffset)
				rtgAsmPopRcx(a)
				if elem.kind == rtgTypeByte || elem.kind == rtgTypeBool {
					rtgAsmLoadByteRaxIndexRcx(a)
				} else {
					rtgAsmLoadQwordRaxIndexRcx8(a)
				}
				return true
			}
			constTok := rtgFindConstStringToken(g, left.nameStart, left.nameEnd)
			if constTok >= 0 {
				if !rtgEmitIntExpr(g, ep, e.right) {
					return false
				}
				msg := rtgDecodeStringToken(g.prog, constTok)
				msgOff := rtgAddStringData(g, msg)
				rtgAsmPushRax(a)
				rtgAsmMovRaxDataAddr(a, msgOff)
				rtgAsmPopRcx(a)
				rtgAsmLoadByteRaxIndexRcx(a)
				return true
			}
			return false
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind == rtgTypeString {
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushRax(a)
			rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
			rtgAsmPopRcx(a)
			rtgAsmLoadByteRaxIndexRcx(a)
			return true
		}
		if t.kind == rtgTypeSlice {
			elem := rtgResolveType(meta, t.elem)
			if elem.kind != rtgTypeInt && elem.kind != rtgTypeByte && elem.kind != rtgTypeBool {
				return false
			}
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushRax(a)
			rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
			rtgAsmPopRcx(a)
			if elem.kind == rtgTypeByte || elem.kind == rtgTypeBool {
				rtgAsmLoadByteRaxIndexRcx(a)
			} else {
				rtgAsmLoadQwordRaxIndexRcx8(a)
			}
			return true
		}
	}
	if left.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, e.left)
		t := rtgResolveType(meta, fieldType)
		if t.kind == rtgTypeString {
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushRax(a)
			if !rtgEmitSelectorAddressRdx(g, ep, e.left) {
				return false
			}
			rtgAsmLoadRaxMemRdxDisp(a, 0)
			rtgAsmPopRcx(a)
			rtgAsmLoadByteRaxIndexRcx(a)
			return true
		}
		if t.kind == rtgTypeSlice {
			elem := rtgResolveType(meta, t.elem)
			if elem.kind != rtgTypeInt && elem.kind != rtgTypeByte && elem.kind != rtgTypeBool {
				return false
			}
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushRax(a)
			if !rtgEmitSelectorAddressRdx(g, ep, e.left) {
				return false
			}
			rtgAsmLoadRaxMemRdxDisp(a, 0)
			rtgAsmPopRcx(a)
			if elem.kind == rtgTypeByte || elem.kind == rtgTypeBool {
				rtgAsmLoadByteRaxIndexRcx(a)
			} else {
				rtgAsmLoadQwordRaxIndexRcx8(a)
			}
			return true
		}
	}
	if left.kind == rtgExprUnary && rtgTokCharIs(p, left.tok, '*') {
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushRax(a)
		if !rtgEmitIntExpr(g, ep, left.left) {
			return false
		}
		rtgAsmMovRdxRax(a)
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		rtgAsmPopRcx(a)
		rtgAsmLoadByteRaxIndexRcx(a)
		return true
	}
	if left.kind == rtgExprIndex {
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushRax(a)
		if !rtgEmitStringPtrExpr(g, ep, e.left) {
			return false
		}
		rtgAsmPopRcx(a)
		rtgAsmLoadByteRaxIndexRcx(a)
		return true
	}
	return false
}

func rtgEmitStringPtrExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	p := g.prog
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprString {
		msg := rtgDecodeStringToken(p, e.tok)
		msgOff := rtgAddStringData(g, msg)
		rtgAsmMovRaxDataAddr(a, msgOff)
		return true
	}
	if e.kind == rtgExprCall && e.argCount == 1 && rtgExprIsIdentText(p, ep, e.left, "string") {
		argIndex := ep.args[e.firstArg]
		arg := &ep.exprs[argIndex]
		if arg.kind != rtgExprIdent {
			return false
		}
		localIndex := rtgFindLocalIndex(g, arg.nameStart, arg.nameEnd)
		if localIndex < 0 {
			return false
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind != rtgTypeSlice {
			return false
		}
		elem := rtgResolveType(meta, t.elem)
		if elem.kind != rtgTypeByte {
			return false
		}
		rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
		return true
	}
	if e.kind == rtgExprCall {
		callType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsString(meta, callType) {
			return false
		}
		return rtgEmitStringValueRegs(g, ep, idx)
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if !rtgTypeIsString(meta, g.locals[localIndex].typ) {
				return false
			}
			rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
		globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 && rtgTypeIsString(meta, globalType) {
			rtgAsmLoadRaxBss(a, globalOffset)
			return true
		}
		constTok := rtgFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constTok >= 0 {
			msg := rtgDecodeStringToken(p, constTok)
			msgOff := rtgAddStringData(g, msg)
			rtgAsmMovRaxDataAddr(a, msgOff)
			return true
		}
		return false
	}
	if e.kind == rtgExprIndex {
		left := &ep.exprs[e.left]
		if left.kind != rtgExprIdent {
			return false
		}
		localIndex := rtgFindLocalIndex(g, left.nameStart, left.nameEnd)
		if localIndex < 0 {
			return false
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind != rtgTypeSlice {
			return false
		}
		elem := rtgResolveType(meta, t.elem)
		if elem.kind != rtgTypeString {
			return false
		}
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushRax(a)
		rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
		rtgAsmPopRcx(a)
		rtgAsmShlRcxImm(a, 4)
		rtgAsmEmit32(a, 134515528)
		return true
	}
	return false
}

func rtgFindLocalOffset(g *rtgLinearGen, nameStart int, nameEnd int) int {
	localIndex := rtgFindLocalIndex(g, nameStart, nameEnd)
	if localIndex < 0 {
		return -1
	}
	return g.locals[localIndex].offset
}

func rtgFindLocalIndex(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := len(g.locals) - 1; i >= 0; i-- {
		if rtgBytesEqualRange(g.prog.src, g.locals[i].nameStart, g.locals[i].nameEnd, nameStart, nameEnd) {
			return i
		}
	}
	return -1
}

func rtgEmitSelectorAddressRdx(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	base := &ep.exprs[e.left]
	baseType := rtgInferParsedExprType(g, ep, e.left)
	fieldOffset := rtgStructFieldOffset(g, baseType, e.nameStart, e.nameEnd)
	if fieldOffset < 0 {
		return false
	}
	if base.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, base.left)
		sliceType := rtgResolveType(meta, leftType)
		elemType := rtgResolveType(meta, sliceType.elem)
		if sliceType.kind != rtgTypeSlice || elemType.kind != rtgTypeStruct {
			return false
		}
		if !rtgEmitIntExpr(g, ep, base.right) {
			return false
		}
		rtgAsmPushRax(a)
		if !rtgEmitSlicePtrLen(g, ep, base.left) {
			return false
		}
		rtgAsmPopRcx(a)
		elemSize := rtgTypeSize(meta, sliceType.elem)
		rtgAsmImulRcxImm(a, elemSize)
		rtgAsmMovRdxRax(a)
		rtgAsmAddRdxRcx(a)
		if fieldOffset != 0 {
			rtgAsmAddRdxImm(a, fieldOffset)
		}
		return true
	}
	if base.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, base.nameStart, base.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, base.nameStart, base.nameEnd)
			globalType := rtgFindGlobalType(g, base.nameStart, base.nameEnd)
			t := rtgResolveType(meta, globalType)
			if globalOffset < 0 {
				return false
			}
			if t.kind == rtgTypePointer {
				rtgAsmLoadRaxBss(a, globalOffset)
				rtgAsmMovRdxRax(a)
				if fieldOffset != 0 {
					rtgAsmAddRdxImm(a, fieldOffset)
				}
				return true
			}
			if t.kind != rtgTypeStruct {
				return false
			}
			rtgAsmMovRaxBssAddr(a, globalOffset)
			rtgAsmMovRdxRax(a)
			if fieldOffset != 0 {
				rtgAsmAddRdxImm(a, fieldOffset)
			}
			return true
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind == rtgTypePointer {
			rtgAsmLoadRdxStack(a, g.locals[localIndex].offset)
			if fieldOffset != 0 {
				rtgAsmAddRdxImm(a, fieldOffset)
			}
			return true
		}
		rtgAsmStackMem(a, g.locals[localIndex].offset-fieldOffset, 36168, 0x55, 0x95)
		return true
	}
	if base.kind == rtgExprSelector {
		if !rtgEmitSelectorAddressRdx(g, ep, e.left) {
			return false
		}
		t := rtgResolveType(meta, baseType)
		if t.kind == rtgTypePointer {
			rtgAsmEmit3(a, 0x48, 0x8b, 0x12)
		}
		if fieldOffset != 0 {
			rtgAsmAddRdxImm(a, fieldOffset)
		}
		return true
	}
	return false
}

func rtgStructFieldIndex(g *rtgLinearGen, typ int, nameStart int, nameEnd int) int {
	meta := g.meta
	if typ < 0 || typ >= len(meta.types) {
		return -1
	}
	t := rtgResolveType(meta, typ)
	if t.kind == rtgTypePointer && t.elem > 0 && t.elem < len(meta.types) {
		t = rtgResolveType(meta, t.elem)
	}
	if t.kind != rtgTypeStruct {
		return -1
	}
	for i := 0; i < t.count; i++ {
		field := &meta.fields[t.first+i]
		if rtgBytesEqualRange(g.prog.src, field.nameStart, field.nameEnd, nameStart, nameEnd) {
			return t.first + i
		}
	}
	return -1
}

func rtgStructFieldOffset(g *rtgLinearGen, typ int, nameStart int, nameEnd int) int {
	fieldIndex := rtgStructFieldIndex(g, typ, nameStart, nameEnd)
	if fieldIndex < 0 {
		return -1
	}
	return g.meta.fields[fieldIndex].offset
}

func rtgStructFieldType(g *rtgLinearGen, typ int, nameStart int, nameEnd int) int {
	fieldIndex := rtgStructFieldIndex(g, typ, nameStart, nameEnd)
	if fieldIndex < 0 {
		return 0
	}
	return g.meta.fields[fieldIndex].typ
}

func rtgCompositeStructFieldIndex(g *rtgLinearGen, typ int, field *rtgCompositeField, pos int) int {
	if field.nameEnd > field.nameStart {
		return rtgStructFieldIndex(g, typ, field.nameStart, field.nameEnd)
	}
	t := rtgResolveType(g.meta, typ)
	if t.kind != rtgTypeStruct || pos < 0 || pos >= t.count {
		return -1
	}
	return t.first + pos
}

func rtgFindGlobalOffset(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.globals); i++ {
		if rtgBytesEqualRange(g.prog.src, g.globals[i].nameStart, g.globals[i].nameEnd, nameStart, nameEnd) {
			return g.globals[i].offset
		}
	}
	return -1
}

func rtgFindGlobalType(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.meta.globals); i++ {
		s := &g.meta.globals[i]
		if s.kind == rtgTokVar && rtgBytesEqualRange(g.prog.src, s.nameStart, s.nameEnd, nameStart, nameEnd) {
			return s.typ
		}
	}
	return 0
}

func rtgFindConstStringToken(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.meta.globals); i++ {
		s := &g.meta.globals[i]
		if s.kind == rtgTokConst && rtgBytesEqualRange(g.prog.src, s.nameStart, s.nameEnd, nameStart, nameEnd) {
			if s.initStart+1 == s.initEnd && rtgTokIsKind(g.prog, s.initStart, rtgTokString) {
				return s.initStart
			}
		}
	}
	return -1
}

func rtgFindSmallConstByName(g *rtgLinearGen, nameStart int, nameEnd int) int {
	if rtgFindLocalIndex(g, nameStart, nameEnd) >= 0 {
		return -129
	}
	if rtgBytesEqualText(g.prog.src, nameStart, nameEnd, "nil") {
		return 0
	}
	for i := 0; i < len(g.meta.globals); i++ {
		s := &g.meta.globals[i]
		if s.kind == rtgTokConst && rtgBytesEqualRange(g.prog.src, s.nameStart, s.nameEnd, nameStart, nameEnd) {
			if s.initStart+1 != s.initEnd {
				return -129
			}
			if rtgTokIsKind(g.prog, s.initStart, rtgTokNumber) {
				value := rtgParseIntToken(g.prog, s.initStart)
				if rtgAsmImmFits8Signed(value) {
					return value
				}
			}
			if rtgTokIsKind(g.prog, s.initStart, rtgTokChar) {
				value := rtgParseCharToken(g.prog, s.initStart)
				if rtgAsmImmFits8Signed(value) {
					return value
				}
			}
			return -129
		}
	}
	return -129
}

func rtgAddTypedLocal(g *rtgLinearGen, nameStart int, nameEnd int, typ int) int {
	size := rtgTypeSize(g.meta, typ)
	if size < 8 {
		size = 8
	}
	g.stackUsed = rtgAlignTo8(g.stackUsed + size)
	offset := g.stackUsed
	g.locals = append(g.locals, rtgLocalInfo{nameStart: nameStart, nameEnd: nameEnd, offset: offset, typ: typ, size: size})
	return offset
}

func rtgZeroLocalAtOffset(g *rtgLinearGen, offset int) {
	a := &g.asm
	size := 8
	typ := rtgTypeInt
	for i := 0; i < len(g.locals); i++ {
		if g.locals[i].offset == offset {
			size = g.locals[i].size
			typ = g.locals[i].typ
		}
	}
	t := rtgResolveType(g.meta, typ)
	if t.kind == rtgTypeSlice {
		elemSize := rtgTypeSize(g.meta, t.elem)
		if elemSize < 1 {
			elemSize = 8
		}
		backingSize := 8388608
		backingOff := g.asm.bssSize
		g.asm.bssSize += backingSize
		rtgAsmMovRaxBssAddr(a, backingOff)
		rtgAsmStoreRaxStack(a, offset)
		rtgAsmMovRaxImm(a, 0)
		rtgAsmStoreRaxStack(a, offset-8)
		rtgAsmMovRaxImm(a, backingSize/elemSize)
		rtgAsmStoreRaxStack(a, offset-16)
		return
	}
	rtgAsmMovRaxImm(a, 0)
	for at := 0; at < size; at += 8 {
		rtgAsmStoreRaxStack(a, offset-at)
	}
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

func rtgEvalBuiltinConst(g *rtgLinearGen, nameStart int, nameEnd int) rtgConstResult {
	p := g.prog
	if rtgBytesEqualText(p.src, nameStart, nameEnd, "iota") {
		if g.constEvalIotaValid != 0 {
			return rtgConstResultOk(g.constEvalIota)
		}
	}
	if rtgBytesEqualText(p.src, nameStart, nameEnd, "nil") {
		return rtgConstResultOk(0)
	}
	if rtgBytesEqualText(p.src, nameStart, nameEnd, "O_RDONLY") {
		return rtgConstResultOk(0)
	}
	if rtgBytesEqualText(p.src, nameStart, nameEnd, "O_WRONLY") {
		return rtgConstResultOk(1)
	}
	if rtgBytesEqualText(p.src, nameStart, nameEnd, "O_RDWR") {
		return rtgConstResultOk(2)
	}
	if rtgBytesEqualText(p.src, nameStart, nameEnd, "O_CREATE") {
		return rtgConstResultOk(64)
	}
	if rtgBytesEqualText(p.src, nameStart, nameEnd, "O_TRUNC") {
		return rtgConstResultOk(512)
	}
	var r rtgConstResult
	return r
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
		if e.argCount == 1 && (callee == rtgIdentInt || callee == rtgIdentByte || callee == rtgIdentInt64) {
			result := rtgEvalConstExpr(g, ep, ep.args[e.firstArg])
			return result
		}
		if e.argCount == 1 {
			calleeExpr := &ep.exprs[e.left]
			if calleeExpr.kind == rtgExprIdent {
				namedType := rtgFindTypeByRange(g, calleeExpr.nameStart, calleeExpr.nameEnd)
				resolved := rtgResolveType(g.meta, namedType)
				if resolved.kind == rtgTypeInt || resolved.kind == rtgTypeInt64 || resolved.kind == rtgTypeBool {
					return rtgEvalConstExpr(g, ep, ep.args[e.firstArg])
				}
				if resolved.kind == rtgTypeByte {
					result := rtgEvalConstExpr(g, ep, ep.args[e.firstArg])
					if result.ok {
						result.value = result.value & 255
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

func rtgFuncInfoFromCall(g *rtgLinearGen, ep *rtgExprParse, idx int) int {
	e := &ep.exprs[idx]
	nameStart := e.nameStart
	nameEnd := e.nameEnd
	wantMethod := false
	if e.kind == rtgExprSelector {
		wantMethod = true
	} else if e.kind != rtgExprIdent {
		return -1
	}
	for i := 0; i < len(g.meta.funcs); i++ {
		f := &g.meta.funcs[i]
		isMethod := f.receiverType != 0
		if isMethod == wantMethod && rtgBytesEqualRange(g.prog.src, f.nameStart, f.nameEnd, nameStart, nameEnd) {
			return i
		}
	}
	return -1
}

func rtgAsmInit(a *rtgAsm) {
	var code []byte
	var labelPos []int
	var labelSet []bool
	var relocs []rtgLabelRef
	var dataRelocs []rtgDataRef
	var absRelocs []rtgAbsRef
	var data []byte
	a.code = code
	a.labelPos = labelPos
	a.labelSet = labelSet
	a.relocs = relocs
	a.dataRelocs = dataRelocs
	a.absRelocs = absRelocs
	a.data = data
	a.bssSize = 0
	a.codeOffset = 120
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

func rtgAsmAddReloc(a *rtgAsm, at int, label int, kind int) {
	a.relocs = append(a.relocs, rtgLabelRef{at: at, label: label, kind: kind})
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

func rtgAsmImmFits8Signed(imm int) bool {
	return imm >= -128 && imm <= 127
}

func rtgAsmMovRaxImm(a *rtgAsm, imm int) {
	if imm == 0 {
		rtgAsmEmit16(a, 49201)
		return
	}
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit2(a, 0x6a, imm)
		rtgAsmPopRax(a)
		return
	}
	rtgAsmEmit16(a, 0xb848)
	rtgAsmEmit64(a, imm)
}

func rtgAsmMovRaxIntToken(a *rtgAsm, p *rtgProgram, tokIndex int) {
	value := rtgParseIntToken(p, tokIndex)
	if rtgIntTokenNeedsMovabs(p, tokIndex) {
		rtgAsmMovRaxImm64(a, value)
		return
	}
	rtgAsmMovRaxImm(a, value)
}

func rtgAsmMovRaxImm64(a *rtgAsm, imm int) {
	rtgAsmEmit16(a, 0xb848)
	rtgAsmEmit64(a, imm)
}

func rtgIntTokenNeedsMovabs(p *rtgProgram, tokIndex int) bool {
	tok := &p.toks[tokIndex]
	start := tok.start
	if tok.end-start > 2 && p.src[start] == '0' {
		return false
	}
	digits := tok.end - start
	if digits > 10 {
		return true
	}
	if digits < 10 {
		return false
	}
	limit := "2147483647"
	for i := 0; i < 10; i++ {
		c := p.src[start+i]
		if c > limit[i] {
			return true
		}
		if c < limit[i] {
			return false
		}
	}
	return false
}

func rtgAsmMovRdxImm(a *rtgAsm, imm int) {
	if imm == 0 {
		rtgAsmEmit16(a, 53809)
		return
	}
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit2(a, 0x6a, imm)
		rtgAsmPopRdx(a)
		return
	}
	if imm >= 0 {
		if imm <= 2147483647 {
			rtgAsmEmit8(a, 0xba)
			rtgAsmEmit32(a, imm)
			return
		}
	} else {
		if imm >= -2147483647 {
			rtgAsmEmit24(a, 12765000)
			rtgAsmEmit32(a, imm)
			return
		}
	}
	rtgAsmEmit16(a, 0xba48)
	rtgAsmEmit64(a, imm)
}

func rtgAsmMovRaxDataAddr(a *rtgAsm, dataOff int) {
	rtgAsmEmit24(a, 363848)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, dataOff, 3)
}

func rtgAsmMovRaxBssAddr(a *rtgAsm, bssOff int) {
	rtgAsmEmit24(a, 363848)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, 4)
}

func rtgAsmMovR10BssAddr(a *rtgAsm, bssOff int) {
	rtgAsmEmit24(a, 1412428)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, 4)
}

func rtgAsmBuildArgvEnvSlices(a *rtgAsm, bssOff int, envOff int, envLenOff int) {
	loopLabel := rtgAsmNewLabel(a)
	strlenLabel := rtgAsmNewLabel(a)
	afterLenLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	envScanLabel := rtgAsmNewLabel(a)
	envStartLabel := rtgAsmNewLabel(a)
	envLoopLabel := rtgAsmNewLabel(a)
	envStrlenLabel := rtgAsmNewLabel(a)
	envAfterLenLabel := rtgAsmNewLabel(a)
	envDoneLabel := rtgAsmNewLabel(a)
	rtgAsmEmit32(a, 604277576)
	rtgAsmEmit24(a, 12618057)
	rtgAsmEmit32(a, 608996684)
	rtgAsmEmit8(a, 0x08)
	rtgAsmMovR10BssAddr(a, bssOff)
	rtgAsmEmit32(a, 1305774413)
	rtgAsmEmit16(a, 56113)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmEmit24(a, 12794189)
	rtgAsmEmit16(a, 36111)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, doneLabel, rtgRel32)
	rtgAsmEmit32(a, -650343605)
	rtgAsmEmit32(a, 1211795785)
	rtgAsmEmit16(a, 49201)
	rtgAsmMarkLabel(a, strlenLabel)
	rtgAsmEmit32(a, 474240)
	rtgAsmJzLabel(a, afterLenLabel)
	rtgAsmEmit24(a, 12648264)
	rtgAsmJmpLabel(a, strlenLabel)
	rtgAsmMarkLabel(a, afterLenLabel)
	rtgAsmEmit32(a, 138578249)
	rtgAsmEmit32(a, 281183049)
	rtgAsmEmit24(a, 12844873)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)

	rtgAsmEmit32(a, 608996684)
	rtgAsmEmit8(a, 0x08)
	rtgAsmMarkLabel(a, envScanLabel)
	rtgAsmEmit32(a, 3769161)
	rtgAsmJzLabel(a, envStartLabel)
	rtgAsmEmit32(a, 146899785)
	rtgAsmJmpLabel(a, envScanLabel)
	rtgAsmMarkLabel(a, envStartLabel)
	rtgAsmEmit32(a, 146899785)
	rtgAsmMovR10BssAddr(a, envOff)
	rtgAsmEmit24(a, 14365005)
	rtgAsmMarkLabel(a, envLoopLabel)
	rtgAsmEmit32(a, -650343605)
	rtgAsmEmit32(a, 16745288)
	rtgAsmJzLabel(a, envDoneLabel)
	rtgAsmEmit32(a, 1211795785)
	rtgAsmEmit16(a, 49201)
	rtgAsmMarkLabel(a, envStrlenLabel)
	rtgAsmEmit32(a, 474240)
	rtgAsmJzLabel(a, envAfterLenLabel)
	rtgAsmEmit24(a, 12648264)
	rtgAsmJmpLabel(a, envStrlenLabel)
	rtgAsmMarkLabel(a, envAfterLenLabel)
	rtgAsmEmit32(a, 138578249)
	rtgAsmEmit32(a, 281183049)
	rtgAsmEmit24(a, 12844873)
	rtgAsmJmpLabel(a, envLoopLabel)
	rtgAsmMarkLabel(a, envDoneLabel)
	rtgAsmEmit24(a, 14190924)
	rtgAsmStoreRaxBss(a, envLenOff)

	rtgAsmEmit32(a, 1290242380)
	rtgAsmEmit32(a, -1991457143)
	rtgAsmEmit8(a, 0xc2)
}

func rtgAsmLoadRaxBss(a *rtgAsm, bssOff int) {
	rtgAsmEmit24(a, 363336)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, 4)
}

func rtgAsmStoreRaxBss(a *rtgAsm, bssOff int) {
	rtgAsmEmit24(a, 362824)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, 4)
}

func rtgAsmMovRdiRax(a *rtgAsm) {
	rtgAsmEmit16(a, 24400)
}

func rtgAsmMovRdxRax(a *rtgAsm) {
	rtgAsmEmit16(a, 23120)
}

func rtgAsmMovRaxRdx(a *rtgAsm) {
	rtgAsmEmit3(a, 0x48, 0x89, 0xd0)
}

func rtgAsmMovRsiRax(a *rtgAsm) {
	rtgAsmEmit16(a, 24144)
}

func rtgAsmMovRcxRax(a *rtgAsm) {
	rtgAsmEmit16(a, 22864)
}

func rtgAsmMovRcxRdx(a *rtgAsm) {
	rtgAsmEmit16(a, 22866)
}

func rtgAsmAddRdxRcx(a *rtgAsm) {
	rtgAsmEmit24(a, 13238600)
}

func rtgAsmSyscall(a *rtgAsm) {
	rtgAsmEmit16(a, 1295)
}

func rtgAsmPushRax(a *rtgAsm) {
	rtgAsmEmit8(a, 0x50)
}

func rtgAsmPushRcx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x51)
}

func rtgAsmPushRdx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x52)
}

func rtgAsmPushImm(a *rtgAsm, imm int) {
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit2(a, 0x6a, imm)
		return
	}
	if imm >= -2147483647 && imm <= 2147483647 {
		rtgAsmEmit8(a, 0x68)
		rtgAsmEmit32(a, imm)
		return
	}
	rtgAsmMovRaxImm(a, imm)
	rtgAsmPushRax(a)
}

func rtgAsmPushSliceRegs(a *rtgAsm) {
	rtgAsmPushRcx(a)
	rtgAsmPushRdx(a)
	rtgAsmPushRax(a)
}

func rtgAsmPushStringRegs(a *rtgAsm) {
	rtgAsmPushRdx(a)
	rtgAsmPushRax(a)
}

func rtgAsmPopRax(a *rtgAsm) {
	rtgAsmEmit8(a, 0x58)
}

func rtgAsmPopRcx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x59)
}

func rtgAsmPopRdx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x5a)
}

func rtgAsmPopRdi(a *rtgAsm) {
	rtgAsmEmit8(a, 0x5f)
}

func rtgAsmStackMem(a *rtgAsm, offset int, base int, disp8 int, disp32 int) {
	rtgAsmEmit16(a, base)
	if offset >= 0 && offset <= 128 {
		rtgAsmEmit8(a, disp8)
		rtgAsmEmit8(a, -offset)
		return
	}
	rtgAsmEmit8(a, disp32)
	rtgAsmEmit32(a, -offset)
}

func rtgAsmStoreRaxStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 35144, 0x45, 0x85)
}

func rtgAsmStoreRdxStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 35144, 0x55, 0x95)
}

func rtgAsmLoadRaxStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 35656, 0x45, 0x85)
}

func rtgAsmLeaRaxStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 36168, 0x45, 0x85)
}

func rtgAsmLeaRdiStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 36168, 0x7d, 0xbd)
}

func rtgAsmLeaRsiStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 36168, 0x75, 0xb5)
}

func rtgAsmLoadRdxStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 35656, 0x55, 0x95)
}

func rtgAsmAddRdxImm(a *rtgAsm, imm int) {
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit4(a, 0x48, 0x83, 0xc2, imm)
		return
	}
	rtgAsmEmit24(a, 12747080)
	rtgAsmEmit32(a, imm)
}

func rtgAsmMemDisp(a *rtgAsm, disp int, op int, disp8 int, disp32 int) {
	rtgAsmEmit16(a, op)
	if rtgAsmImmFits8Signed(disp) {
		rtgAsmEmit2(a, disp8, disp)
		return
	}
	rtgAsmEmit8(a, disp32)
	rtgAsmEmit32(a, disp)
}

func rtgAsmLoadRcxStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 35656, 0x4d, 0x8d)
}

func rtgAsmStoreSliceStack(a *rtgAsm, offset int) {
	rtgAsmStoreRaxStack(a, offset)
	rtgAsmStoreRdxStack(a, offset-8)
	rtgAsmStackMem(a, offset-16, 35144, 0x4d, 0x8d)
}

func rtgAsmPopStoreStringMemRdx(a *rtgAsm, disp int) {
	rtgAsmPopRax(a)
	rtgAsmStoreRaxMemRdxDisp(a, disp)
	rtgAsmPopRax(a)
	rtgAsmStoreRaxMemRdxDisp(a, disp+8)
}

func rtgAsmPopStoreSliceMemRdx(a *rtgAsm, disp int) {
	rtgAsmPopRax(a)
	rtgAsmStoreRaxMemRdxDisp(a, disp)
	rtgAsmPopRax(a)
	rtgAsmStoreRaxMemRdxDisp(a, disp+8)
	rtgAsmPopRax(a)
	rtgAsmStoreRaxMemRdxDisp(a, disp+16)
}

func rtgAsmLoadQwordRaxIndexRcx8(a *rtgAsm) {
	rtgAsmEmit32(a, -939226296)
}

func rtgAsmLoadQwordRaxIndexRcxDisp(a *rtgAsm, disp int) {
	rtgAsmEmit16(a, 35656)
	if rtgAsmImmFits8Signed(disp) {
		rtgAsmEmit3(a, 0x44, 0x08, disp)
		return
	}
	rtgAsmEmit16(a, 2180)
	rtgAsmEmit32(a, disp)
}

func rtgAsmLoadRaxMemRdxDisp(a *rtgAsm, disp int) {
	if disp == 0 {
		rtgAsmEmit3(a, 0x48, 0x8b, 0x02)
		return
	}
	rtgAsmMemDisp(a, disp, 35656, 0x42, 0x82)
}

func rtgAsmLoadByteRaxIndexRcx(a *rtgAsm) {
	rtgAsmEmit32(a, 79040328)
	rtgAsmEmit8(a, 0x08)
}

func rtgAsmStoreRaxMemRdxRcx8(a *rtgAsm) {
	rtgAsmEmit32(a, -905672376)
}

func rtgAsmStoreRaxMemRdxDisp(a *rtgAsm, disp int) {
	if disp == 0 {
		rtgAsmEmit3(a, 0x48, 0x89, 0x02)
		return
	}
	rtgAsmMemDisp(a, disp, 35144, 0x42, 0x82)
}

func rtgAsmStoreAlMemRdxRcx1(a *rtgAsm) {
	rtgAsmEmit24(a, 656520)
}

func rtgAsmIncMemRdx(a *rtgAsm) {
	rtgAsmEmit24(a, 196424)
}

func rtgAsmDecMemRdx(a *rtgAsm) {
	rtgAsmEmit24(a, 720712)
}

func rtgAsmIncRcx(a *rtgAsm) {
	rtgAsmEmit16(a, 49663)
}

func rtgAsmIncRax(a *rtgAsm) {
	rtgAsmEmit16(a, 49407)
}

func rtgAsmBoolNotRax(a *rtgAsm) {
	rtgAsmEmit24(a, 12617032)
	rtgAsmEmit24(a, 12620815)
	rtgAsmEmit24(a, 12629519)
}

func rtgAsmCmpRaxImm8(a *rtgAsm, imm int) {
	if imm == 0 {
		rtgAsmEmit16(a, 49285)
		return
	}
	rtgAsmEmit4(a, 0x48, 0x83, 0xf8, imm)
}

func rtgAsmAddRaxRcx(a *rtgAsm) {
	rtgAsmEmit24(a, 13107528)
}

func rtgAsmShlRcxImm(a *rtgAsm, imm int) {
	rtgAsmEmit4(a, 0x48, 0xc1, 0xe1, imm)
}

func rtgAsmShlRaxImm(a *rtgAsm, imm int) {
	rtgAsmEmit4(a, 0x48, 0xc1, 0xe0, imm)
}

func rtgAsmSarRaxImm(a *rtgAsm, imm int) {
	rtgAsmEmit4(a, 0x48, 0xc1, 0xf8, imm)
}

func rtgAsmImulRcxImm(a *rtgAsm, imm int) {
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit3(a, 0x6b, 0xc9, imm)
		return
	}
	rtgAsmEmit16(a, 51561)
	rtgAsmEmit32(a, imm)
}

func rtgAsmDivLeftRcxRightRax(a *rtgAsm, mod bool) {
	rtgAsmEmit32(a, 1220774216)
	rtgAsmEmit32(a, -1723283319)
	rtgAsmEmit24(a, 16512840)
	if mod {
		rtgAsmEmit24(a, 13666632)
	}
}

func rtgAsmCmpRcxRaxSet(a *rtgAsm, setcc int) {
	rtgAsmEmit32(a, 264321352)
	rtgAsmEmit3(a, setcc, 0xc0, 0x0f)
	rtgAsmEmit16(a, 49334)
}

func rtgAsmRet(a *rtgAsm) {
	rtgAsmEmit8(a, 0xc3)
}

func rtgAsmLeave(a *rtgAsm) {
	rtgAsmEmit8(a, 0xc9)
}

func rtgAsmCallLabel(a *rtgAsm, label int) {
	rtgAsmEmit8(a, 0xe8)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label, rtgRel32)
}

func rtgAsmJmpLabel(a *rtgAsm, label int) {
	rtgAsmEmit8(a, 0xe9)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label, rtgRel32)
}

func rtgAsmJzLabel(a *rtgAsm, label int) {
	rtgAsmEmit16(a, 33807)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label, rtgRel32)
}

func rtgAsmJnzLabel(a *rtgAsm, label int) {
	rtgAsmEmit16(a, 34063)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label, rtgRel32)
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
	for i := 0; i < len(a.dataRelocs); i++ {
		r := a.dataRelocs[i]
		target := a.dataOffset + r.off
		next := a.codeOffset + r.at + 4
		disp := target - next
		rtgPut32At(a.code, r.at, disp)
	}
	for i := 0; i < len(a.absRelocs); i++ {
		r := a.absRelocs[i]
		target := a.dataOffset + r.off
		if r.kind == 4 {
			target = a.dataOffset + len(a.data) + r.off
		}
		next := a.codeOffset + r.at + 4
		disp := target - next
		rtgPut32At(a.code, r.at, disp)
	}
}

func rtgAsmImage(a *rtgAsm) []byte {
	rtgAsmPatch(a)
	fileSize := a.codeOffset + len(a.code) + len(a.data)
	memSize := fileSize + a.bssSize
	var out []byte
	out = rtgAppendElfHeader(out, a.codeOffset, fileSize, memSize)
	for i := 0; i < len(a.code); i++ {
		out = append(out, a.code[i])
	}
	for i := 0; i < len(a.data); i++ {
		out = append(out, a.data[i])
	}
	return out
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

func rtgAppendElfHeader(out []byte, entryOff int, fileSize int, memSize int) []byte {
	base := 0x400000

	out = append(out, 0x7f)
	out = append(out, 'E')
	out = append(out, 'L')
	out = append(out, 'F')
	out = append(out, 2)
	out = append(out, 1)
	out = append(out, 1)
	out = append(out, 0)
	for i := 0; i < 8; i++ {
		out = append(out, 0)
	}
	out = rtgAppend16(out, 2)
	out = rtgAppend16(out, 0x3e)
	out = rtgAppend32(out, 1)
	out = rtgAppend64(out, base+entryOff)
	out = rtgAppend64(out, 64)
	out = rtgAppend64(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend16(out, 64)
	out = rtgAppend16(out, 56)
	out = rtgAppend16(out, 1)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 0)

	out = rtgAppend32(out, 1)
	out = rtgAppend32(out, 7)
	out = rtgAppend64(out, 0)
	out = rtgAppend64(out, base)
	out = rtgAppend64(out, base)
	out = rtgAppend64(out, fileSize)
	out = rtgAppend64(out, memSize)
	out = rtgAppend64(out, 0x1000)
	return out
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

func rtgParseExpression(p *rtgProgram, start int, end int) rtgExprParse {
	var ep rtgExprParse
	ep.prog = p
	ep.pos = start
	ep.end = end
	ep.ok = true
	rtgParseBinaryExpr(&ep, 1)
	if ep.pos < ep.end {
		ep.ok = false
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
		ep.ok = false
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
				ep.ok = false
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
				ep.ok = false
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
				ep.ok = false
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
				ep.ok = false
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
		ep.ok = false
		return 0
	}
	var compositeFields []rtgCompositeField
	ep.pos++
	for ep.ok && ep.pos < ep.end && !rtgTokCharIs(ep.prog, ep.pos, '}') {
		if !rtgTokIsKind(ep.prog, ep.pos, rtgTokIdent) || !rtgTokCharIs(ep.prog, ep.pos+1, ':') {
			ep.ok = false
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
		ep.ok = false
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
		ep.ok = false
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
			ep.ok = false
			return inner
		}
		ep.pos++
		return inner
	}
	ep.ok = false
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
	rtgAddBuiltinType(m, rtgTypeInt, 8)
	rtgAddBuiltinType(m, rtgTypeInt64, 8)
	rtgAddBuiltinType(m, rtgTypeByte, 1)
	rtgAddBuiltinType(m, rtgTypeBool, 1)
	rtgAddBuiltinType(m, rtgTypeString, 16)
	rtgAddBuiltinType(m, rtgTypeFloat64, 8)
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
			m.ok = false
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
			m.ok = false
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
			m.ok = false
			return
		}
		if valueCount != len(names) {
			m.ok = false
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
					result.value = result.value & 255
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
		m.ok = false
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
			m.ok = false
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
		m.ok = false
		return
	}
	lparen := rtgFindNextTokenText(p, nameTok+1, fn.bodyStart, '(')
	if lparen <= nameTok {
		m.ok = false
		return
	}
	rparen := rtgFindMatchingExprClose(p, lparen+1, fn.bodyStart, '(', ')')
	if rparen <= lparen {
		m.ok = false
		return
	}
	firstParam := len(m.params)
	paramCount := 0
	receiverType := 0
	if fn.receiverStart < fn.receiverEnd {
		beforeReceiver := len(m.params)
		rtgParseParamList(m, p, fn.receiverStart, fn.receiverEnd, &paramCount)
		if len(m.params) <= beforeReceiver {
			m.ok = false
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
			m.ok = false
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
			m.ok = false
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
			m.ok = false
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

func rtgTypeIsString(m *rtgMeta, typ int) bool {
	t := rtgResolveType(m, typ)
	return t.kind == rtgTypeString
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
