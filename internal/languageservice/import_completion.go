package languageservice

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

// ImportPathContext describes an import string containing the caret.
type ImportPathContext struct {
	Prefix       string
	ReplaceStart int
	Quote        byte
	Closed       bool
	Ok           bool
}

// ParsedImport is an import declaration accepted by the Renvo Go parser.
type ParsedImport struct {
	Name string
	Path string
}

// SelectorContext describes a package-style selector at the caret.
type SelectorContext struct {
	Base         string
	Prefix       string
	ReplaceStart int
	Ok           bool
}

// ParseImports returns imports from the frontend parse, including imports
// retained before a later syntax error in an actively edited file.
func ParseImports(source []byte) []ParsedImport {
	file := syntax.ParseFile(source)
	imports := make([]ParsedImport, 0, len(file.Imports))
	for i := 0; i < len(file.Imports); i++ {
		declaration := file.Imports[i]
		if declaration.PathTok < 0 || declaration.PathTok >= len(file.Tokens) {
			continue
		}
		path, ok := syntax.StringLiteralValue(file.Src, file.Tokens[declaration.PathTok])
		if !ok {
			continue
		}
		name := ""
		if declaration.NameTok >= 0 && declaration.NameTok < len(file.Tokens) {
			name = string(syntax.TokenText(file.Src, file.Tokens[declaration.NameTok]))
		}
		imports = append(imports, ParsedImport{Name: name, Path: path})
	}
	return imports
}

// SelectorAt returns selector context from the frontend token stream. It is
// intentionally usable while the surrounding expression is incomplete.
func SelectorAt(source []byte, caret int) SelectorContext {
	if caret < 0 {
		caret = 0
	}
	if caret > len(source) {
		caret = len(source)
	}
	tokens := syntax.Scan(source[:caret])
	last := len(tokens) - 1
	for last >= 0 && tokens[last].KindLine&255 == syntax.TokenEOF {
		last--
	}
	if last < 1 {
		return SelectorContext{}
	}
	prefix := ""
	replaceStart := caret
	if tokens[last].KindLine&255 == syntax.TokenIdent && syntax.TokenEnd(tokens[last]) == caret {
		prefix = string(syntax.TokenText(source, tokens[last]))
		replaceStart = syntax.TokenStart(tokens[last])
		last--
	}
	if last < 1 || string(syntax.TokenText(source, tokens[last])) != "." || syntax.TokenEnd(tokens[last]) != replaceStart {
		return SelectorContext{}
	}
	baseToken := tokens[last-1]
	if baseToken.KindLine&255 != syntax.TokenIdent || syntax.TokenEnd(baseToken) != syntax.TokenStart(tokens[last]) {
		return SelectorContext{}
	}
	return SelectorContext{Base: string(syntax.TokenText(source, baseToken)), Prefix: prefix, ReplaceStart: replaceStart, Ok: true}
}

// ImportPathAt finds an import path string at caret. It supports both direct
// and grouped imports, including aliases.
func ImportPathAt(source []byte, caret int) ImportPathContext {
	if caret < 0 {
		caret = 0
	}
	if caret > len(source) {
		caret = len(source)
	}
	quote, quoteAt, ok := parsedImportQuoteAt(source, caret)
	if !ok {
		return ImportPathContext{}
	}
	closed := parsedImportQuoteClosed(source, quoteAt, caret)
	return ImportPathContext{
		Prefix:       string(source[quoteAt+1 : caret]),
		ReplaceStart: quoteAt + 1,
		Quote:        quote,
		Closed:       closed,
		Ok:           true,
	}
}

func parsedImportQuoteAt(source []byte, caret int) (byte, int, bool) {
	for _, quote := range []byte{'"', '`'} {
		for _, prefix := range [][]byte{nil, []byte("package repl\n")} {
			probe := make([]byte, 0, len(prefix)+caret+4)
			probe = append(probe, prefix...)
			probe = append(probe, source[:caret]...)
			probe = append(probe, quote, '\n', ')', '\n')
			file := syntax.ParseFile(probe)
			for i := 0; i < len(file.Imports); i++ {
				pathTok := file.Imports[i].PathTok
				if pathTok < 0 || pathTok >= len(file.Tokens) {
					continue
				}
				token := file.Tokens[pathTok]
				if syntax.TokenEnd(token) == len(prefix)+caret+1 && probe[syntax.TokenStart(token)] == quote {
					return quote, syntax.TokenStart(token) - len(prefix), true
				}
			}
		}
	}
	return 0, -1, false
}

func parsedImportQuoteClosed(source []byte, quoteAt int, caret int) bool {
	tokens := syntax.Scan(source)
	for i := 0; i < len(tokens); i++ {
		if tokens[i].KindLine&255 == syntax.TokenString && syntax.TokenStart(tokens[i]) == quoteAt {
			return syntax.TokenEnd(tokens[i]) > caret
		}
	}
	return false
}

// CompleteStandardImportPaths returns target-enabled packages present in the
// configured standard-library tree.
func CompleteStandardImportPaths(stdRoot string, target string, tags []string, prefix string, fs driver.SourceFS) []string {
	stdRoot = load.CleanPath(stdRoot)
	var out []string
	completeImportDirectory(stdRoot, "", target, tags, prefix, fs, &out)
	sortImportPaths(out)
	return out
}

func completeImportDirectory(dir string, importPath string, target string, tags []string, prefix string, fs driver.SourceFS, out *[]string) {
	entries, ok := fs.ReadDir(dir)
	if !ok {
		return
	}
	sortDirEntries(entries)
	packageEnabled := false
	for i := 0; i < len(entries); i++ {
		if entries[i].IsDir {
			continue
		}
		mark := arena.Mark()
		source, readOK := fs.ReadFile(load.JoinPath(dir, entries[i].Name))
		enabled := false
		if readOK {
			enabled, _ = driver.SourceFileEnabled(entries[i].Name, source, target, tags)
		}
		arena.Discard(mark, arena.Mark())
		if enabled {
			packageEnabled = true
			break
		}
	}
	if packageEnabled && importPath != "" && importPathHasPrefix(importPath, prefix) {
		*out = append(*out, importPath)
	}
	for i := 0; i < len(entries); i++ {
		if !entries[i].IsDir || entries[i].Name == "" || entries[i].Name[0] == '.' ||
			entries[i].Name[0] == '_' {
			continue
		}
		childPath := entries[i].Name
		if importPath != "" {
			childPath = importPath + "/" + entries[i].Name
		}
		completeImportDirectory(load.JoinPath(dir, entries[i].Name), childPath, target, tags, prefix, fs, out)
	}
}

func sortDirEntries(entries []driver.DirEntry) {
	for i := 1; i < len(entries); i++ {
		item := entries[i]
		j := i - 1
		for j >= 0 && entries[j].Name > item.Name {
			entries[j+1] = entries[j]
			j--
		}
		entries[j+1] = item
	}
}

func importPathHasPrefix(value string, prefix string) bool {
	if len(prefix) > len(value) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if value[i] != prefix[i] {
			return false
		}
	}
	return true
}

func sortImportPaths(paths []string) {
	for i := 1; i < len(paths); i++ {
		item := paths[i]
		j := i - 1
		for j >= 0 && importPathAfter(paths[j], item) {
			paths[j+1] = paths[j]
			j--
		}
		paths[j+1] = item
	}
}

func importPathAfter(left string, right string) bool {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for i := 0; i < limit; i++ {
		if left[i] != right[i] {
			return left[i] > right[i]
		}
	}
	return len(left) > len(right)
}
