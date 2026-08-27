package uefi

import (
	"testing"
	"unsafe"
)

func TestStatus(t *testing.T) {
	if Success.Failed() || !NotFound.Failed() || NotFound.Error() != "not found" {
		t.Fatalf("unexpected status behavior: %v %v %q", Success.Failed(), NotFound.Failed(), NotFound.Error())
	}
	unknown := Status(99) | Status(1)<<(uintptrBits-1)
	if got := unknown.Error(); got != "UEFI error 99" {
		t.Fatalf("unknown status = %q", got)
	}
}

func TestMemoryMapControl(t *testing.T) {
	control := make([]byte, 32)
	base := uintptr(unsafe.Pointer(&control[0]))
	store64(base, 0, 1024)
	store64(base, 8, 7)
	store64(base, 16, 48)
	store32(base, 24, 1)
	view := MemoryMap(base)
	if view.Size() != 1024 || view.Key() != 7 || view.DescriptorSize() != 48 || view.Version() != 1 {
		t.Fatalf("memory map control = %d %d %d %d", view.Size(), view.Key(), view.DescriptorSize(), view.Version())
	}
}

func TestUTF16RoundTrip(t *testing.T) {
	units := utf16z("Renvo U0001f33f")
	if units[len(units)-1] != 0 || string16(&units[0]) != "Renvo U0001f33f" {
		t.Fatalf("UTF-16 round trip = %#v, %q", units, string16(&units[0]))
	}
}

func TestCallDispatch(t *testing.T) {
	t.Cleanup(func() { CallHook = nil })
	CallHook = func(function uintptr, arguments []uintptr) uintptr {
		if function != 7 || len(arguments) != 3 || arguments[2] != 13 {
			t.Fatalf("call = %d %#v", function, arguments)
		}
		return 0
	}
	if status := Call(7, 11, 12, 13); status != Success {
		t.Fatalf("status = %v", status)
	}
}

func TestFileOpenEncodesPath(t *testing.T) {
	t.Cleanup(func() { CallHook = nil })
	CallHook = func(function uintptr, arguments []uintptr) uintptr {
		if function != 7 || len(arguments) != 5 {
			t.Fatalf("open call = %d %#v", function, arguments)
		}
		if got := string16((*uint16)(unsafe.Pointer(arguments[2]))); got != "EFI\\Renvo U0001f33f" {
			t.Fatalf("firmware path = %q", got)
		}
		return 0
	}
	file := &File{OpenFunction: 7}
	if _, status := file.Open("EFI\\Renvo U0001f33f", FileModeRead, 0); status != Success {
		t.Fatalf("Open status = %v", status)
	}
	if _, status := file.Open(string(make([]byte, 260)), FileModeRead, 0); status != InvalidParameter {
		t.Fatalf("oversized Open status = %v", status)
	}
}
