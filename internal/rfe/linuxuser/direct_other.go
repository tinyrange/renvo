//go:build renvo || !linux || !amd64 || !cgo

package linuxuser

func newDirectMemory() (directMemory, error) { return nil, directUnavailable() }
