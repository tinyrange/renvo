package syntax

import "testing"

func TestLinkedParseCapacityPreservesTokensAndDeclarations(t *testing.T) {
	for _, src := range []string{
		"package main\nfunc f(a ...int) int {return a[0]}\n",
		"package main\ntype LongLinkedIdentifier struct {First, Second [16]byte}\nfunc f(v LongLinkedIdentifier) [16]byte { return v.Second }\n",
		"package main\n/* lines\ncomments */ var α = `line\ntext`; func f(){println(α, \"escaped\\n\")}\n",
	} {
		want, wantLines := ParseLinkedFile([]byte(src))
		if !want.Ok {
			t.Fatal("bad test source", want.Error)
		}
		for _, capacity := range []int{0, 1, 3, len(want.Tokens), len(src) * 2} {
			got, lines := ParseLinkedFileCapacity([]byte(src), capacity)
			if got.Ok != want.Ok || len(got.Tokens) != len(want.Tokens) || len(got.Decls) != len(want.Decls) || len(got.Funcs) != len(want.Funcs) || len(lines) != len(wantLines) {
				t.Fatal("capacity changed parsing", capacity)
			}
			for j := range lines {
				if lines[j] != wantLines[j] {
					t.Fatal("capacity changed physical lines")
				}
			}
			for j := range got.Tokens {
				if got.Tokens[j] != want.Tokens[j] {
					t.Fatal("capacity changed token", j)
				}
			}
			for j := range got.Decls {
				if got.Decls[j] != want.Decls[j] {
					t.Fatal("capacity changed declaration", j)
				}
			}
			for j := range got.Funcs {
				if got.Funcs[j] != want.Funcs[j] {
					t.Fatal("capacity changed function", j)
				}
			}
		}
	}
}
