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
	// KindLine packs the token kind into the low byte and its source line above it.
	KindLine int
	Start    int
	Size     int
}

func MakeToken(kind int, start int, size int, line int) Token {
	return Token{KindLine: kind | line<<8, Start: start, Size: size}
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

type Import struct {
	NameTok int
	PathTok int
}

type Symbol struct {
	Name    string
	Package int
	Token   int
}

type Call struct {
	Kind      int
	CalleeTok int
	BaseTok   int
	DotTok    int
}

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

type TypeRef struct {
	Kind    int
	Token   int
	BaseTok int
	DotTok  int
	Package int
	Symbol  int
}

// ConcurrencySite carries checker-owned facts needed by whole-program
// goroutine/channel lowering. It is an internal frontend/cache contract and is
// deliberately omitted from CoreProgram after lowering.
type ConcurrencySite struct {
	Kind         int
	Token        int
	Direction    int
	ReceiveArity int
	ElementType  string
}

// PackageInfo preserves the independently compiled package that owns a
// contiguous part of a linked program. Keys are frontend-computed identities:
// GraphKey includes transitive dependencies while SourceKey covers this
// package's files. A package may have more than one fragment when whole-program
// lowering appends generated helpers.
type PackageInfo struct {
	Name       string
	ImportPath string
	GraphKeyA  int
	GraphKeyB  int
	SourceKeyA int
	SourceKeyB int
	TextStart  int
	TextEnd    int
	TokenStart int
	TokenEnd   int
	DeclStart  int
	DeclEnd    int
	FuncStart  int
	FuncEnd    int
}

// RTGAssemblySource preserves one selected project assembly file verbatim.
// Path is package-relative and stable across relocating a checkout.
type RTGAssemblySource struct {
	Path   string
	Source []byte
}

// RTGAssemblyBinding attaches one entry ordinal in a source file to a function
// index. Package units use package-local indexes; the linker remaps them to the
// final linked function table before serialization.
type RTGAssemblyBinding struct {
	Func   int
	Source int
	Entry  int
	Code   []byte
}

const (
	ForeignProgramBytes = iota + 1
	ForeignProgramEntrypoint
)

// ForeignProgram binds one package variable to a separately targeted unit.
// Frontends fill Unit, orchestration replaces it with Artifact, and the outer
// backend consumes only the resolved artifact and entry offset.
type ForeignProgram struct {
	Name        string
	Kind        int
	Target      string
	InPlace     bool
	Unit        []byte
	Artifact    []byte
	EntryOffset int
}

// Program is the shared lowering and linking model. Checker-only semantic
// tables stay outside this boundary.
type Program struct {
	Package          string
	ImportPath       string
	Text             []byte
	Tokens           []Token
	Imports          []Import
	Symbols          []Symbol
	Decls            []Decl
	Funcs            []Func
	TypeRefs         []TypeRef
	Calls            []Call
	Refs             []NameRef
	Selectors        []Selector
	ConcurrencySites []ConcurrencySite
	Packages         []PackageInfo
	RTGAssembly      []RTGAssemblySource
	RTGAssemblyFuncs []RTGAssemblyBinding
	// Entrypoint is encoded as function index + 1; zero retains the ordinary
	// package-main entry selection used by existing units.
	Entrypoint      int
	ForeignPrograms []ForeignProgram
}

// CoreProgram is the complete serialized contract consumed by compiler
// backends. Link-only resolution tables are deliberately absent.
type CoreProgram struct {
	Package          string
	ImportPath       string
	Text             []byte
	Tokens           []Token
	Decls            []Decl
	Funcs            []Func
	Packages         []PackageInfo
	RTGAssembly      []RTGAssemblySource
	RTGAssemblyFuncs []RTGAssemblyBinding
	Entrypoint       int
	ForeignPrograms  []ForeignProgram
}

func CoreProgramFrom(program Program) CoreProgram {
	return CoreProgram{
		Package:          program.Package,
		ImportPath:       program.ImportPath,
		Text:             program.Text,
		Tokens:           program.Tokens,
		Decls:            program.Decls,
		Funcs:            program.Funcs,
		Packages:         program.Packages,
		RTGAssembly:      program.RTGAssembly,
		RTGAssemblyFuncs: program.RTGAssemblyFuncs,
		Entrypoint:       program.Entrypoint,
		ForeignPrograms:  program.ForeignPrograms,
	}
}

const (
	CallUnknown = iota
	CallScope
	CallPackage
	CallImportSelector
	CallBuiltin
)

const (
	ConcurrencyGo = iota + 1
	ConcurrencyChannelType
	ConcurrencySend
	ConcurrencyReceive
	ConcurrencySelect
)

const (
	RefUnknown = iota
	RefScope
	RefPackage
	RefImport
	RefBuiltin
	RefLabel
)

const (
	TypeRefUnknown = iota
	TypeRefScope
	TypeRefPackage
	TypeRefImportSelector
	TypeRefBuiltin
)

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
