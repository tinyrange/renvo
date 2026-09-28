package main

import (
	"bytes"
	"testing"
)

func TestStringDataInterning(t *testing.T) {
	for _, alignment := range []int{1, 2, 4, 8} {
		for _, initial := range [][]byte{
			nil,
			[]byte("prefix\x00fix\x00x\x00\x00"),
			[]byte("ab\x00cd\x00ab\x00ce\x00"),
			bytes.Repeat([]byte("a\x00bc\x00"), 120),
		} {
			for _, msg := range [][]byte{nil, []byte("x"), []byte("fix"), []byte("prefix"), []byte("ab\x00cd"), []byte("bc"), bytes.Repeat([]byte("z"), 600)} {
				g := renvoLinearGen{asm: renvoAsm{data: append([]byte(nil), initial...)}}
				needle := append(append([]byte(nil), msg...), 0)
				start := len(initial) - renvoStringInternSearchBytes
				if start < 0 {
					start = 0
				}
				want := -1
				for at := start; at+len(needle) <= len(initial); at++ {
					if at%alignment == 0 && bytes.Equal(initial[at:at+len(needle)], needle) {
						want = at
						break
					}
				}
				expected := append([]byte(nil), initial...)
				if want < 0 {
					for len(expected)%alignment != 0 {
						expected = append(expected, 0)
					}
					want = len(expected)
					expected = append(expected, needle...)
				}
				got := renvoAddStringDataAligned(&g, msg, alignment)
				if got != want || !bytes.Equal(g.asm.data, expected) {
					t.Fatalf("alignment=%d initial=%d msg=%q: offset=%d want=%d data size=%d want=%d", alignment, len(initial), msg, got, want, len(g.asm.data), len(expected))
				}
			}
		}
	}
}
