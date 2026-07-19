//go:build renvo && !wasi && !wasip1

package os

// Native open APIs consume a NUL-terminated path. Go strings forwarded from a
// caller are not necessarily followed by a zero byte, so always own the
// terminator at the runtime boundary.
func renvoPathCString(path string) string {
	return path + "\x00"
}
