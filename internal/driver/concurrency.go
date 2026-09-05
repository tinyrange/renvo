package driver

import "renvo.dev/internal/syntax"

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
	return fs.base.ReadFile(fs.physical(path))
}
func (fs concurrencySourceFS) ReadDir(path string) ([]DirEntry, bool) {
	return fs.base.ReadDir(fs.physical(path))
}
func (fs concurrencySourceFS) PathExists(path string) bool {
	return fs.base.PathExists(fs.physical(path))
}

// Pull the default handler into the ordinary dependency graph only for source
// that contains concurrency syntax. Comments and string literals do not count.
// No newline is inserted, preserving authored diagnostic line numbers.
func sourceConcurrencyImport(src []byte) ([]byte, bool) {
	if sourceTextOffset(src, "chan") == 0 && sourceTextOffset(src, "go") == 0 && sourceTextOffset(src, "select") == 0 {
		return src, false
	}
	tokens := syntax.Scan(src)
	needed := false
	for i := 0; i < len(tokens); i++ {
		kind := tokens[i].KindLine & 255
		if kind == syntax.TokenChan || kind == syntax.TokenGo || kind == syntax.TokenSelect {
			needed = true
		}
	}
	if !needed || len(tokens) < 2 || tokens[0].KindLine&255 != syntax.TokenPackage {
		return src, false
	}
	end := syntax.TokenEnd(tokens[1])
	out := make([]byte, 0, len(src)+55)
	out = append(out, src[:end]...)
	out = append(out, "; import _ \"renvo.dev/x/runtime/serial\";"...)
	out = append(out, src[end:]...)
	return out, true
}
