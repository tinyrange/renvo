//go:build windows

package testmeasure

import (
	"testing"
	"unsafe"
)

func TestWindowsJobABILayout(t *testing.T) {
	var limits jobExtended
	wantSize, wantIO, wantPeak := uintptr(144), uintptr(64), uintptr(136)
	if unsafe.Sizeof(uintptr(0)) == 4 {
		wantSize, wantIO, wantPeak = 112, 48, 108
	}
	if unsafe.Sizeof(limits) != wantSize || unsafe.Offsetof(limits.IO) != wantIO || unsafe.Offsetof(limits.PeakJobMemory) != wantPeak || unsafe.Sizeof(jobAccounting{}) != 48 {
		t.Fatal("job accounting structures do not match the Windows ABI")
	}
}
