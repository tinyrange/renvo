//go:build renvo && (wasi || wasip1)

package os

// WASI path_open receives an explicit string length rather than a C string.
func renvoPathCString(path string) string { return path }
