//go:build !renvo && (!cgo || !linux || !amd64)

package rfenativebridge

import "unsafe"

const Available = false

func Call(entry uintptr, state, context unsafe.Pointer, top uintptr) {
	panic("RFE foreign execution boundary is unavailable")
}

func EnableFaults() bool { return false }
func CallFaults(entry uintptr, state, context unsafe.Pointer, top uintptr, sites []FaultSite) {
	Call(entry, state, context, top)
}
