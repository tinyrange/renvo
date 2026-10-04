package driver

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/syntax"
)

// Resolve the companion module through the std directory itself, rather than
// its lexical parent: installed development trees may symlink only std.
type concurrencySourceFS struct {
	base       SourceFS
	stdRoot    string
	moduleRoot string
}

func (fs concurrencySourceFS) physical(path string) string {
	prefix := fs.moduleRoot
	if path == prefix {
		return fs.stdRoot + "/.."
	}
	if len(path) > len(prefix) && path[:len(prefix)] == prefix && path[len(prefix)] == '/' {
		return fs.stdRoot + "/.." + path[len(prefix):]
	}
	return path
}

func (fs concurrencySourceFS) ReadFile(path string) ([]byte, bool) {
	physical := fs.physical(path)
	data, ok := fs.base.ReadFile(physical)
	if !ok && physical != path {
		return fs.base.ReadFile(path)
	}
	return data, ok
}
func (fs concurrencySourceFS) ReadDir(path string) ([]DirEntry, bool) {
	physical := fs.physical(path)
	entries, ok := fs.base.ReadDir(physical)
	if !ok && physical != path {
		return fs.base.ReadDir(path)
	}
	return entries, ok
}
func (fs concurrencySourceFS) PathExists(path string) bool {
	physical := fs.physical(path)
	return fs.base.PathExists(physical) || physical != path && fs.base.PathExists(path)
}

// Pull the default handler into the ordinary dependency graph only for source
// that contains concurrency syntax. Comments and string literals do not count.
// No newline is inserted, preserving authored diagnostic line numbers.
func sourceConcurrencyImport(src []byte) ([]byte, bool) {
	if !sourceConcurrencyCandidate(src) {
		return src, false
	}
	mark := arena.Mark()
	tokens := syntax.Scan(src)
	needed := false
	for i := 0; i < len(tokens); i++ {
		kind := tokens[i].KindLine & 255
		if kind == syntax.TokenChan || kind == syntax.TokenGo || kind == syntax.TokenSelect {
			needed = true
		}
	}
	if !needed || len(tokens) < 2 || tokens[0].KindLine&255 != syntax.TokenPackage {
		arena.Reset(mark)
		return src, false
	}
	end := syntax.TokenEnd(tokens[1])
	arena.Reset(mark)
	out := make([]byte, 0, len(src)+55)
	out = append(out, src[:end]...)
	out = append(out, "; import _ \"renvo.dev/x/runtime/serial\";"...)
	out = append(out, src[end:]...)
	return out, true
}

// Cheap lexical filter before allocating a full token table. The scanner still
// confirms candidates, so diagnostics and keyword recognition stay centralized.
func sourceConcurrencyCandidate(src []byte) bool {
	for pos := 0; pos < len(src); pos++ {
		c := src[pos]
		if c == '/' && pos+1 < len(src) && (src[pos+1] == '/' || src[pos+1] == '*') {
			pos = renvoImportSkipSpace(src, pos) - 1
			continue
		}
		if c == '"' || c == '\'' || c == '`' {
			quote := c
			pos++
			for pos < len(src) {
				// Walk ordinary literal bytes with one range bound. Resume after an
				// escape without interpreting it; the full scanner validates syntax.
				for _, next := range src[pos:] {
					if next == quote || quote != '`' && next == '\\' {
						break
					}
					pos++
				}
				if pos >= len(src) || src[pos] == quote {
					break
				}
				pos += 2
			}
			continue
		}
		if uint(c|32)-'a' < 26 || uint(c)-'0' < 10 || c == '_' || c >= 128 {
			start := pos
			for _, next := range src[pos+1:] {
				if !(uint(next|32)-'a' < 26 || uint(next)-'0' < 10 || next == '_' || next >= 128) {
					break
				}
				pos++
			}
			// Skip whole identifiers, so keyword suffixes and long generated
			// names do not repeat comment, quote, and keyword checks per byte.
			size := pos - start + 1
			if size == 2 && c == 'g' && src[start+1] == 'o' ||
				size == 4 && c == 'c' && renvoImportTextIs(src, start, pos+1, "chan") ||
				size == 6 && c == 's' && renvoImportTextIs(src, start, pos+1, "select") {
				return true
			}
		}
	}
	return false
}
