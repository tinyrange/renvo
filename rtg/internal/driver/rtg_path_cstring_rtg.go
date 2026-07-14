//go:build rtg && !wasi && !wasip1

package driver

func rtgPathCString(path string) string {
	var out []byte
	for i := 0; i < len(path); i++ {
		out = append(out, path[i])
	}
	out = append(out, 0)
	return string(out)
}
