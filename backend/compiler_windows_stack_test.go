package main

import (
	"encoding/binary"
	"testing"
)

func TestWindowsPE64ReservesGrowableStack(t *testing.T) {
	for _, target := range []int{renvoTargetWindowsAmd64, renvoTargetWindowsArm64} {
		renvoSetTarget(target)
		image := renvoAppendPEHeader64(nil, 0x200, 0x200, 0x2000, 0x200, 0x200, 0, 0, 0, 0)
		peOffset := int(binary.LittleEndian.Uint32(image[0x3c:]))
		optionalHeader := peOffset + 24
		stackReserve := binary.LittleEndian.Uint64(image[optionalHeader+72:])
		stackCommit := binary.LittleEndian.Uint64(image[optionalHeader+80:])
		if stackReserve != 8<<20 {
			t.Fatalf("target %d stack reserve = %d, want %d", target, stackReserve, 8<<20)
		}
		if stackCommit >= stackReserve {
			t.Fatalf("target %d stack commit = %d, want less than reserve %d", target, stackCommit, stackReserve)
		}
	}
}
