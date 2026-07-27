// Package rtg parses the shared Renvo target-generation language.
//
// The package intentionally owns no filesystem, process, cache, or compiler
// policy. Callers provide one source byte slice and receive a closed document
// plus structured diagnostics.
package rtg

const (
	TokenInvalid = iota
	TokenEOF
	TokenIdent
	TokenNumber
	TokenString
	TokenOperator
)

const (
	DeclSystem  = "system"
	DeclArch    = "arch"
	DeclABI     = "abi"
	DeclRuntime = "runtime"
	DeclFormat  = "format"
	DeclTarget  = "target"
	DeclIR      = "ir"
	DeclGo      = "go"
)

type Position struct {
	Offset int
	Line   int
	Column int
}

type Span struct {
	Start Position
	End   Position
}

type Diagnostic struct {
	Filename string
	Span     Span
	Code     string
	Message  string
}

type Token struct {
	Kind   int
	Start  int
	End    int
	Line   int
	Column int
}

type Field struct {
	Name       string
	ValueStart int
	ValueEnd   int
	Span       Span
}

// Statement is a syntax-preserving declaration-body statement. Tokens contains
// the statement header without a trailing nested block. Children is non-empty
// for constructs such as "instructions { ... }" and "relocation rel32 { ... }".
// Keeping the token spelling here lets semantic decoders stay small and typed
// without turning the definition language into a second general-purpose AST.
type Statement struct {
	Tokens   []string
	Children []Statement
	Span     Span
}

type Declaration struct {
	Kind       string
	Name       string
	Start      int
	End        int
	BodyStart  int
	BodyEnd    int
	Fields     []Field
	Statements []Statement
	GoSource   []byte
	Span       Span
}

type Document struct {
	Filename     string
	Source       []byte
	Version      int
	Unit         string
	Implements   []string
	Declarations []Declaration
	Tokens       []Token
	Diagnostics  []Diagnostic
	Hash         [32]byte
	Ok           bool
}

func (d Document) Declaration(kind string, name string) (Declaration, bool) {
	for i := 0; i < len(d.Declarations); i++ {
		if d.Declarations[i].Kind == kind && d.Declarations[i].Name == name {
			return d.Declarations[i], true
		}
	}
	return Declaration{}, false
}

func tokenText(source []byte, token Token) string {
	if token.Start < 0 || token.End < token.Start || token.End > len(source) {
		return ""
	}
	return string(source[token.Start:token.End])
}

func position(source []byte, offset int) Position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(source) {
		offset = len(source)
	}
	line := 1
	column := 1
	for i := 0; i < offset; i++ {
		if source[i] == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	return Position{Offset: offset, Line: line, Column: column}
}

func sourceSpan(source []byte, start int, end int) Span {
	return Span{Start: position(source, start), End: position(source, end)}
}
