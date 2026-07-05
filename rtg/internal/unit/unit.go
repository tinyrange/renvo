//go:build rtg

package unit

const (
	Magic   = "RTGU"
	Version = 1
)

const (
	TagUnit       = 1
	TagPackage    = 2
	TagImportPath = 3
	TagText       = 7
	TagTokens     = 8
	TagDecls      = 9
	TagFuncs      = 10
	TagIndexes    = 11
	TagComps      = 12
	TagAssigns    = 13
	TagReturns    = 14
	TagCalls      = 15
	TagRefs       = 16
	TagSels       = 17
	TagTypes      = 18
	TagTypeRefs   = 19
	TagLocals     = 20
	TagSigs       = 21
	TagDeclMeta   = 22
	TagImports    = 23
	TagSymbols    = 24
	TagInitOrder  = 25
	TagConsts     = 26
	TagTypeFields = 27
	TagTypeIfaces = 28
	TagMethods    = 29
	TagTypeFuncs  = 30
	TagStmts      = 31
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

type Import struct {
	Name       string
	ImportPath string
	Package    int
	NameTok    int
	PathTok    int
	Dot        bool
	Blank      bool
}

const (
	SymbolConst = iota + 1
	SymbolVar
	SymbolType
	SymbolFunc
	SymbolMethod
)

type Symbol struct {
	Name       string
	Kind       int
	Package    int
	Token      int
	OwnerKind  int
	OwnerIndex int
}

type DeclMeta struct {
	DeclIndex  int
	Symbol     int
	ValueIndex int
	TypeStart  int
	TypeEnd    int
	ValueStart int
	ValueEnd   int
	Values     []ExprSpan
	Alias      bool
}

const (
	ConstInt = iota + 1
	ConstString
	ConstBool
)

type ConstValue struct {
	DeclIndex int
	Kind      int
	Int       int
	String    string
	Bool      bool
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

type Field struct {
	NameTok   int
	TypeStart int
	TypeEnd   int
	Variadic  bool
}

type FuncSignature struct {
	FuncIndex int
	Receiver  []Field
	Params    []Field
	Results   []Field
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

type TypeFields struct {
	TypeIndex int
	Fields    []Field
}

type InterfaceMethod struct {
	NameTok int
	Params  []Field
	Results []Field
}

type InterfaceEmbed struct {
	TypeStart int
	TypeEnd   int
}

type TypeIface struct {
	TypeIndex int
	Methods   []InterfaceMethod
	Embeds    []InterfaceEmbed
}

type TypeFuncSig struct {
	TypeIndex int
	Params    []Field
	Results   []Field
}

type MethodInfo struct {
	NameTok   int
	TypeIndex int
	Symbol    int
	FuncIndex int
	Pointer   bool
}

const (
	OwnerDecl = iota + 1
	OwnerFunc
)

type ExprSpan struct {
	StartTok int
	EndTok   int
}

const (
	StmtOther = iota
	StmtReturn
	StmtIf
	StmtFor
	StmtSwitch
	StmtCase
	StmtDefault
	StmtDecl
	StmtAssign
	StmtExpr
	StmtBlock
	StmtBreak
	StmtContinue
	StmtGoto
	StmtDefer
	StmtGo
	StmtFallthrough
	StmtLabel
)

type Statement struct {
	FuncIndex int
	Kind      int
	StartTok  int
	EndTok    int
	ExprStart int
	ExprEnd   int
	BodyStart int
	BodyEnd   int
	ElseStart int
	ElseEnd   int
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

type LocalDecl struct {
	FuncIndex  int
	Kind       int
	NameStart  int
	NameEnd    int
	Token      int
	Scope      int
	ValueIndex int
	TypeStart  int
	TypeEnd    int
	ValueStart int
	ValueEnd   int
	Values     []ExprSpan
	Alias      bool
}

type Program struct {
	Package    string
	ImportPath string
	Text       []byte
	Tokens     []Token
	Imports    []Import
	Symbols    []Symbol
	Decls      []Decl
	DeclMeta   []DeclMeta
	InitOrder  []int
	Consts     []ConstValue
	Funcs      []Func
	Signatures []FuncSignature
	Stmts      []Statement
	Types      []TypeInfo
	TypeFields []TypeFields
	TypeIfaces []TypeIface
	TypeFuncs  []TypeFuncSig
	Methods    []MethodInfo
	TypeRefs   []TypeRef
	Locals     []LocalDecl
	Indexes    []IndexExpr
	Composites []CompositeExpr
	Assigns    []Assignment
	Returns    []Return
	Calls      []Call
	Refs       []NameRef
	Selectors  []Selector
}

var LastMarshalError int
var LastMarshalIndex int
var LastMarshalDetail int
var LastMarshalA int
var LastMarshalB int
var LastMarshalC int

func Marshal(program Program) ([]byte, bool) {
	LastMarshalError = 0
	LastMarshalIndex = -1
	LastMarshalDetail = 0
	LastMarshalA = 0
	LastMarshalB = 0
	LastMarshalC = 0
	if len(program.Package) == 0 || len(program.Text) == 0 || len(program.Tokens) == 0 {
		LastMarshalError = 1
		return nil, false
	}
	tokenData, ok := encodeTokens(program.Text, program.Tokens)
	if !ok {
		LastMarshalError = 2
		return nil, false
	}
	declData, ok := encodeDecls(program.Decls)
	if !ok {
		LastMarshalError = 5
		return nil, false
	}
	funcData, ok := encodeFuncs(program.Funcs)
	if !ok {
		LastMarshalError = 9
		return nil, false
	}
	rootLen := 0
	rootLen += nodeSizeString(program.Package)
	rootLen += nodeSizeString(program.ImportPath)
	rootLen += nodeSize(program.Text)
	rootLen += nodeSize(tokenData)
	rootLen += nodeSize(declData)
	rootLen += nodeSize(funcData)

	out := make([]byte, 0, 14+rootLen)
	out = append(out, 'R')
	out = append(out, 'T')
	out = append(out, 'G')
	out = append(out, 'U')
	out = appendUint16(out, Version)
	out = appendUint16(out, 0)
	out = appendUint16(out, TagUnit)
	out = appendUint32(out, rootLen)
	out = appendNode(out, TagPackage, []byte(program.Package))
	out = appendNode(out, TagImportPath, []byte(program.ImportPath))
	out = appendNode(out, TagText, program.Text)
	out = appendNode(out, TagTokens, tokenData)
	out = appendNode(out, TagDecls, declData)
	out = appendNode(out, TagFuncs, funcData)
	return out, true
}

func Unmarshal(data []byte) (Program, bool) {
	return Program{}, false
}

func encodeTokens(text []byte, tokens []Token) ([]byte, bool) {
	out := make([]byte, 0, len(tokens)*4)
	out = appendVarint(out, len(tokens))
	prevStart := 0
	prevLine := 0
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok.Kind < 0 || tok.Kind > 255 || tok.Start < prevStart || tok.Line < prevLine || tok.Size < 0 {
			LastMarshalIndex = i
			LastMarshalDetail = 1
			return nil, false
		}
		if tok.Start > len(text) || tok.Start+tok.Size > len(text) {
			LastMarshalIndex = i
			LastMarshalDetail = 2
			LastMarshalA = tok.Start
			LastMarshalB = tok.Size
			LastMarshalC = len(text)
			return nil, false
		}
		if tok.Start > 0xffffff || tok.Line > 0xffff {
			LastMarshalIndex = i
			LastMarshalDetail = 3
			return nil, false
		}
		if tok.Kind == TokenOp {
			if tok.Size > 255 {
				LastMarshalIndex = i
				LastMarshalDetail = 4
				return nil, false
			}
		} else if tok.Size > 0xffff {
			LastMarshalIndex = i
			LastMarshalDetail = 5
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
	for i := 0; i < len(payload); i++ {
		out = append(out, payload[i])
	}
	return out
}

func nodeSize(payload []byte) int {
	return 6 + len(payload)
}

func nodeSizeString(payload string) int {
	return 6 + len(payload)
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
