package main

import (
	"bytes"
	"testing"
)

func TestGeneratedAArch64DirectEmitterBindings(t *testing.T) {
	x0 := RTGRegister{Code: 0, Valid: true}
	x1 := RTGRegister{Code: 1, Valid: true}
	x5 := RTGRegister{Code: 5, Valid: true}

	tests := []struct {
		name string
		emit func(*renvoAsm)
		want []byte
	}{
		{
			name: "move",
			emit: func(out *renvoAsm) { rtgAarch64DirectMove(out, x0, x1) },
			want: aarch64TestWords(0xaa0103e0),
		},
		{
			name: "signed load",
			emit: func(out *renvoAsm) {
				rtgAarch64DirectLoadI32(out, x0, renvoRTGAddress{
					Base: x1, Displacement: -8,
				})
			},
			want: aarch64TestWords(0xb89f8020),
		},
		{
			name: "add",
			emit: func(out *renvoAsm) { rtgAarch64DirectAdd(out, x0, x1) },
			want: aarch64TestWords(0x8b010000),
		},
		{
			name: "immediate",
			emit: func(out *renvoAsm) {
				rtgAarch64DirectMoveImmediate(out, x0, 0x1234)
			},
			want: aarch64TestWords(0xd2824680),
		},
		{
			name: "condition",
			emit: func(out *renvoAsm) {
				rtgAarch64DirectSetCondition(
					out, RTGCondition{Code: 0}, x5)
			},
			want: aarch64TestWords(0x9a9f17e5),
		},
		{
			name: "indirect call",
			emit: func(out *renvoAsm) {
				rtgAarch64DirectCallIndirect(out, x5)
			},
			want: aarch64TestWords(0xd63f00a0),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var out renvoAsm
			test.emit(&out)
			if !bytes.Equal(out.code, test.want) {
				t.Fatalf("code = %x, want %x", out.code, test.want)
			}
		})
	}
}

func TestGeneratedAArch64DirectRelocations(t *testing.T) {
	var out renvoAsm
	label := renvoAsmNewLabel(&out)
	rtgAarch64DirectAddress(&out, RTGRegister{Code: 0, Valid: true},
		renvoRTGAddress{Target: label, TargetValid: true})
	renvoAsmMarkLabel(&out, label)
	rtgAarch64PatchRelocations(&out)
	want := aarch64TestWords(0x10000020)
	if !bytes.Equal(out.code, want) {
		t.Fatalf("ADR code = %x, want %x", out.code, want)
	}
	if len(out.relocs) != 2 || int(out.relocs[0]) != 0 ||
		int(out.relocs[1]) != label {
		t.Fatalf("ADR relocations = %v, want [0 %d]", out.relocs, label)
	}
}

func aarch64TestWords(words ...uint32) []byte {
	out := make([]byte, 0, len(words)*4)
	for _, word := range words {
		out = append(out, byte(word), byte(word>>8), byte(word>>16), byte(word>>24))
	}
	return out
}
