//go:build rtg && !wasi && !wasip1

package os

// Native open APIs consume a NUL-terminated path. Go strings forwarded from a
// caller are not necessarily followed by a zero byte, so always own the
// terminator at the runtime boundary.
func rtgPathCString(path string) string {
	out := make([]byte, 0, len(path)+1)
	for i := 0; i < len(path); i++ {
		out = append(out, path[i])
	}
	out = append(out, 0)
	return string(out)
}
