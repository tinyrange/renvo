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

// ImportPathAt finds an import path string at caret. It supports both direct
// and grouped imports, including aliases.
func ImportPathAt(source []byte, caret int) ImportPathContext {
	if caret < 0 {
		caret = 0
	}
	if caret > len(source) {
		caret = len(source)
	}
	quote := byte(0)
	quoteAt := -1
	escaped := false
	lineComment := false
	blockComment := false
	for i := 0; i < caret; i++ {
		ch := source[i]
		next := byte(0)
		if i+1 < caret {
			next = source[i+1]
		}
		if lineComment {
			if ch == '\n' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if ch == '*' && next == '/' {
				blockComment = false
				i++
			}
			continue
		}
		if quote != 0 {
			if quote == '`' {
				if ch == '`' {
					quote = 0
					quoteAt = -1
				}
				continue
			}
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == quote {
				quote = 0
				quoteAt = -1
			}
			continue
		}
		if ch == '/' && next == '/' {
			lineComment = true
			i++
		} else if ch == '/' && next == '*' {
			blockComment = true
			i++
		} else if ch == '"' || ch == '`' {
			quote = ch
			quoteAt = i
		}
	}
	if quoteAt < 0 || !importQuoteAt(source, quoteAt) {
		return ImportPathContext{}
	}
	closed := importQuoteClosed(source, caret, quote)
	return ImportPathContext{
		Prefix:       string(source[quoteAt+1 : caret]),
		ReplaceStart: quoteAt + 1,
		Quote:        quote,
		Closed:       closed,
		Ok:           true,
	}
}

func importQuoteAt(source []byte, quoteAt int) bool {
	tokens := syntax.Scan(source[:quoteAt])
	var indexes []int
	for i := 0; i < len(tokens); i++ {
		if tokens[i].KindLine&255 != syntax.TokenEOF {
			indexes = append(indexes, i)
		}
	}
	if len(indexes) == 0 {
		return false
	}
	last := indexes[len(indexes)-1]
	if tokens[last].KindLine&255 == syntax.TokenImport {
		return true
	}
	if len(indexes) >= 2 && tokens[indexes[len(indexes)-2]].KindLine&255 == syntax.TokenImport {
		return true
	}
	depth := 0
	for i := len(indexes) - 1; i >= 0; i-- {
		at := indexes[i]
		text := syntax.TokenText(source, tokens[at])
		if len(text) != 1 {
			continue
		}
		if text[0] == ')' {
			depth++
			continue
		}
		if text[0] != '(' {
			continue
		}
		if depth > 0 {
			depth--
			continue
		}
		return i > 0 && tokens[indexes[i-1]].KindLine&255 == syntax.TokenImport
	}
	return false
}

func importQuoteClosed(source []byte, caret int, quote byte) bool {
	escaped := false
	for i := caret; i < len(source); i++ {
		ch := source[i]
		if quote != '`' && ch == '\n' {
			return false
		}
		if quote != '`' && escaped {
			escaped = false
			continue
		}
		if quote != '`' && ch == '\\' {
			escaped = true
			continue
		}
		if ch == quote {
			return true
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
