//go:build rtg && (wasi || wasip1)

package os

// WASI path_open receives an explicit string length rather than a C string.
func rtgPathCString(path string) string { return path }
