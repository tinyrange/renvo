//go:build !renvo

package unsafe

import stdunsafe "unsafe"

type Pointer = stdunsafe.Pointer

func Sizeof[T any](x T) uintptr {
	return stdunsafe.Sizeof(x)
}

func Alignof[T any](x T) uintptr {
	return stdunsafe.Alignof(x)
}

// Offsetof is a compiler intrinsic in Go. This source-level shim exists so the
// package has the symbol during host-side std tests, but it cannot reproduce the
// real selector-expression semantics.
func Offsetof(x uintptr) uintptr {
	return x
}
