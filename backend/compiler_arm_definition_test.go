package main

import (
	"bytes"
	"testing"
)

func TestGeneratedARMDirectEmitterBindings(t *testing.T) {
	r0 := RTGRegister{Code: 0, Valid: true}
	r1 := RTGRegister{Code: 1, Valid: true}
	r2 := RTGRegister{Code: 2, Valid: true}
	r3 := RTGRegister{Code: 3, Valid: true}
	r5 := RTGRegister{Code: 5, Valid: true}

	tests := []struct {
		name string
		emit func(*renvoAsm)
		want []byte
	}{
		{
			name: "move",
			emit: func(out *renvoAsm) { rtgArmDirectMove(out, r0, r1) },
			want: armTestWords(0xe1a00001),
		},
		{
			name: "signed byte load",
			emit: func(out *renvoAsm) {
				rtgArmDirectLoadI8(out, r0, renvoRTGAddress{
					Base: r1, Displacement: 4,
				})
			},
			want: armTestWords(0xe1d100d4),
		},
		{
			name: "unsigned halfword load",
			emit: func(out *renvoAsm) {
				rtgArmDirectLoadU16(out, r2, renvoRTGAddress{
					Base: r3, Displacement: -8,
				})
			},
			want: armTestWords(0xe15320b8),
		},
		{
			name: "scaled address",
			emit: func(out *renvoAsm) {
				rtgArmDirectAddress(out, r0, renvoRTGAddress{
					Base: r1, Index: r2, Scale: 4,
				})
			},
			want: armTestWords(0xe0810102),
		},
		{
			name: "condition",
			emit: func(out *renvoAsm) {
				rtgArmDirectSetCondition(out, RTGCondition{Code: 0}, r5)
			},
			want: armTestWords(0xe3a05000, 0x03a05001),
		},
		{
			name: "indirect call",
			emit: func(out *renvoAsm) { rtgArmDirectCallIndirect(out, r3) },
			want: armTestWords(0xe12fff33),
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

func TestGeneratedARMDirectRelocations(t *testing.T) {
	var out renvoAsm
	label := renvoAsmNewLabel(&out)
	rtgArmDirectAddress(&out, RTGRegister{Code: 0, Valid: true},
		renvoRTGAddress{Target: label, TargetValid: true})
	renvoAsmMarkLabel(&out, label)
	rtgArmPatchRelocations(&out)
	want := armTestWords(0xe30f0ffc, 0xe34f0fff, 0xe08f0000)
	if !bytes.Equal(out.code, want) {
		t.Fatalf("address code = %x, want %x", out.code, want)
	}
}

func armTestWords(words ...uint32) []byte {
	out := make([]byte, 0, len(words)*4)
	for _, word := range words {
		out = append(out, byte(word), byte(word>>8), byte(word>>16), byte(word>>24))
	}
	return out
}
