package main

import "testing"

func TestARMAdjacentFrameReload(t *testing.T) {
	for _, target := range []int{renvoTargetLinuxArm, renvoTargetLinuxAarch64} {
		for _, offset := range []int{16, 512, 8192} {
			context := renvoNewCompileContext(target, false, false, false)
			a := renvoAsm{c: context, lastPrimaryStoreEnd: -1}
			renvoAsmStorePrimaryStack(&a, offset)
			end := len(a.code)
			renvoAsmLoadPrimaryStack(&a, offset)
			if len(a.code) != end {
				t.Fatalf("target %d offset %d: redundant load retained", target, offset)
			}
			label := renvoAsmNewLabel(&a)
			renvoAsmMarkLabel(&a, label)
			renvoAsmLoadPrimaryStack(&a, offset)
			if len(a.code) == end {
				t.Fatalf("target %d offset %d: load at branch entry removed", target, offset)
			}
		}
	}
}

func TestAarch64AdjacentCallArgumentMove(t *testing.T) {
	for _, barrier := range []string{"none", "label", "relocation", "instruction"} {
		t.Run(barrier, func(t *testing.T) {
			context := renvoNewCompileContext(renvoTargetLinuxAarch64, false, false, false)
			a := renvoAsm{c: context, lastPrimaryStoreEnd: -1}
			renvoAsmPushPrimary(&a)
			if barrier == "label" {
				label := renvoAsmNewLabel(&a)
				renvoAsmMarkLabel(&a, label)
			}
			if barrier == "relocation" {
				renvoAsmAddReloc(&a, 0, renvoAsmNewLabel(&a))
			}
			if barrier == "instruction" {
				renvoAarch64AsmMovRegReg(&a, 1, 2)
			}
			end := len(a.code)
			renvoAsmPopCallWord0(&a)
			if barrier == "none" {
				if len(a.code) != 4 || renvoUnitRead32(a.code, 0) != 0xaa0003e3 {
					t.Fatalf("call argument move = %x", a.code)
				}
			} else if len(a.code) != end+4 {
				t.Fatalf("fold crossed %s barrier: %x", barrier, a.code)
			}
		})
	}
}
