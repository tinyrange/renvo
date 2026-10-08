//go:build !renvo && (!cgo || !linux || !amd64)

package rfenativebridge

import "unsafe"

const Available = false

func Call(entry uintptr, state, context unsafe.Pointer, top uintptr) {
	panic("RFE foreign execution boundary is unavailable")
}
