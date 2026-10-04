package check

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"renvo.dev/internal/load"
)

func TestGenericPreparationUsesTargetWidths(t *testing.T) {
	for _, test := range []struct {
		name, source     string
		valid32, valid64 bool
	}{
		{"signed", `func Value[T ~int]() T{return 1<<31};func main(){}`, false, true},
		{"negative", `func Value[T ~int]() T{return -1<<31};func main(){}`, true, true},
		{"unsigned", `func Value[T ~uint]() T{return 1<<32};func main(){}`, false, true},
		{"pointer", `func Value[T ~uintptr]() T{return 1<<32};func main(){}`, false, true},
		{"inference", `func Identity[T any](v T)T{return v};func main(){_=Identity(1<<31)}`, false, true},
		{"default_variable", `func Value[T any](){v:=1<<31;_=v};func main(){}`, false, true},
		{"typed_index", `func Value[S ~[]byte](s S)byte{return s[int64(1<<31)]};func main(){}`, false, true},
		{"slice_bound", `func Value[S ~[]byte](s S)S{return s[:uint64(1<<31)]};func main(){}`, false, true},
		{"make_length", `func Value[S ~[]byte]()S{return make(S,1<<31)};func main(){}`, false, true},
		{"array_length", `type Value[T any] [1<<31]T;func main(){}`, false, true},
		{"inferred_array_extent", `func Value[T any](v T){_=[...]T{(1<<31)-1:v}};func main(){}`, true, true},
		{"slice_extent", `func Value[T any](v T){_=[]T{(1<<31)-1:v}};func main(){}`, true, true},
		{"slice_extent_wide", `func Value[T any](v T){_=[]T{(1<<63)-1:v}};func main(){}`, false, true},
		{"inferred_extent_wide", `func Value[T any](v T){_=[...]T{(1<<63)-1:v}};func main(){}`, false, true},
		{"implicit_index_wide32", `func Value[T any](v T){_=[]T{(1<<31)-1:v,v}};func main(){}`, true, true},
		{"implicit_index_wide64", `func Value[T any](v T){_=[]T{(1<<63)-1:v,v}};func main(){}`, false, true},
		{"inferred_len_explicit_bound", `func Value[T any](v T){a:=[...]T{(1<<31)-1:v};_=[len(a)]T{}};func main(){}`, false, true},
		{"wide_array_length", `type Value[T any] [1<<32]T;func main(){}`, false, true},
		{"wide_array_len", `func Value[T ~int]()T{var p *[1<<32]byte;return T(len(p))};func main(){}`, false, true},
		{"wide_array_cap", `func Value[T ~int]()T{var p *[(1<<32)+1]byte;return T(cap(p))};func main(){}`, false, true},
		{"array_len_bound", `func Value[T any](p *[1<<32]T)T{return p[(1<<32)-1]};func main(){}`, false, true},
		{"array_index_past_end", `func Value[T any](p *[1<<32]T)T{return p[1<<32]};func main(){}`, false, false},
		{"wide_slice_order", `func Value[S ~[]byte](s S)S{return s[1<<32:(1<<32)+1]};func main(){}`, false, true},
		{"wide_slice_bad_order", `func Value[S ~[]byte](s S)S{return s[(1<<32)+1:1<<32]};func main(){}`, false, false},
		{"wide_make", `func Value[S ~[]byte]()S{return make(S,1<<32,(1<<32)+1)};func main(){}`, false, true},
		{"wide_make_bad_order", `func Value[S ~[]byte]()S{return make(S,(1<<32)+1,1<<32)};func main(){}`, false, false},
		{"wide_literal_indices", `func Value[T any](v T){_=[]T{0:v,1<<32:v,(1<<32)+1:v}};func main(){}`, false, true},
		{"wide_literal_duplicate", `func Value[T any](v T){_=[]T{1<<32:v,(1<<31)*2:v}};func main(){}`, false, false},
		{"max_array_length", `type Value[T any] [(1<<63)-1]T;func main(){}`, false, true},
		{"array_length_overflow", `type Value[T any] [1<<63]T;func main(){}`, false, false},
		{"max_slice_extent", `func Value[T any](v T){_=[]T{(1<<63)-2:v}};func main(){}`, false, true},
		{"fixed_width", `func Value[T ~int64]()T{return 1<<31};func main(){}`, true, true},
		{"complement", `func Value[T ~uint]()T{return T(^uint(0))};func main(){_=Value[uint]()}`, true, true},
	} {
		for _, width := range []int{32, 64} {
			t.Run(test.name+"/"+genericDecimal(width), func(t *testing.T) {
				want := test.valid32
				arch := "386"
				if width == 64 {
					want = test.valid64
					arch = "amd64"
				}
				source := []byte("package main;" + test.source)
				set := token.NewFileSet()
				file, err := parser.ParseFile(set, "main.go", source, 0)
				if err != nil {
					t.Fatal(err)
				}
				config := types.Config{Sizes: types.SizesFor("gc", arch)}
				if _, err := config.Check("example.com/case", set, []*ast.File{file}, nil); (err == nil) != want {
					t.Fatalf("Go target validity differs from test: %v", err)
				}
				graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: source}})
				graph.Layout = load.TargetLayout{WordBits: width, PointerBits: width}
				got := PrepareGenerics(graph)
				if got.Ok != want {
					t.Fatalf("valid=%v want=%v: %s", got.Ok, want, got.Message)
				}
			})
		}
	}
}

func TestGenericWideArrayLengthIdentityAndSpelling(t *testing.T) {
	var types genericTypes
	integer := types.basic("int")
	first := types.intern(genericType{kind: genericArray, elem: integer, length: 1 << 32})
	second := types.intern(genericType{kind: genericArray, elem: integer, length: (1 << 32) + 1})
	small := types.intern(genericType{kind: genericArray, elem: integer, length: 0})
	if first == second || first == small || second == small {
		t.Fatal("wide array length lost type identity")
	}
	if types.intern(genericType{kind: genericArray, elem: integer, length: 1 << 32}) != first {
		t.Fatal("wide array type was not interned")
	}
	e := genericEnvironment{types: types}
	if got := e.typeDisplay(first, true); got != "[4294967296]int" {
		t.Fatalf("wide array spelling = %s", got)
	}
	wide := types.intern(genericType{kind: genericArray, elem: integer, length: 1 << 63})
	e.types = types
	if got := e.typeDisplay(wide, true); got != "[9223372036854775808]int" {
		t.Fatalf("inferred array spelling = %s", got)
	}
}

func TestGenericConstantsUseIndependentPointerWidth(t *testing.T) {
	graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main;func Value[T ~int]()T{return 1<<32};func main(){}`)}})
	graph.Layout = load.TargetLayout{WordBits: 64, PointerBits: 32}
	if got := PrepareGenerics(graph); !got.Ok {
		t.Fatal(got.Message)
	}
	graph = genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main;func Value[T ~uintptr]()T{return 1<<32};func main(){}`)}})
	graph.Layout = load.TargetLayout{WordBits: 64, PointerBits: 32}
	if got := PrepareGenerics(graph); got.Ok {
		t.Fatal("uintptr constant exceeded pointer width")
	}
}
