//go:build !renvo

package backendcompiled

import "testing"

// Parse real recursive aggregates, arrays and function fields through the
// compiler rather than testing a parallel model of its layout arithmetic.
// The integer carrier is deliberately narrower than the selected address
// descriptor, as in an ILP32 ABI on a 64-bit-address machine.
func TestNativeAddressLayoutIndependentOfIntegerCarrier(t *testing.T) {
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	context.renvoNativeIntSize = 4
	context.objectFile = true
	source := []byte(`package main
 type Node struct {
  tag byte
  next *Node
  callback func(int) int
  links [2]*Node
  tail byte
 }
 func appMain(args []string) int { return 0 }
 `)
	program := renvoParseProgramWithContext(source, context)
	meta := renvoBuildMeta(&program)
	found := false
	for i := range meta.types {
		typ := &meta.types[i]
		if typ.kind != renvoTypeStruct || typ.count != 5 {
			continue
		}
		found = true
		if typ.size != 48 || typ.nativeAlign != 8 {
			t.Fatalf("recursive native aggregate size/alignment = %d/%d, want 48/8", typ.size, typ.nativeAlign)
		}
		for j, want := range []int{0, 8, 16, 24, 40} {
			field := &meta.fields[typ.first+j]
			if field.offset != want {
				t.Errorf("field %d offset = %d, want %d", j, field.offset, want)
			}
		}
		pointer := renvoResolveType(&meta, meta.fields[typ.first+1].typ)
		callback := renvoResolveType(&meta, meta.fields[typ.first+2].typ)
		if pointer.size != 8 || callback.size != 8 {
			t.Errorf("pointer/function widths = %d/%d, want 8/8", pointer.size, callback.size)
		}
	}
	if !found {
		t.Fatal("parsed aggregate missing")
	}
	if meta.types[renvoTypeInt].size != 4 {
		t.Fatal("address layout changed language integer size")
	}
	// The normalized compiler slot is not a native pointer and must not shrink.
	if renvoBackendValueSlotSize != 8 {
		t.Fatal("address layout changed compiler carrier ABI")
	}
}
