package syntax

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestGenericDeclarationParameters(t *testing.T) {
	src := []byte(`package sample
type (
 Pair[A, B any] struct { First A; Second B }
 Alias[T ~int | ~string] = Pair[T, T]
)
func Convert[S ~[]E, E interface { ~int | ~int64 }](s S) []E { return s }
func (p *Pair[X, Y]) FirstValue() X { return p.First }
`)
	f := ParseFile(src)
	if !f.Ok {
		t.Fatalf("parse: %d at %d", f.Error, f.ErrorTok)
	}
	if len(TypeParameterLists(&f)) != 3 {
		t.Fatalf("generic declarations: %d", len(TypeParameterLists(&f)))
	}
	wantNames := [][]string{{"A", "B"}, {"T"}, {"S", "E"}}
	wantConstraints := [][]string{{"any", "any"}, {"~int | ~string"}, {"~[]E", "interface { ~int | ~int64 }"}}
	for i, list := range TypeParameterLists(&f) {
		if len(list.Parameters) != len(wantNames[i]) {
			t.Fatalf("parameters: %#v", list)
		}
		for j, p := range list.Parameters {
			name := tokenString(f, p.NameTok)
			constraint := string(src[f.Tokens[p.ConstraintStart].Start:f.Tokens[p.ConstraintEnd-1].End])
			if name != wantNames[i][j] || constraint != wantConstraints[i][j] {
				t.Fatalf("parameter %d/%d = %s %s", i, j, name, constraint)
			}
		}
	}
	if f.Funcs[0].ParamsStart != TypeParameterLists(&f)[2].EndTok {
		t.Fatal("value parameters overlap type parameters")
	}
	if len(TypeParameters(&f, f.Funcs[1].NameTok).Parameters) != 0 {
		t.Fatal("receiver parameters became method parameters")
	}
}

func TestGenericTypeArrayAmbiguity(t *testing.T) {
	for _, spec := range []string{
		"A [P *C]int", "A [P(C)]int", "A [P*C | Q]int", "A [N]int", "A [pkg.N]int",
		"A [P *C,] int", "A [P (C),] int", "A [P *C | ~int] int",
		"A [P *struct{}] int", "A [P []int] int", "A [P [3]int] int",
		"A [P chan int] int", "A [P func(int, int)] int", "A [P map[int]string] int",
		"A [P interface{ M(int, string) }] int", "A [P, Q any] int",
		"A [P any] = []P", "A [P ~int] int", "A [P (*C)] int",
	} {
		t.Run(spec, func(t *testing.T) {
			src := "package sample; type " + spec
			ref, err := parser.ParseFile(token.NewFileSet(), "test.go", src, 0)
			if err != nil {
				t.Fatal(err)
			}
			want := ref.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec).TypeParams != nil
			got := ParseFile([]byte(src))
			if !got.Ok {
				t.Fatalf("parse: %d at %d", got.Error, got.ErrorTok)
			}
			if (len(TypeParameterLists(&got)) > 0) != want {
				t.Fatalf("generic = %v, want %v", len(TypeParameterLists(&got)) > 0, want)
			}
		})
	}
}

func TestGenericParameterListSyntaxErrors(t *testing.T) {
	for _, decl := range []string{
		"func F[]() {}", "func F[T]() {}", "func F[T,]() {}",
		"func F[T any, U]() {}", "func F[, T any]() {}",
		"func F[T any; U any]() {}", "func (v V) M[T any]() {}",
	} {
		if f := ParseFile([]byte("package sample; " + decl)); f.Ok {
			t.Errorf("accepted %s", decl)
		}
	}
}
