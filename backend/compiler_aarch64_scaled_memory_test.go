package main

import "testing"

func TestAArch64ScaledMemoryOffsets(t *testing.T) {
	for _, tc := range []struct{ size, load, store int }{
		{1, 0x39400000, 0x39000000},
		{2, 0x79800000, 0x79000000},
		{4, 0xb9800000, 0xb9000000},
		{8, 0xf9400000, 0xf9000000},
	} {
		for _, disp := range []int{256, tc.size * 4095} {
			load := generatedAArch64Word(t, func(a *renvoAsm) { renvoAarch64AsmLoadRegMem(a, 3, 5, disp, tc.size) })
			store := generatedAArch64Word(t, func(a *renvoAsm) { renvoAarch64AsmStoreRegMem(a, 3, 5, disp, tc.size) })
			operand := (disp/tc.size)<<10 | 5<<5 | 3
			if load != tc.load|operand || store != tc.store|operand {
				t.Fatalf("size %d offset %d: load=%#x store=%#x", tc.size, disp, load, store)
			}
		}
		for _, disp := range []int{-264, tc.size * 4096} {
			var a renvoAsm
			renvoAarch64AsmStoreRegMem(&a, 3, 5, disp, tc.size)
			if len(a.code) <= 4 {
				t.Fatalf("out-of-range offset %d encoded directly", disp)
			}
		}
		if tc.size > 1 {
			var a renvoAsm
			renvoAarch64AsmLoadRegMem(&a, 3, 5, 257, tc.size)
			if len(a.code) <= 4 {
				t.Fatalf("unaligned size %d offset encoded directly", tc.size)
			}
		}
	}
}
