package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestArmImageReusesCodeCapacity(t *testing.T) {
	for _, strip := range []bool{false, true} {
		for _, capacity := range []int{512, 4096} {
			for _, dataSize := range []int{0, 173} {
				t.Run(fmt.Sprintf("strip=%v/cap=%d/data=%d", strip, capacity, dataSize), func(t *testing.T) {
					makeAsm := func() *renvoAsm {
						a := &renvoAsm{c: &renvoCompileContext{renvoTargetArch: renvoArchArm, renvoTargetOS: renvoOSLinux, renvoNativeIntSize: 4, stripSymbols: strip}, codeOffset: renvoLinuxArmCodeOffset, bssSize: 97}
						a.code = make([]byte, 512, capacity)
						for i := range a.code {
							a.code[i] = byte(i*31 + 7)
						}
						a.data = make([]byte, dataSize)
						for i := range a.data {
							a.data[i] = byte(i*17 + 3)
						}
						return a
					}
					reference := makeAsm()
					want := renvoAsmImageArm(reference)
					a := makeAsm()
					backing := &a.code[0]
					got := renvoAsmImageArmReuseCode(a, a.code)
					if !bytes.Equal(got, want) {
						t.Fatal("image differs from allocating builder")
					}
					if strip && capacity == 4096 && &got[0] != backing {
						t.Fatal("spare code capacity was not reused")
					}
					if !bytes.Equal(a.code, reference.code) {
						t.Fatal("instruction view corrupted during image assembly")
					}
				})
			}
		}
	}
}
