//go:build rtg && (wasi || wasip1)

package driver

func rtgPathCString(path string) string {
	return path
}
