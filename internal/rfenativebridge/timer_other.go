//go:build !renvo && (!cgo || !linux || !amd64)

package rfenativebridge

import "unsafe"

func MonotonicNS() uint64 { return 0 }
func CallUntil(entry uintptr, state, context unsafe.Pointer, top uintptr, sites []FaultSite,
	remaining, total, memory *uint64, deadline uint64) bool {
	panic("RFE native timer boundary is unavailable")
}
