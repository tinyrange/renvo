package main

import (
	"bytes"
	"testing"
)

func TestGeneratedVM32DirectEmitterBindings(t *testing.T) {
	rax := RTGRegister{Code: 0, Valid: true}
	rdx := RTGRegister{Code: 1, Valid: true}
	rcx := RTGRegister{Code: 2, Valid: true}
	rdi := RTGRegister{Code: 3, Valid: true}

	tests := []struct {
		name string
		emit func(*renvoAsm)
		want []byte
	}{
		{
			name: "move",
			emit: func(out *renvoAsm) {
				rtgWasm32DirectMove(out, rdx, rax)
			},
			want: []byte{renvoWasm32OpMovRegReg, 1, 0},
		},
		{
			name: "signed byte load",
			emit: func(out *renvoAsm) {
				rtgWasm32DirectLoadI8(out, rax, renvoRTGAddress{
					Base: rdx, Displacement: 4,
				})
			},
			want: []byte{
				renvoWasm32OpLoadMem, 0, 1,
				4, 0, 0, 0, 129,
			},
		},
		{
			name: "unsigned halfword indexed load",
			emit: func(out *renvoAsm) {
				rtgWasm32DirectLoadU16(out, rcx, renvoRTGAddress{
					Base: rdx, Index: rdi, Scale: 2, Displacement: -8,
				})
			},
			want: []byte{
				renvoWasm32OpLoadIndex, 2, 1, 3, 2,
				0xf8, 0xff, 0xff, 0xff, 66,
			},
		},
		{
			name: "multiply",
			emit: func(out *renvoAsm) {
				rtgWasm32DirectMultiply(out, rax, rcx)
			},
			want: []byte{renvoWasm32OpMulRegReg, 0, 2},
		},
		{
			name: "condition",
			emit: func(out *renvoAsm) {
				rtgWasm32DirectSetCondition(
					out, RTGCondition{Code: renvoWasm32CondNe}, rdx)
			},
			want: []byte{
				renvoWasm32OpSetCond, renvoWasm32CondNe,
				renvoWasm32OpMovRegReg, 1, 0,
			},
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

func TestGeneratedVM32DirectRelocations(t *testing.T) {
	var out renvoAsm
	label := renvoAsmNewLabel(&out)
	rtgWasm32DirectAddress(&out, RTGRegister{Code: 0, Valid: true},
		renvoRTGAddress{Target: label, TargetValid: true, Addend: 7})
	renvoAsmMarkLabel(&out, label)
	rtgWasm32PatchRelocations(&out)
	want := []byte{renvoWasm32OpMovRegImm, 0, 13, 0, 0, 0}
	if !bytes.Equal(out.code, want) {
		t.Fatalf("address code = %x, want %x", out.code, want)
	}
}
