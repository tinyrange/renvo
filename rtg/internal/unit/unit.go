package unit

const (
	Magic   = "RTGU"
	Version = 1
)

const (
	TagUnit    = 1
	TagPackage = 2
	TagText    = 7
	TagTokens  = 8
	TagDecls   = 9
	TagFuncs   = 10
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

type Program struct {
	Package string
	Text    []byte
	Tokens  []Token
	Decls   []Decl
	Funcs   []Func
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
	var root []byte
	root = appendNode(root, TagPackage, []byte(program.Package))
	root = appendNode(root, TagText, program.Text)
	root = appendNode(root, TagTokens, tokenData)
	root = appendNode(root, TagDecls, declData)
	root = appendNode(root, TagFuncs, funcData)

	out := make([]byte, 0, 14+len(root))
	out = append(out, 'R', 'T', 'G', 'U')
	out = appendUint16(out, Version)
	out = appendUint16(out, 0)
	out = appendNode(out, TagUnit, root)
	return out, true
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

func appendVarint(out []byte, v int) []byte {
	for v >= 0x80 {
		out = append(out, byte(v)|0x80)
		v = v >> 7
	}
	return append(out, byte(v))
}
