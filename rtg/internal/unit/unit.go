//go:build rtg

package unit

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

type Import struct {
	NameTok int
	PathTok int
}

type Symbol struct {
	Name    string
	Package int
	Token   int
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
	CallUnknown = iota
	CallScope
	CallPackage
	CallImportSelector
	CallBuiltin
)

type Call struct {
	Kind      int
	CalleeTok int
	BaseTok   int
	DotTok    int
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
	Kind    int
	Token   int
	Index   int
	Package int
}

type Selector struct {
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
	Kind    int
	Token   int
	BaseTok int
	DotTok  int
	Package int
	Symbol  int
}

type Program struct {
	Package    string
	ImportPath string
	Text       []byte
	Tokens     []Token
	Imports    []Import
	Symbols    []Symbol
	Decls      []Decl
	Funcs      []Func
	TypeRefs   []TypeRef
	Calls      []Call
	Refs       []NameRef
	Selectors  []Selector
}

func Marshal(program Program) ([]byte, bool) {
	tokenLen := encodedTokensLen(program.Tokens)
	declLen := encodedDeclsLen(program.Decls)
	funcLen := encodedFuncsLen(program.Funcs)
	rootLen := 36 + len(program.Package) + len(program.ImportPath) + len(program.Text) + tokenLen + declLen + funcLen

	out := make([]byte, 0, 14+rootLen)
	for i := 0; i < len(Magic); i++ {
		out = append(out, Magic[i])
	}
	out = appendUint16(out, Version)
	out = appendUint16(out, 0)
	out = appendUint16(out, TagUnit)
	out = appendUint32(out, rootLen)
	out = appendStringNode(out, TagPackage, program.Package)
	out = appendStringNode(out, TagImportPath, program.ImportPath)
	out = appendNode(out, TagText, program.Text)
	out = appendNodeHeader(out, TagTokens, tokenLen)
	out = appendEncodedTokens(out, program.Tokens)
	out = appendNodeHeader(out, TagDecls, declLen)
	out = appendEncodedDecls(out, program.Decls)
	out = appendNodeHeader(out, TagFuncs, funcLen)
	out = appendEncodedFuncs(out, program.Funcs)
	return out, true
}

func encodedTokensLen(tokens []Token) int {
	size := varintLen(len(tokens))
	prevStart := 0
	prevLine := 0
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		size += varintLen(tok.Kind)
		size += varintLen(tok.Start - prevStart)
		size += varintLen(tok.Size)
		size += varintLen(tok.Line - prevLine)
		prevStart = tok.Start
		prevLine = tok.Line
	}
	return size
}

func appendEncodedTokens(out []byte, tokens []Token) []byte {
	out = appendVarint(out, len(tokens))
	prevStart := 0
	prevLine := 0
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		out = appendVarint(out, tok.Kind)
		out = appendVarint(out, tok.Start-prevStart)
		out = appendVarint(out, tok.Size)
		out = appendVarint(out, tok.Line-prevLine)
		prevStart = tok.Start
		prevLine = tok.Line
	}
	return out
}

func encodedDeclsLen(decls []Decl) int {
	size := varintLen(len(decls))
	for i := 0; i < len(decls); i++ {
		decl := decls[i]
		size += varintLen(decl.Kind)
		size += varintLen(decl.NameStart)
		size += varintLen(decl.NameEnd - decl.NameStart)
		size += varintLen(decl.StartTok)
		size += varintLen(decl.EndTok - decl.StartTok)
	}
	return size
}

func appendEncodedDecls(out []byte, decls []Decl) []byte {
	out = appendVarint(out, len(decls))
	for i := 0; i < len(decls); i++ {
		decl := decls[i]
		out = appendVarint(out, decl.Kind)
		out = appendVarint(out, decl.NameStart)
		out = appendVarint(out, decl.NameEnd-decl.NameStart)
		out = appendVarint(out, decl.StartTok)
		out = appendVarint(out, decl.EndTok-decl.StartTok)
	}
	return out
}

func encodedFuncsLen(funcs []Func) int {
	size := varintLen(len(funcs))
	for i := 0; i < len(funcs); i++ {
		fn := funcs[i]
		size += varintLen(fn.NameStart)
		size += varintLen(fn.NameEnd - fn.NameStart)
		size += varintLen(fn.StartTok)
		size += varintLen(fn.NameTok - fn.StartTok)
		size += varintLen(fn.ReceiverStart)
		size += varintLen(fn.ReceiverEnd - fn.ReceiverStart)
		size += varintLen(fn.BodyStart)
		size += varintLen(fn.BodyEnd - fn.BodyStart)
		size += varintLen(fn.EndTok - fn.BodyEnd)
	}
	return size
}

func appendEncodedFuncs(out []byte, funcs []Func) []byte {
	out = appendVarint(out, len(funcs))
	for i := 0; i < len(funcs); i++ {
		fn := funcs[i]
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
	return out
}

func appendStringNode(out []byte, tag int, payload string) []byte {
	out = appendNodeHeader(out, tag, len(payload))
	for i := 0; i < len(payload); i++ {
		out = append(out, payload[i])
	}
	return out
}

func appendNode(out []byte, tag int, payload []byte) []byte {
	out = appendNodeHeader(out, tag, len(payload))
	for i := 0; i < len(payload); i++ {
		out = append(out, payload[i])
	}
	return out
}

func appendNodeHeader(out []byte, tag int, size int) []byte {
	out = appendUint16(out, tag)
	out = appendUint32(out, size)
	return out
}

func appendUint16(out []byte, v int) []byte {
	out = append(out, byte(v))
	out = append(out, byte(v>>8))
	return out
}

func appendUint32(out []byte, v int) []byte {
	out = append(out, byte(v))
	out = append(out, byte(v>>8))
	out = append(out, byte(v>>16))
	out = append(out, byte(v>>24))
	return out
}

func appendVarint(out []byte, v int) []byte {
	for v >= 0x80 {
		out = append(out, byte(v)|0x80)
		v = v >> 7
	}
	return append(out, byte(v))
}

func varintLen(v int) int {
	size := 1
	for v >= 0x80 {
		size++
		v = v >> 7
	}
	return size
}
