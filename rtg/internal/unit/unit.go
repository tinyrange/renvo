package unit

const (
	Magic   = "RTGU"
	Version = 1
)

const (
	TagUnit     = 1
	TagPackage  = 2
	TagText     = 7
	TagTokens   = 8
	TagDecls    = 9
	TagFuncs    = 10
	TagIndexes  = 11
	TagComps    = 12
	TagAssigns  = 13
	TagReturns  = 14
	TagCalls    = 15
	TagRefs     = 16
	TagSels     = 17
	TagTypes    = 18
	TagTypeRefs = 19
)

const (
	TokenEOF = iota
	TokenIdent
	TokenNumber
	TokenFloat
	TokenString
	TokenChar
	TokenPackage
	TokenConst
	TokenVar
	TokenType
	TokenFunc
	TokenStruct
	TokenReturn
	TokenIf
	TokenElse
	TokenFor
	TokenBreak
	TokenContinue
	TokenGoto
	TokenSwitch
	TokenCase
	TokenDefault
	TokenOp
)

type Token struct {
	Kind  int
	Start int
	Size  int
	Line  int
}

type Decl struct {
	Kind      int
	NameStart int
	NameEnd   int
	StartTok  int
	EndTok    int
}

type Func struct {
	NameStart     int
	NameEnd       int
	StartTok      int
	NameTok       int
	ReceiverStart int
	ReceiverEnd   int
	BodyStart     int
	BodyEnd       int
	EndTok        int
}

const (
	TypeOther = iota
	TypeNamed
	TypeStruct
	TypeInterface
	TypeMap
	TypeSlice
	TypeArray
	TypePointer
	TypeFunc
)

type TypeInfo struct {
	NameStart int
	NameEnd   int
	Kind      int
	Decl      int
	Symbol    int
	Alias     bool
	TypeStart int
	TypeEnd   int
	LenStart  int
	LenEnd    int
	KeyStart  int
	KeyEnd    int
	ElemStart int
	ElemEnd   int
}

const (
	OwnerDecl = iota + 1
	OwnerFunc
)

type ExprSpan struct {
	StartTok int
	EndTok   int
}

type IndexExpr struct {
	OwnerKind  int
	OwnerIndex int
	StartTok   int
	EndTok     int
	BaseStart  int
	BaseEnd    int
	OpenTok    int
	CloseTok   int
	IndexStart int
	IndexEnd   int
}

type CompositeExpr struct {
	OwnerKind  int
	OwnerIndex int
	StartTok   int
	EndTok     int
	TypeStart  int
	TypeEnd    int
	OpenTok    int
	CloseTok   int
	Elems      []ExprSpan
}

const (
	AssignUnknown = iota
	AssignSet
	AssignDefine
	AssignAdd
	AssignSub
	AssignMul
	AssignDiv
	AssignMod
	AssignAnd
	AssignOr
	AssignXor
)

type Assignment struct {
	FuncIndex  int
	Kind       int
	StartTok   int
	EndTok     int
	OpTok      int
	LeftStart  int
	LeftEnd    int
	RightStart int
	RightEnd   int
	Targets    []ExprSpan
	Values     []ExprSpan
}

type Return struct {
	FuncIndex int
	StartTok  int
	EndTok    int
	Values    []ExprSpan
}

const (
	CallUnknown = iota
	CallScope
	CallPackage
	CallImportSelector
	CallBuiltin
)

type Call struct {
	OwnerKind  int
	OwnerIndex int
	Kind       int
	CalleeTok  int
	BaseTok    int
	DotTok     int
	ArgsStart  int
	ArgsEnd    int
	Args       []ExprSpan
}

const (
	RefUnknown = iota
	RefScope
	RefPackage
	RefImport
	RefBuiltin
	RefLabel
)

type NameRef struct {
	OwnerKind  int
	OwnerIndex int
	Kind       int
	Token      int
	Index      int
	Package    int
}

const (
	SelectorUnknown = iota
	SelectorImport
)

type Selector struct {
	OwnerKind   int
	OwnerIndex  int
	Kind        int
	BaseTok     int
	DotTok      int
	NameTok     int
	BaseKind    int
	BaseIndex   int
	BasePackage int
	Package     int
	Symbol      int
}

const (
	TypeRefUnknown = iota
	TypeRefScope
	TypeRefPackage
	TypeRefImportSelector
	TypeRefBuiltin
)

type TypeRef struct {
	OwnerKind  int
	OwnerIndex int
	Kind       int
	Token      int
	BaseTok    int
	DotTok     int
	Package    int
	Symbol     int
}

type Program struct {
	Package    string
	Text       []byte
	Tokens     []Token
	Decls      []Decl
	Funcs      []Func
	Types      []TypeInfo
	TypeRefs   []TypeRef
	Indexes    []IndexExpr
	Composites []CompositeExpr
	Assigns    []Assignment
	Returns    []Return
	Calls      []Call
	Refs       []NameRef
	Selectors  []Selector
}

func Marshal(program Program) ([]byte, bool) {
	if len(program.Package) == 0 || len(program.Text) == 0 || len(program.Tokens) == 0 {
		return nil, false
	}
	tokenData, ok := encodeTokens(program.Text, program.Tokens)
	if !ok {
		return nil, false
	}
	declData, ok := encodeDecls(program.Decls)
	if !ok {
		return nil, false
	}
	funcData, ok := encodeFuncs(program.Funcs)
	if !ok {
		return nil, false
	}
	typeData, ok := encodeTypes(program.Types, len(program.Text), len(program.Tokens), len(program.Decls))
	if !ok {
		return nil, false
	}
	typeRefData, ok := encodeTypeRefs(program.TypeRefs, len(program.Tokens), len(program.Decls), len(program.Funcs))
	if !ok {
		return nil, false
	}
	indexData, ok := encodeIndexes(program.Indexes, len(program.Tokens), len(program.Decls), len(program.Funcs))
	if !ok {
		return nil, false
	}
	compData, ok := encodeComposites(program.Composites, len(program.Tokens), len(program.Decls), len(program.Funcs))
	if !ok {
		return nil, false
	}
	assignData, ok := encodeAssignments(program.Assigns, len(program.Tokens), len(program.Funcs))
	if !ok {
		return nil, false
	}
	returnData, ok := encodeReturns(program.Returns, len(program.Tokens), len(program.Funcs))
	if !ok {
		return nil, false
	}
	callData, ok := encodeCalls(program.Calls, len(program.Tokens), len(program.Decls), len(program.Funcs))
	if !ok {
		return nil, false
	}
	refData, ok := encodeRefs(program.Refs, len(program.Tokens), len(program.Decls), len(program.Funcs))
	if !ok {
		return nil, false
	}
	selectorData, ok := encodeSelectors(program.Selectors, len(program.Tokens), len(program.Decls), len(program.Funcs))
	if !ok {
		return nil, false
	}
	var root []byte
	root = appendNode(root, TagPackage, []byte(program.Package))
	root = appendNode(root, TagText, program.Text)
	root = appendNode(root, TagTokens, tokenData)
	root = appendNode(root, TagDecls, declData)
	root = appendNode(root, TagFuncs, funcData)
	root = appendNode(root, TagTypes, typeData)
	root = appendNode(root, TagTypeRefs, typeRefData)
	root = appendNode(root, TagIndexes, indexData)
	root = appendNode(root, TagComps, compData)
	root = appendNode(root, TagAssigns, assignData)
	root = appendNode(root, TagReturns, returnData)
	root = appendNode(root, TagCalls, callData)
	root = appendNode(root, TagRefs, refData)
	root = appendNode(root, TagSels, selectorData)

	out := make([]byte, 0, 14+len(root))
	out = append(out, 'R', 'T', 'G', 'U')
	out = appendUint16(out, Version)
	out = appendUint16(out, 0)
	out = appendNode(out, TagUnit, root)
	return out, true
}

func Unmarshal(data []byte) (Program, bool) {
	var program Program
	if len(data) < 14 {
		return program, false
	}
	if data[0] != 'R' || data[1] != 'T' || data[2] != 'G' || data[3] != 'U' {
		return program, false
	}
	if readUint16(data, 4) != Version {
		return program, false
	}
	rootTag := readUint16(data, 8)
	rootLength := readUint32(data, 10)
	if rootTag != TagUnit {
		return program, false
	}
	rootStart := 14
	rootEnd := rootStart + rootLength
	if rootEnd < rootStart || rootEnd != len(data) {
		return program, false
	}
	tokenData := []byte{}
	typeData := []byte{}
	typeRefData := []byte{}
	indexData := []byte{}
	compData := []byte{}
	assignData := []byte{}
	returnData := []byte{}
	callData := []byte{}
	refData := []byte{}
	selectorData := []byte{}
	seenPackage := false
	seenText := false
	seenTokens := false
	seenDecls := false
	seenFuncs := false
	seenTypes := false
	seenTypeRefs := false
	seenIndexes := false
	seenComps := false
	seenAssigns := false
	seenReturns := false
	seenCalls := false
	seenRefs := false
	seenSelectors := false
	pos := rootStart
	for pos < rootEnd {
		if pos+6 > rootEnd {
			return program, false
		}
		tag := readUint16(data, pos)
		length := readUint32(data, pos+2)
		pos += 6
		next := pos + length
		if next < pos || next > rootEnd {
			return program, false
		}
		payload := data[pos:next]
		if tag == TagPackage {
			if seenPackage {
				return program, false
			}
			seenPackage = true
			program.Package = string(payload)
		} else if tag == TagText {
			if seenText {
				return program, false
			}
			seenText = true
			program.Text = payload
		} else if tag == TagTokens {
			if seenTokens {
				return program, false
			}
			seenTokens = true
			tokenData = payload
		} else if tag == TagDecls {
			if seenDecls {
				return program, false
			}
			seenDecls = true
			decls, ok := decodeDecls(payload)
			if !ok {
				return program, false
			}
			program.Decls = decls
		} else if tag == TagFuncs {
			if seenFuncs {
				return program, false
			}
			seenFuncs = true
			funcs, ok := decodeFuncs(payload)
			if !ok {
				return program, false
			}
			program.Funcs = funcs
		} else if tag == TagTypes {
			if seenTypes {
				return program, false
			}
			seenTypes = true
			typeData = payload
		} else if tag == TagTypeRefs {
			if seenTypeRefs {
				return program, false
			}
			seenTypeRefs = true
			typeRefData = payload
		} else if tag == TagIndexes {
			if seenIndexes {
				return program, false
			}
			seenIndexes = true
			indexData = payload
		} else if tag == TagComps {
			if seenComps {
				return program, false
			}
			seenComps = true
			compData = payload
		} else if tag == TagAssigns {
			if seenAssigns {
				return program, false
			}
			seenAssigns = true
			assignData = payload
		} else if tag == TagReturns {
			if seenReturns {
				return program, false
			}
			seenReturns = true
			returnData = payload
		} else if tag == TagCalls {
			if seenCalls {
				return program, false
			}
			seenCalls = true
			callData = payload
		} else if tag == TagRefs {
			if seenRefs {
				return program, false
			}
			seenRefs = true
			refData = payload
		} else if tag == TagSels {
			if seenSelectors {
				return program, false
			}
			seenSelectors = true
			selectorData = payload
		} else {
			return program, false
		}
		pos = next
	}
	if !seenPackage || !seenText || !seenTokens || !seenDecls || !seenFuncs {
		return program, false
	}
	if len(program.Package) == 0 || len(program.Text) == 0 {
		return program, false
	}
	tokens, ok := decodeTokens(program.Text, tokenData)
	if !ok || len(tokens) == 0 {
		return program, false
	}
	program.Tokens = tokens
	if seenTypes {
		types, ok := decodeTypes(typeData, len(program.Text), len(program.Tokens), len(program.Decls))
		if !ok {
			return program, false
		}
		program.Types = types
	}
	if seenTypeRefs {
		typeRefs, ok := decodeTypeRefs(typeRefData, len(program.Tokens), len(program.Decls), len(program.Funcs))
		if !ok {
			return program, false
		}
		program.TypeRefs = typeRefs
	}
	if seenIndexes {
		indexes, ok := decodeIndexes(indexData, len(program.Tokens), len(program.Decls), len(program.Funcs))
		if !ok {
			return program, false
		}
		program.Indexes = indexes
	}
	if seenComps {
		composites, ok := decodeComposites(compData, len(program.Tokens), len(program.Decls), len(program.Funcs))
		if !ok {
			return program, false
		}
		program.Composites = composites
	}
	if seenAssigns {
		assigns, ok := decodeAssignments(assignData, len(program.Tokens), len(program.Funcs))
		if !ok {
			return program, false
		}
		program.Assigns = assigns
	}
	if seenReturns {
		returns, ok := decodeReturns(returnData, len(program.Tokens), len(program.Funcs))
		if !ok {
			return program, false
		}
		program.Returns = returns
	}
	if seenCalls {
		calls, ok := decodeCalls(callData, len(program.Tokens), len(program.Decls), len(program.Funcs))
		if !ok {
			return program, false
		}
		program.Calls = calls
	}
	if seenRefs {
		refs, ok := decodeRefs(refData, len(program.Tokens), len(program.Decls), len(program.Funcs))
		if !ok {
			return program, false
		}
		program.Refs = refs
	}
	if seenSelectors {
		selectors, ok := decodeSelectors(selectorData, len(program.Tokens), len(program.Decls), len(program.Funcs))
		if !ok {
			return program, false
		}
		program.Selectors = selectors
	}
	return program, true
}

func encodeTokens(text []byte, tokens []Token) ([]byte, bool) {
	out := make([]byte, 0, len(tokens)*4)
	out = appendVarint(out, len(tokens))
	prevStart := 0
	prevLine := 0
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok.Kind < 0 || tok.Kind > 255 || tok.Start < prevStart || tok.Line < prevLine || tok.Size < 0 {
			return nil, false
		}
		if tok.Start > len(text) || tok.Start+tok.Size > len(text) {
			return nil, false
		}
		if tok.Start > 0xffffff || tok.Line > 0xffff {
			return nil, false
		}
		if tok.Kind == TokenOp {
			if tok.Size > 255 {
				return nil, false
			}
		} else if tok.Size > 0xffff {
			return nil, false
		}
		out = appendVarint(out, tok.Kind)
		out = appendVarint(out, tok.Start-prevStart)
		out = appendVarint(out, tok.Size)
		out = appendVarint(out, tok.Line-prevLine)
		prevStart = tok.Start
		prevLine = tok.Line
	}
	return out, true
}

func decodeTokens(text []byte, data []byte) ([]Token, bool) {
	pos := 0
	count, ok := readVarint(data, &pos)
	if !ok || count < 0 {
		return nil, false
	}
	tokens := make([]Token, 0, count)
	prevStart := 0
	prevLine := 0
	for i := 0; i < count; i++ {
		kind, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		startDelta, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		size, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		lineDelta, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		tok := Token{
			Kind:  kind,
			Start: prevStart + startDelta,
			Size:  size,
			Line:  prevLine + lineDelta,
		}
		if tok.Kind < 0 || tok.Kind > 255 || tok.Size < 0 || tok.Start < prevStart || tok.Line < prevLine {
			return nil, false
		}
		if tok.Start > len(text) || tok.Start+tok.Size > len(text) {
			return nil, false
		}
		if tok.Start > 0xffffff || tok.Line > 0xffff {
			return nil, false
		}
		if tok.Kind == TokenOp {
			if tok.Size > 255 {
				return nil, false
			}
		} else if tok.Size > 0xffff {
			return nil, false
		}
		tokens = append(tokens, tok)
		prevStart = tok.Start
		prevLine = tok.Line
	}
	if pos != len(data) {
		return nil, false
	}
	return tokens, true
}

func encodeDecls(decls []Decl) ([]byte, bool) {
	out := make([]byte, 0, len(decls)*5+1)
	out = appendVarint(out, len(decls))
	for i := 0; i < len(decls); i++ {
		decl := decls[i]
		if decl.Kind < 0 || decl.NameStart < 0 || decl.NameEnd < decl.NameStart || decl.StartTok < 0 || decl.EndTok < decl.StartTok {
			return nil, false
		}
		out = appendVarint(out, decl.Kind)
		out = appendVarint(out, decl.NameStart)
		out = appendVarint(out, decl.NameEnd-decl.NameStart)
		out = appendVarint(out, decl.StartTok)
		out = appendVarint(out, decl.EndTok-decl.StartTok)
	}
	return out, true
}

func decodeDecls(data []byte) ([]Decl, bool) {
	pos := 0
	count, ok := readVarint(data, &pos)
	if !ok || count < 0 {
		return nil, false
	}
	decls := make([]Decl, 0, count)
	for i := 0; i < count; i++ {
		kind, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		nameStart, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		nameSize, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		startTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		tokenSize, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		decl := Decl{
			Kind:      kind,
			NameStart: nameStart,
			NameEnd:   nameStart + nameSize,
			StartTok:  startTok,
			EndTok:    startTok + tokenSize,
		}
		if decl.Kind < 0 || decl.NameStart < 0 || decl.NameEnd < decl.NameStart || decl.StartTok < 0 || decl.EndTok < decl.StartTok {
			return nil, false
		}
		decls = append(decls, decl)
	}
	if pos != len(data) {
		return nil, false
	}
	return decls, true
}

func encodeFuncs(funcs []Func) ([]byte, bool) {
	out := make([]byte, 0, len(funcs)*9+1)
	out = appendVarint(out, len(funcs))
	for i := 0; i < len(funcs); i++ {
		fn := funcs[i]
		if fn.NameStart < 0 || fn.NameEnd < fn.NameStart || fn.StartTok < 0 || fn.NameTok < fn.StartTok {
			return nil, false
		}
		if fn.ReceiverStart < 0 || fn.ReceiverEnd < fn.ReceiverStart || fn.BodyStart < 0 || fn.BodyEnd < fn.BodyStart || fn.EndTok < fn.BodyEnd {
			return nil, false
		}
		out = appendVarint(out, fn.NameStart)
		out = appendVarint(out, fn.NameEnd-fn.NameStart)
		out = appendVarint(out, fn.StartTok)
		out = appendVarint(out, fn.NameTok-fn.StartTok)
		out = appendVarint(out, fn.ReceiverStart)
		out = appendVarint(out, fn.ReceiverEnd-fn.ReceiverStart)
		out = appendVarint(out, fn.BodyStart)
		out = appendVarint(out, fn.BodyEnd-fn.BodyStart)
		out = appendVarint(out, fn.EndTok-fn.BodyEnd)
	}
	return out, true
}

func decodeFuncs(data []byte) ([]Func, bool) {
	pos := 0
	count, ok := readVarint(data, &pos)
	if !ok || count < 0 {
		return nil, false
	}
	funcs := make([]Func, 0, count)
	for i := 0; i < count; i++ {
		nameStart, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		nameSize, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		startTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		nameDelta, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		receiverStart, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		receiverSize, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		bodyStart, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		bodySize, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		endSize, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		fn := Func{
			NameStart:     nameStart,
			NameEnd:       nameStart + nameSize,
			StartTok:      startTok,
			NameTok:       startTok + nameDelta,
			ReceiverStart: receiverStart,
			ReceiverEnd:   receiverStart + receiverSize,
			BodyStart:     bodyStart,
			BodyEnd:       bodyStart + bodySize,
			EndTok:        bodyStart + bodySize + endSize,
		}
		if fn.NameStart < 0 || fn.NameEnd < fn.NameStart || fn.StartTok < 0 || fn.NameTok < fn.StartTok {
			return nil, false
		}
		if fn.ReceiverStart < 0 || fn.ReceiverEnd < fn.ReceiverStart || fn.BodyStart < 0 || fn.BodyEnd < fn.BodyStart || fn.EndTok < fn.BodyEnd {
			return nil, false
		}
		funcs = append(funcs, fn)
	}
	if pos != len(data) {
		return nil, false
	}
	return funcs, true
}

func encodeTypes(types []TypeInfo, textLimit int, tokenLimit int, declLimit int) ([]byte, bool) {
	out := make([]byte, 0, len(types)*14+1)
	out = appendVarint(out, len(types))
	ok := true
	for i := 0; i < len(types); i++ {
		typ := types[i]
		if typ.Kind < TypeOther || typ.Kind > TypeFunc ||
			!validTextSpan(textLimit, typ.NameStart, typ.NameEnd) ||
			typ.Decl < 0 || typ.Decl >= declLimit ||
			!validNullable(typ.Symbol) {
			return nil, false
		}
		out = appendVarint(out, typ.Kind)
		out = appendVarint(out, typ.NameStart)
		out = appendVarint(out, typ.NameEnd-typ.NameStart)
		out = appendVarint(out, typ.Decl)
		out = appendNullable(out, typ.Symbol)
		if typ.Alias {
			out = appendVarint(out, 1)
		} else {
			out = appendVarint(out, 0)
		}
		out = appendNullableSpan(out, typ.TypeStart, typ.TypeEnd, tokenLimit, &ok)
		out = appendNullableSpan(out, typ.LenStart, typ.LenEnd, tokenLimit, &ok)
		out = appendNullableSpan(out, typ.KeyStart, typ.KeyEnd, tokenLimit, &ok)
		out = appendNullableSpan(out, typ.ElemStart, typ.ElemEnd, tokenLimit, &ok)
		if !ok {
			return nil, false
		}
	}
	return out, true
}

func decodeTypes(data []byte, textLimit int, tokenLimit int, declLimit int) ([]TypeInfo, bool) {
	pos := 0
	count, ok := readVarint(data, &pos)
	if !ok || count < 0 {
		return nil, false
	}
	types := make([]TypeInfo, 0, count)
	for i := 0; i < count; i++ {
		kind, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		nameStart, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		nameSize, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		decl, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		symbol, ok := readNullable(data, &pos)
		if !ok {
			return nil, false
		}
		aliasValue, ok := readVarint(data, &pos)
		if !ok || aliasValue > 1 {
			return nil, false
		}
		typeStart, typeEnd, ok := readNullableSpan(data, &pos, tokenLimit)
		if !ok {
			return nil, false
		}
		lenStart, lenEnd, ok := readNullableSpan(data, &pos, tokenLimit)
		if !ok {
			return nil, false
		}
		keyStart, keyEnd, ok := readNullableSpan(data, &pos, tokenLimit)
		if !ok {
			return nil, false
		}
		elemStart, elemEnd, ok := readNullableSpan(data, &pos, tokenLimit)
		if !ok {
			return nil, false
		}
		typ := TypeInfo{
			NameStart: nameStart,
			NameEnd:   nameStart + nameSize,
			Kind:      kind,
			Decl:      decl,
			Symbol:    symbol,
			Alias:     aliasValue == 1,
			TypeStart: typeStart,
			TypeEnd:   typeEnd,
			LenStart:  lenStart,
			LenEnd:    lenEnd,
			KeyStart:  keyStart,
			KeyEnd:    keyEnd,
			ElemStart: elemStart,
			ElemEnd:   elemEnd,
		}
		if typ.Kind < TypeOther || typ.Kind > TypeFunc ||
			!validTextSpan(textLimit, typ.NameStart, typ.NameEnd) ||
			typ.Decl < 0 || typ.Decl >= declLimit ||
			!validNullable(typ.Symbol) {
			return nil, false
		}
		types = append(types, typ)
	}
	if pos != len(data) {
		return nil, false
	}
	return types, true
}

func encodeTypeRefs(refs []TypeRef, tokenLimit int, declLimit int, funcLimit int) ([]byte, bool) {
	out := make([]byte, 0, len(refs)*9+1)
	out = appendVarint(out, len(refs))
	for i := 0; i < len(refs); i++ {
		ref := refs[i]
		if !validOwner(ref.OwnerKind, ref.OwnerIndex, declLimit, funcLimit) ||
			ref.Kind < TypeRefUnknown || ref.Kind > TypeRefBuiltin ||
			!validToken(tokenLimit, ref.Token) ||
			!validToken(tokenLimit, ref.BaseTok) ||
			!validToken(tokenLimit, ref.DotTok) ||
			!validNullable(ref.Package) ||
			!validNullable(ref.Symbol) {
			return nil, false
		}
		out = appendVarint(out, ref.OwnerKind)
		out = appendVarint(out, ref.OwnerIndex)
		out = appendVarint(out, ref.Kind)
		out = appendVarint(out, ref.Token)
		out = appendVarint(out, ref.BaseTok)
		out = appendVarint(out, ref.DotTok)
		out = appendNullable(out, ref.Package)
		out = appendNullable(out, ref.Symbol)
	}
	return out, true
}

func decodeTypeRefs(data []byte, tokenLimit int, declLimit int, funcLimit int) ([]TypeRef, bool) {
	pos := 0
	count, ok := readVarint(data, &pos)
	if !ok || count < 0 {
		return nil, false
	}
	refs := make([]TypeRef, 0, count)
	for i := 0; i < count; i++ {
		ownerKind, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		ownerIndex, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		kind, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		token, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		baseTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		dotTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		pkg, ok := readNullable(data, &pos)
		if !ok {
			return nil, false
		}
		symbol, ok := readNullable(data, &pos)
		if !ok {
			return nil, false
		}
		ref := TypeRef{
			OwnerKind:  ownerKind,
			OwnerIndex: ownerIndex,
			Kind:       kind,
			Token:      token,
			BaseTok:    baseTok,
			DotTok:     dotTok,
			Package:    pkg,
			Symbol:     symbol,
		}
		if !validOwner(ref.OwnerKind, ref.OwnerIndex, declLimit, funcLimit) ||
			ref.Kind < TypeRefUnknown || ref.Kind > TypeRefBuiltin ||
			!validToken(tokenLimit, ref.Token) ||
			!validToken(tokenLimit, ref.BaseTok) ||
			!validToken(tokenLimit, ref.DotTok) ||
			!validNullable(ref.Package) ||
			!validNullable(ref.Symbol) {
			return nil, false
		}
		refs = append(refs, ref)
	}
	if pos != len(data) {
		return nil, false
	}
	return refs, true
}

func encodeIndexes(indexes []IndexExpr, tokenLimit int, declLimit int, funcLimit int) ([]byte, bool) {
	out := make([]byte, 0, len(indexes)*10+1)
	out = appendVarint(out, len(indexes))
	for i := 0; i < len(indexes); i++ {
		index := indexes[i]
		if !validOwner(index.OwnerKind, index.OwnerIndex, declLimit, funcLimit) {
			return nil, false
		}
		if !validSpan(tokenLimit, index.StartTok, index.EndTok) ||
			!validSpan(tokenLimit, index.BaseStart, index.BaseEnd) ||
			!validSpan(tokenLimit, index.IndexStart, index.IndexEnd) ||
			!validToken(tokenLimit, index.OpenTok) ||
			!validToken(tokenLimit, index.CloseTok) {
			return nil, false
		}
		out = appendVarint(out, index.OwnerKind)
		out = appendVarint(out, index.OwnerIndex)
		out = appendVarint(out, index.StartTok)
		out = appendVarint(out, index.EndTok-index.StartTok)
		out = appendVarint(out, index.BaseStart)
		out = appendVarint(out, index.BaseEnd-index.BaseStart)
		out = appendVarint(out, index.OpenTok)
		out = appendVarint(out, index.CloseTok)
		out = appendVarint(out, index.IndexStart)
		out = appendVarint(out, index.IndexEnd-index.IndexStart)
	}
	return out, true
}

func decodeIndexes(data []byte, tokenLimit int, declLimit int, funcLimit int) ([]IndexExpr, bool) {
	pos := 0
	count, ok := readVarint(data, &pos)
	if !ok || count < 0 {
		return nil, false
	}
	indexes := make([]IndexExpr, 0, count)
	for i := 0; i < count; i++ {
		ownerKind, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		ownerIndex, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		startTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		tokCount, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		baseStart, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		baseCount, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		openTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		closeTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		indexStart, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		indexCount, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		index := IndexExpr{
			OwnerKind:  ownerKind,
			OwnerIndex: ownerIndex,
			StartTok:   startTok,
			EndTok:     startTok + tokCount,
			BaseStart:  baseStart,
			BaseEnd:    baseStart + baseCount,
			OpenTok:    openTok,
			CloseTok:   closeTok,
			IndexStart: indexStart,
			IndexEnd:   indexStart + indexCount,
		}
		if !validOwner(index.OwnerKind, index.OwnerIndex, declLimit, funcLimit) ||
			!validSpan(tokenLimit, index.StartTok, index.EndTok) ||
			!validSpan(tokenLimit, index.BaseStart, index.BaseEnd) ||
			!validSpan(tokenLimit, index.IndexStart, index.IndexEnd) ||
			!validToken(tokenLimit, index.OpenTok) ||
			!validToken(tokenLimit, index.CloseTok) {
			return nil, false
		}
		indexes = append(indexes, index)
	}
	if pos != len(data) {
		return nil, false
	}
	return indexes, true
}

func encodeComposites(composites []CompositeExpr, tokenLimit int, declLimit int, funcLimit int) ([]byte, bool) {
	out := make([]byte, 0, len(composites)*10+1)
	out = appendVarint(out, len(composites))
	for i := 0; i < len(composites); i++ {
		composite := composites[i]
		if !validOwner(composite.OwnerKind, composite.OwnerIndex, declLimit, funcLimit) {
			return nil, false
		}
		if !validSpan(tokenLimit, composite.StartTok, composite.EndTok) ||
			!validSpan(tokenLimit, composite.TypeStart, composite.TypeEnd) ||
			!validToken(tokenLimit, composite.OpenTok) ||
			!validToken(tokenLimit, composite.CloseTok) {
			return nil, false
		}
		out = appendVarint(out, composite.OwnerKind)
		out = appendVarint(out, composite.OwnerIndex)
		out = appendVarint(out, composite.StartTok)
		out = appendVarint(out, composite.EndTok-composite.StartTok)
		out = appendVarint(out, composite.TypeStart)
		out = appendVarint(out, composite.TypeEnd-composite.TypeStart)
		out = appendVarint(out, composite.OpenTok)
		out = appendVarint(out, composite.CloseTok)
		out = appendVarint(out, len(composite.Elems))
		for j := 0; j < len(composite.Elems); j++ {
			elem := composite.Elems[j]
			if !validSpan(tokenLimit, elem.StartTok, elem.EndTok) {
				return nil, false
			}
			out = appendVarint(out, elem.StartTok)
			out = appendVarint(out, elem.EndTok-elem.StartTok)
		}
	}
	return out, true
}

func decodeComposites(data []byte, tokenLimit int, declLimit int, funcLimit int) ([]CompositeExpr, bool) {
	pos := 0
	count, ok := readVarint(data, &pos)
	if !ok || count < 0 {
		return nil, false
	}
	composites := make([]CompositeExpr, 0, count)
	for i := 0; i < count; i++ {
		ownerKind, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		ownerIndex, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		startTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		tokCount, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		typeStart, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		typeCount, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		openTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		closeTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		elemCount, ok := readVarint(data, &pos)
		if !ok || elemCount < 0 {
			return nil, false
		}
		composite := CompositeExpr{
			OwnerKind:  ownerKind,
			OwnerIndex: ownerIndex,
			StartTok:   startTok,
			EndTok:     startTok + tokCount,
			TypeStart:  typeStart,
			TypeEnd:    typeStart + typeCount,
			OpenTok:    openTok,
			CloseTok:   closeTok,
			Elems:      make([]ExprSpan, 0, elemCount),
		}
		if !validOwner(composite.OwnerKind, composite.OwnerIndex, declLimit, funcLimit) ||
			!validSpan(tokenLimit, composite.StartTok, composite.EndTok) ||
			!validSpan(tokenLimit, composite.TypeStart, composite.TypeEnd) ||
			!validToken(tokenLimit, composite.OpenTok) ||
			!validToken(tokenLimit, composite.CloseTok) {
			return nil, false
		}
		for j := 0; j < elemCount; j++ {
			elemStart, ok := readVarint(data, &pos)
			if !ok {
				return nil, false
			}
			elemCount, ok := readVarint(data, &pos)
			if !ok {
				return nil, false
			}
			elem := ExprSpan{StartTok: elemStart, EndTok: elemStart + elemCount}
			if !validSpan(tokenLimit, elem.StartTok, elem.EndTok) {
				return nil, false
			}
			composite.Elems = append(composite.Elems, elem)
		}
		composites = append(composites, composite)
	}
	if pos != len(data) {
		return nil, false
	}
	return composites, true
}

func encodeAssignments(assigns []Assignment, tokenLimit int, funcLimit int) ([]byte, bool) {
	out := make([]byte, 0, len(assigns)*14+1)
	out = appendVarint(out, len(assigns))
	for i := 0; i < len(assigns); i++ {
		assign := assigns[i]
		ok := true
		if assign.FuncIndex < 0 || assign.FuncIndex >= funcLimit || assign.Kind < AssignUnknown || assign.Kind > AssignXor {
			return nil, false
		}
		if !validSpan(tokenLimit, assign.StartTok, assign.EndTok) ||
			!validSpan(tokenLimit, assign.LeftStart, assign.LeftEnd) ||
			!validSpan(tokenLimit, assign.RightStart, assign.RightEnd) ||
			!validToken(tokenLimit, assign.OpTok) {
			return nil, false
		}
		out = appendVarint(out, assign.FuncIndex)
		out = appendVarint(out, assign.Kind)
		out = appendVarint(out, assign.StartTok)
		out = appendVarint(out, assign.EndTok-assign.StartTok)
		out = appendVarint(out, assign.OpTok)
		out = appendVarint(out, assign.LeftStart)
		out = appendVarint(out, assign.LeftEnd-assign.LeftStart)
		out = appendVarint(out, assign.RightStart)
		out = appendVarint(out, assign.RightEnd-assign.RightStart)
		out = appendExprSpans(out, assign.Targets, tokenLimit, &ok)
		if !ok {
			return nil, false
		}
		out = appendExprSpans(out, assign.Values, tokenLimit, &ok)
		if !ok {
			return nil, false
		}
	}
	return out, true
}

func decodeAssignments(data []byte, tokenLimit int, funcLimit int) ([]Assignment, bool) {
	pos := 0
	count, ok := readVarint(data, &pos)
	if !ok || count < 0 {
		return nil, false
	}
	assigns := make([]Assignment, 0, count)
	for i := 0; i < count; i++ {
		funcIndex, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		kind, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		startTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		tokCount, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		opTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		leftStart, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		leftCount, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		rightStart, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		rightCount, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		targets, ok := readExprSpans(data, &pos, tokenLimit)
		if !ok {
			return nil, false
		}
		values, ok := readExprSpans(data, &pos, tokenLimit)
		if !ok {
			return nil, false
		}
		assign := Assignment{
			FuncIndex:  funcIndex,
			Kind:       kind,
			StartTok:   startTok,
			EndTok:     startTok + tokCount,
			OpTok:      opTok,
			LeftStart:  leftStart,
			LeftEnd:    leftStart + leftCount,
			RightStart: rightStart,
			RightEnd:   rightStart + rightCount,
			Targets:    targets,
			Values:     values,
		}
		if assign.FuncIndex < 0 || assign.FuncIndex >= funcLimit || assign.Kind < AssignUnknown || assign.Kind > AssignXor {
			return nil, false
		}
		if !validSpan(tokenLimit, assign.StartTok, assign.EndTok) ||
			!validSpan(tokenLimit, assign.LeftStart, assign.LeftEnd) ||
			!validSpan(tokenLimit, assign.RightStart, assign.RightEnd) ||
			!validToken(tokenLimit, assign.OpTok) {
			return nil, false
		}
		assigns = append(assigns, assign)
	}
	if pos != len(data) {
		return nil, false
	}
	return assigns, true
}

func encodeReturns(returns []Return, tokenLimit int, funcLimit int) ([]byte, bool) {
	out := make([]byte, 0, len(returns)*5+1)
	out = appendVarint(out, len(returns))
	ok := true
	for i := 0; i < len(returns); i++ {
		ret := returns[i]
		if ret.FuncIndex < 0 || ret.FuncIndex >= funcLimit || !validSpan(tokenLimit, ret.StartTok, ret.EndTok) {
			return nil, false
		}
		out = appendVarint(out, ret.FuncIndex)
		out = appendVarint(out, ret.StartTok)
		out = appendVarint(out, ret.EndTok-ret.StartTok)
		out = appendExprSpans(out, ret.Values, tokenLimit, &ok)
		if !ok {
			return nil, false
		}
	}
	return out, true
}

func decodeReturns(data []byte, tokenLimit int, funcLimit int) ([]Return, bool) {
	pos := 0
	count, ok := readVarint(data, &pos)
	if !ok || count < 0 {
		return nil, false
	}
	returns := make([]Return, 0, count)
	for i := 0; i < count; i++ {
		funcIndex, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		startTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		tokCount, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		values, ok := readExprSpans(data, &pos, tokenLimit)
		if !ok {
			return nil, false
		}
		ret := Return{
			FuncIndex: funcIndex,
			StartTok:  startTok,
			EndTok:    startTok + tokCount,
			Values:    values,
		}
		if ret.FuncIndex < 0 || ret.FuncIndex >= funcLimit || !validSpan(tokenLimit, ret.StartTok, ret.EndTok) {
			return nil, false
		}
		returns = append(returns, ret)
	}
	if pos != len(data) {
		return nil, false
	}
	return returns, true
}

func encodeCalls(calls []Call, tokenLimit int, declLimit int, funcLimit int) ([]byte, bool) {
	out := make([]byte, 0, len(calls)*10+1)
	out = appendVarint(out, len(calls))
	for i := 0; i < len(calls); i++ {
		call := calls[i]
		ok := true
		if !validOwner(call.OwnerKind, call.OwnerIndex, declLimit, funcLimit) || call.Kind < CallUnknown || call.Kind > CallBuiltin {
			return nil, false
		}
		if !validToken(tokenLimit, call.CalleeTok) ||
			!validToken(tokenLimit, call.BaseTok) ||
			!validToken(tokenLimit, call.DotTok) ||
			!validSpan(tokenLimit, call.ArgsStart, call.ArgsEnd) {
			return nil, false
		}
		out = appendVarint(out, call.OwnerKind)
		out = appendVarint(out, call.OwnerIndex)
		out = appendVarint(out, call.Kind)
		out = appendVarint(out, call.CalleeTok)
		out = appendVarint(out, call.BaseTok)
		out = appendVarint(out, call.DotTok)
		out = appendVarint(out, call.ArgsStart)
		out = appendVarint(out, call.ArgsEnd-call.ArgsStart)
		out = appendExprSpans(out, call.Args, tokenLimit, &ok)
		if !ok {
			return nil, false
		}
	}
	return out, true
}

func decodeCalls(data []byte, tokenLimit int, declLimit int, funcLimit int) ([]Call, bool) {
	pos := 0
	count, ok := readVarint(data, &pos)
	if !ok || count < 0 {
		return nil, false
	}
	calls := make([]Call, 0, count)
	for i := 0; i < count; i++ {
		ownerKind, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		ownerIndex, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		kind, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		calleeTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		baseTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		dotTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		argsStart, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		argsCount, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		args, ok := readExprSpans(data, &pos, tokenLimit)
		if !ok {
			return nil, false
		}
		call := Call{
			OwnerKind:  ownerKind,
			OwnerIndex: ownerIndex,
			Kind:       kind,
			CalleeTok:  calleeTok,
			BaseTok:    baseTok,
			DotTok:     dotTok,
			ArgsStart:  argsStart,
			ArgsEnd:    argsStart + argsCount,
			Args:       args,
		}
		if !validOwner(call.OwnerKind, call.OwnerIndex, declLimit, funcLimit) || call.Kind < CallUnknown || call.Kind > CallBuiltin {
			return nil, false
		}
		if !validToken(tokenLimit, call.CalleeTok) ||
			!validToken(tokenLimit, call.BaseTok) ||
			!validToken(tokenLimit, call.DotTok) ||
			!validSpan(tokenLimit, call.ArgsStart, call.ArgsEnd) {
			return nil, false
		}
		calls = append(calls, call)
	}
	if pos != len(data) {
		return nil, false
	}
	return calls, true
}

func encodeRefs(refs []NameRef, tokenLimit int, declLimit int, funcLimit int) ([]byte, bool) {
	out := make([]byte, 0, len(refs)*7+1)
	out = appendVarint(out, len(refs))
	for i := 0; i < len(refs); i++ {
		ref := refs[i]
		if !validOwner(ref.OwnerKind, ref.OwnerIndex, declLimit, funcLimit) ||
			ref.Kind < RefUnknown || ref.Kind > RefLabel ||
			!validToken(tokenLimit, ref.Token) ||
			!validNullable(ref.Index) ||
			!validNullable(ref.Package) {
			return nil, false
		}
		out = appendVarint(out, ref.OwnerKind)
		out = appendVarint(out, ref.OwnerIndex)
		out = appendVarint(out, ref.Kind)
		out = appendVarint(out, ref.Token)
		out = appendNullable(out, ref.Index)
		out = appendNullable(out, ref.Package)
	}
	return out, true
}

func decodeRefs(data []byte, tokenLimit int, declLimit int, funcLimit int) ([]NameRef, bool) {
	pos := 0
	count, ok := readVarint(data, &pos)
	if !ok || count < 0 {
		return nil, false
	}
	refs := make([]NameRef, 0, count)
	for i := 0; i < count; i++ {
		ownerKind, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		ownerIndex, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		kind, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		token, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		index, ok := readNullable(data, &pos)
		if !ok {
			return nil, false
		}
		pkg, ok := readNullable(data, &pos)
		if !ok {
			return nil, false
		}
		ref := NameRef{
			OwnerKind:  ownerKind,
			OwnerIndex: ownerIndex,
			Kind:       kind,
			Token:      token,
			Index:      index,
			Package:    pkg,
		}
		if !validOwner(ref.OwnerKind, ref.OwnerIndex, declLimit, funcLimit) ||
			ref.Kind < RefUnknown || ref.Kind > RefLabel ||
			!validToken(tokenLimit, ref.Token) ||
			!validNullable(ref.Index) ||
			!validNullable(ref.Package) {
			return nil, false
		}
		refs = append(refs, ref)
	}
	if pos != len(data) {
		return nil, false
	}
	return refs, true
}

func encodeSelectors(selectors []Selector, tokenLimit int, declLimit int, funcLimit int) ([]byte, bool) {
	out := make([]byte, 0, len(selectors)*12+1)
	out = appendVarint(out, len(selectors))
	for i := 0; i < len(selectors); i++ {
		selector := selectors[i]
		if !validOwner(selector.OwnerKind, selector.OwnerIndex, declLimit, funcLimit) ||
			selector.Kind < SelectorUnknown || selector.Kind > SelectorImport ||
			!validToken(tokenLimit, selector.BaseTok) ||
			!validToken(tokenLimit, selector.DotTok) ||
			!validToken(tokenLimit, selector.NameTok) ||
			selector.BaseKind < RefUnknown || selector.BaseKind > RefLabel ||
			!validNullable(selector.BaseIndex) ||
			!validNullable(selector.BasePackage) ||
			!validNullable(selector.Package) ||
			!validNullable(selector.Symbol) {
			return nil, false
		}
		out = appendVarint(out, selector.OwnerKind)
		out = appendVarint(out, selector.OwnerIndex)
		out = appendVarint(out, selector.Kind)
		out = appendVarint(out, selector.BaseTok)
		out = appendVarint(out, selector.DotTok)
		out = appendVarint(out, selector.NameTok)
		out = appendVarint(out, selector.BaseKind)
		out = appendNullable(out, selector.BaseIndex)
		out = appendNullable(out, selector.BasePackage)
		out = appendNullable(out, selector.Package)
		out = appendNullable(out, selector.Symbol)
	}
	return out, true
}

func decodeSelectors(data []byte, tokenLimit int, declLimit int, funcLimit int) ([]Selector, bool) {
	pos := 0
	count, ok := readVarint(data, &pos)
	if !ok || count < 0 {
		return nil, false
	}
	selectors := make([]Selector, 0, count)
	for i := 0; i < count; i++ {
		ownerKind, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		ownerIndex, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		kind, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		baseTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		dotTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		nameTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		baseKind, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		baseIndex, ok := readNullable(data, &pos)
		if !ok {
			return nil, false
		}
		basePackage, ok := readNullable(data, &pos)
		if !ok {
			return nil, false
		}
		pkg, ok := readNullable(data, &pos)
		if !ok {
			return nil, false
		}
		symbol, ok := readNullable(data, &pos)
		if !ok {
			return nil, false
		}
		selector := Selector{
			OwnerKind:   ownerKind,
			OwnerIndex:  ownerIndex,
			Kind:        kind,
			BaseTok:     baseTok,
			DotTok:      dotTok,
			NameTok:     nameTok,
			BaseKind:    baseKind,
			BaseIndex:   baseIndex,
			BasePackage: basePackage,
			Package:     pkg,
			Symbol:      symbol,
		}
		if !validOwner(selector.OwnerKind, selector.OwnerIndex, declLimit, funcLimit) ||
			selector.Kind < SelectorUnknown || selector.Kind > SelectorImport ||
			!validToken(tokenLimit, selector.BaseTok) ||
			!validToken(tokenLimit, selector.DotTok) ||
			!validToken(tokenLimit, selector.NameTok) ||
			selector.BaseKind < RefUnknown || selector.BaseKind > RefLabel ||
			!validNullable(selector.BaseIndex) ||
			!validNullable(selector.BasePackage) ||
			!validNullable(selector.Package) ||
			!validNullable(selector.Symbol) {
			return nil, false
		}
		selectors = append(selectors, selector)
	}
	if pos != len(data) {
		return nil, false
	}
	return selectors, true
}

func appendExprSpans(out []byte, spans []ExprSpan, tokenLimit int, ok *bool) []byte {
	out = appendVarint(out, len(spans))
	for i := 0; i < len(spans); i++ {
		span := spans[i]
		if !validSpan(tokenLimit, span.StartTok, span.EndTok) {
			*ok = false
			return out
		}
		out = appendVarint(out, span.StartTok)
		out = appendVarint(out, span.EndTok-span.StartTok)
	}
	return out
}

func readExprSpans(data []byte, pos *int, tokenLimit int) ([]ExprSpan, bool) {
	count, ok := readVarint(data, pos)
	if !ok || count < 0 {
		return nil, false
	}
	spans := make([]ExprSpan, 0, count)
	for i := 0; i < count; i++ {
		startTok, ok := readVarint(data, pos)
		if !ok {
			return nil, false
		}
		tokCount, ok := readVarint(data, pos)
		if !ok {
			return nil, false
		}
		span := ExprSpan{StartTok: startTok, EndTok: startTok + tokCount}
		if !validSpan(tokenLimit, span.StartTok, span.EndTok) {
			return nil, false
		}
		spans = append(spans, span)
	}
	return spans, true
}

func appendNullableSpan(out []byte, start int, end int, tokenLimit int, ok *bool) []byte {
	if start < 0 && end < 0 {
		return appendVarint(out, 0)
	}
	if !validSpan(tokenLimit, start, end) {
		*ok = false
		return out
	}
	out = appendVarint(out, start+1)
	out = appendVarint(out, end-start)
	return out
}

func readNullableSpan(data []byte, pos *int, tokenLimit int) (int, int, bool) {
	encodedStart, ok := readVarint(data, pos)
	if !ok {
		return 0, 0, false
	}
	if encodedStart == 0 {
		return -1, -1, true
	}
	start := encodedStart - 1
	count, ok := readVarint(data, pos)
	if !ok {
		return 0, 0, false
	}
	end := start + count
	if !validSpan(tokenLimit, start, end) {
		return 0, 0, false
	}
	return start, end, true
}

func validOwner(kind int, index int, declLimit int, funcLimit int) bool {
	if kind == OwnerDecl {
		return index >= 0 && index < declLimit
	}
	if kind == OwnerFunc {
		return index >= 0 && index < funcLimit
	}
	return false
}

func validSpan(tokenLimit int, start int, end int) bool {
	return start >= 0 && end >= start && end <= tokenLimit
}

func validTextSpan(textLimit int, start int, end int) bool {
	return start >= 0 && end >= start && end <= textLimit
}

func validToken(tokenLimit int, tok int) bool {
	return tok >= 0 && tok < tokenLimit
}

func validNullable(v int) bool {
	return v >= -1
}

func appendNullable(out []byte, v int) []byte {
	return appendVarint(out, v+1)
}

func readNullable(data []byte, pos *int) (int, bool) {
	value, ok := readVarint(data, pos)
	if !ok {
		return 0, false
	}
	return value - 1, true
}

func appendNode(out []byte, tag int, payload []byte) []byte {
	out = appendUint16(out, tag)
	out = appendUint32(out, len(payload))
	out = append(out, payload...)
	return out
}

func appendUint16(out []byte, v int) []byte {
	return append(out, byte(v), byte(v>>8))
}

func appendUint32(out []byte, v int) []byte {
	return append(out, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

func readUint16(data []byte, pos int) int {
	return int(data[pos]) | int(data[pos+1])<<8
}

func readUint32(data []byte, pos int) int {
	return int(data[pos]) | int(data[pos+1])<<8 | int(data[pos+2])<<16 | int(data[pos+3])<<24
}

func appendVarint(out []byte, v int) []byte {
	for v >= 0x80 {
		out = append(out, byte(v)|0x80)
		v = v >> 7
	}
	return append(out, byte(v))
}

func readVarint(data []byte, pos *int) (int, bool) {
	value := 0
	shift := 0
	for *pos < len(data) {
		b := int(data[*pos])
		*pos = *pos + 1
		if shift >= 28 && b >= 0x10 {
			return 0, false
		}
		value = value | (b&0x7f)<<shift
		if b < 0x80 {
			return value, true
		}
		shift += 7
		if shift > 28 {
			return 0, false
		}
	}
	return 0, false
}
