package syntax

import "testing"

func TestReflectDirectiveIsExplicitAndAdjacent(t *testing.T) {
	tests := []struct {
		source string
		want   bool
	}{
		{"package p\n//renvo:reflect\ntype T struct{}\n", true},
		{"package p\r\n//renvo:reflect\r\ntype T struct{}\r\n", true},
		{"package p\ntype (\n //renvo:reflect\n T struct{}\n)\n", true},
		{"package p\n//renvo:reflect\n\ntype T struct{}\n", false},
		{"package p\n//renvo:reflect extra\ntype T struct{}\n", false},
		{"package p\ntype T struct{}\n", false},
		{"package p\n//renvo:reflect\ntype (\n T struct{}\n)\n", false},
	}
	for _, test := range tests {
		file := ParseFile([]byte(test.source))
		if !file.Ok || len(file.Decls) != 1 {
			t.Fatal("invalid fixture")
		}
		if got := ReflectDirective(file, file.Decls[0]); got != test.want {
			t.Fatalf("directive=%v want=%v: %s", got, test.want, test.source)
		}
	}
}
