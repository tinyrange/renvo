//go:build !renvo || !uefi || !amd64

package uefi

var ImageHandleHook func() uintptr
var SystemTableHook func() uintptr
var CallHook func(function uintptr, arguments []uintptr) uintptr

func imageHandle() uintptr {
	if ImageHandleHook != nil {
		return ImageHandleHook()
	}
	return 0
}
func systemTable() uintptr {
	if SystemTableHook != nil {
		return SystemTableHook()
	}
	return 0
}
func hostCall(function uintptr, arguments ...uintptr) uintptr {
	if CallHook != nil {
		return CallHook(function, arguments)
	}
	return uintptr(Unsupported)
}
func call0(function uintptr) uintptr                 { return hostCall(function) }
func call1(function, a0 uintptr) uintptr             { return hostCall(function, a0) }
func call2(function, a0, a1 uintptr) uintptr         { return hostCall(function, a0, a1) }
func call3(function, a0, a1, a2 uintptr) uintptr     { return hostCall(function, a0, a1, a2) }
func call4(function, a0, a1, a2, a3 uintptr) uintptr { return hostCall(function, a0, a1, a2, a3) }
func call5(function, a0, a1, a2, a3, a4 uintptr) uintptr {
	return hostCall(function, a0, a1, a2, a3, a4)
}
