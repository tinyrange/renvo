//go:build renvo && !wasi && !wasip1

package driver

func renvoPathCString(path string) string {
	return path + "\x00"
}
