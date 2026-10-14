//go:build renvo || !linux || !amd64 || !cgo

package systemmemory

func newDirectRAM(size int) ([]byte, directMemory, error) {
	return make([]byte, size), nil, nil
}
