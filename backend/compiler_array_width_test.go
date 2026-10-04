package main

import (
	"fmt"
	"testing"
)

func TestArrayLengthPreserves64BitTargetOn32BitCompiler(t *testing.T) {
	for _, test := range []struct {
		bound string
		want  uint64
	}{
		{"4294967296", 1 << 32},
		{"0x100000000", 1 << 32},
		{"1<<32", 1 << 32},
		{"(1<<32)+3", 1<<32 + 3},
		{"(uint64(1)<<40)>>8", 1 << 32},
		{"uint64(^uint32(0))+1", 1 << 32},
		{"int(^uint(0)>>2)", 1<<62 - 1},
		{"^uint32(0)", 1<<32 - 1},
		{"(-8589934592)/(-2)", 1 << 32},
		{"(1<<20)*(1<<12)", 1 << 32},
		{"NamedBound(1)<<32", 1 << 32},
		{"SignedBound(-8589934592)/SignedBound(-2)", 1 << 32},
		{"^ShortBound(0)", 1<<32 - 1},
		{"AliasBound(1)<<32", 1 << 32},
	} {
		t.Run(test.bound, func(t *testing.T) {
			resetRuntime()
			source := []byte(fmt.Sprintf("package main\ntype Element struct{}\ntype NamedBound uint64\ntype SignedBound int64\ntype ShortBound uint32\ntype AliasBound=uint64\ntype Value [%s]Element\nfunc appMain()int{return 0}\n", test.bound))
			context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
			program := renvoParseProgramWithContext(source, context)
			// Force the same two-word parser contract used by a real 32-bit
			// self-hosted compiler. Cross-host execution also has a corpus case.
			program.compilerInt32 = true
			var meta renvoMeta
			renvoBuildMetaInto(&program, &meta)
			if !meta.ok {
				t.Fatal("valid 64-bit target array was rejected")
			}
			found := false
			for _, typ := range meta.types {
				if typ.kind == renvoTypeArray {
					found = true
					if typ.arrayLength != test.want {
						t.Fatalf("length=%d, want %d", typ.arrayLength, test.want)
					}
				}
			}
			if !found {
				t.Fatal("array metadata was not built")
			}
		})
	}
}

func TestWideArrayLengthsRetainAnonymousTypeIdentity(t *testing.T) {
	resetRuntime()
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	program := renvoParseProgramWithContext([]byte("package main\ntype E struct{}\ntype A = [4294967296]E\ntype B = [4294967297]E\nfunc appMain()int{return 0}\n"), context)
	program.compilerInt32 = true
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("valid array aliases were rejected")
	}
	var arrays []int
	for i, typ := range meta.types {
		if typ.kind == renvoTypeArray {
			arrays = append(arrays, i)
		}
	}
	if len(arrays) != 2 || renvoTypesEquivalent(&meta, arrays[0], arrays[1]) || renvoRuntimeTypesIdentical(&meta, arrays[0], arrays[1], 0) {
		t.Fatal("different wide array lengths acquired one type identity")
	}
}

func TestWideArrayStructPointeesRetainMetadata(t *testing.T) {
	for _, test := range []struct {
		element  string
		overflow bool
	}{
		{"struct{}", false},
		{"int64", true},
	} {
		for _, compilerInt32 := range []bool{false, true} {
			t.Run(fmt.Sprintf("element=%s/compiler32=%v", test.element, compilerInt32), func(t *testing.T) {
				resetRuntime()
				context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
				program := renvoParseProgramWithContext([]byte(fmt.Sprintf(`package main
type Element %s
type Huge [int(^uint(0)>>2)]Element
type Nested struct{Prefix byte;Value Huge;Suffix int}
type Alias = Nested
type Wrap struct{Value Alias}
func appMain()int{var value *Wrap;return len(value.Value.Value)}
`, test.element)), context)
				program.compilerInt32 = compilerInt32
				var meta renvoMeta
				renvoBuildMetaInto(&program, &meta)
				if !meta.ok {
					t.Fatal("describing an unmaterialized large-array pointee failed")
				}
				structs := 0
				for _, typ := range meta.types {
					if typ.kind == renvoTypeStruct && typ.count > 0 {
						structs++
						if (typ.size < 0) != test.overflow {
							t.Fatalf("aggregate size=%d, want overflow=%v", typ.size, test.overflow)
						}
					}
				}
				if structs != 2 {
					t.Fatalf("found %d aggregate layouts, want 2", structs)
				}
			})
		}
	}
}
