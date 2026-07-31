// Package rtg parses the shared Renvo target-generation language.
//
// The package intentionally owns no filesystem, process, cache, or compiler
// policy. Callers provide source byte slices (and, for imported definitions, a
// narrow loader capability) and receive a closed document plus structured
// diagnostics.
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
	Value      string
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
	Package    string
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
	sourceMap    []sourceSegment
	packages     []virtualPackage
}

// ImportLoader resolves an import path relative to the importing source. The
// returned filename is the stable name used for diagnostics and cycle checks.
// Filesystem-backed callers normally return a cleaned absolute path; virtual
// filesystems can return their own canonical path.
type ImportLoader interface {
	LoadImport(importingFilename string, importPath string) ImportSource
}

type ImportSource struct {
	Source   []byte
	Filename string
	Ok       bool
}

type sourceSegment struct {
	logicalStart int
	logicalEnd   int
	sourceStart  int
	filename     string
	source       []byte
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

func documentDiagnostic(document Document, span Span, code string, message string) Diagnostic {
	filename, start := documentPosition(document, span.Start.Offset)
	endFilename, end := documentPosition(document, span.End.Offset)
	if endFilename != filename {
		end = start
	}
	return Diagnostic{
		Filename: filename,
		Span:     Span{Start: start, End: end},
		Code:     code,
		Message:  message,
	}
}

func documentPosition(document Document, offset int) (string, Position) {
	if len(document.sourceMap) == 0 {
		return document.Filename, position(document.Source, offset)
	}
	for i := 0; i < len(document.sourceMap); i++ {
		segment := document.sourceMap[i]
		if offset >= segment.logicalStart && offset < segment.logicalEnd {
			sourceOffset := segment.sourceStart + offset - segment.logicalStart
			return segment.filename, position(segment.source, sourceOffset)
		}
	}
	if offset == len(document.Source) {
		for i := len(document.sourceMap) - 1; i >= 0; i-- {
			segment := document.sourceMap[i]
			if segment.logicalEnd == offset {
				return segment.filename, position(segment.source,
					segment.sourceStart+segment.logicalEnd-segment.logicalStart)
			}
		}
	}
	return document.Filename, position(document.Source, offset)
}
