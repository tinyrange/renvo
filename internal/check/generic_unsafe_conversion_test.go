package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestGenericUnsafeConversionsRequireUniformUnderlyingTypes(t *testing.T) {
	for _, test := range []struct {
		name, source string
		valid        bool
	}{
		{"pointer_to_integer", `func F[P ~unsafe.Pointer](p P)uintptr{return uintptr(p)}`, true},
		{"integer_to_pointer", `func F[P ~uintptr](p P)unsafe.Pointer{return unsafe.Pointer(p)}`, true},
		{"mixed_source_integer", `func F[P ~unsafe.Pointer|~uintptr](p P)uintptr{return uintptr(p)}`, false},
		{"mixed_source_pointer", `func F[P ~unsafe.Pointer|~uintptr](p P)unsafe.Pointer{return unsafe.Pointer(p)}`, false},
		{"mixed_target_integer", `func F[P ~unsafe.Pointer|~uintptr](p uintptr)P{return P(p)}`, false},
		{"mixed_target_pointer", `func F[P ~unsafe.Pointer|~uintptr](p unsafe.Pointer)P{return P(p)}`, false},
		{"ordinary_pointer_union", `func F[P ~*byte|~*int](p P)unsafe.Pointer{return unsafe.Pointer(p)}`, true},
		{"pointer_target_union", `func F[P ~*byte|~*int](p unsafe.Pointer)P{return P(p)}`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			graph := genericTestGraph(t, []load.SourceFile{
				{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;import \"unsafe\";" + test.source + ";func main(){}")},
				{Path: "/std/unsafe/unsafe.go", Src: []byte("package unsafe;type Pointer *byte")},
			})
			got := PrepareGenerics(graph)
			if got.Ok != test.valid {
				t.Fatalf("ok=%v want=%v: %s", got.Ok, test.valid, got.Message)
			}
		})
	}
}
