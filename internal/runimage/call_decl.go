//go:build !renvo && ((linux && (amd64 || 386 || arm64 || arm)) || (windows && (amd64 || 386 || arm64)) || (darwin && arm64))

package runimage

// callJIT switches to stackTop, calls the linked-image entry using its
// four-word slice ABI, restores the Go registers and stack, and returns the
// entry result.
func callJIT(entry, stackTop, argsData, argsLen, envData, envLen uintptr) int
