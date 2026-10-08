//go:build !renvo && (amd64 || arm64)

package runimage

//go:noescape
func callPure(entry, state, stackTop uintptr)

//go:noescape
func callContext(entry, state, context, stackTop uintptr)
