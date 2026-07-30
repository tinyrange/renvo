package main

import (
	"bytes"
	"testing"
)

func TestGenerated386DirectEmitterBindings(t *testing.T) {
	eax := RTGRegister{Code: 0, Valid: true}
	ecx := RTGRegister{Code: 1, Valid: true}
	edx := RTGRegister{Code: 2, Valid: true}
	ebx := RTGRegister{Code: 3, Valid: true}

	tests := []struct {
		name string
		emit func(*renvoAsm)
		want []byte
	}{
		{
			name: "move",
			emit: func(out *renvoAsm) { rtgX8632DirectMove(out, eax, ecx) },
			want: []byte{0x89, 0xc8},
		},
		{
			name: "signed load",
			emit: func(out *renvoAsm) {
				rtgX8632DirectLoadI16(out, eax, renvoRTGAddress{
					Base: edx, Displacement: 4,
				})
			},
			want: []byte{0x0f, 0xbf, 0x42, 0x04},
		},
		{
			name: "scaled address",
			emit: func(out *renvoAsm) {
				rtgX8632DirectAddress(out, eax, renvoRTGAddress{
					Base: ebx, Index: ecx, Scale: 4, Displacement: 8,
				})
			},
			want: []byte{0x8d, 0x44, 0x8b, 0x08},
		},
		{
			name: "zero immediate",
			emit: func(out *renvoAsm) {
				rtgX8632DirectMoveImmediate(out, edx, 0)
			},
			want: []byte{0x31, 0xd2},
		},
		{
			name: "fixed syscall",
			emit: func(out *renvoAsm) { rtgX8632DirectHostSyscall(out) },
			want: []byte{0xcd, 0x80},
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

func TestGenerated386DirectRelocations(t *testing.T) {
	var out renvoAsm
	label := renvoAsmNewLabel(&out)
	rtgX8632DirectAddress(&out, RTGRegister{Code: 0, Valid: true},
		renvoRTGAddress{Target: label, TargetValid: true})
	renvoAsmMarkLabel(&out, label)
	rtgX8632PatchRelocations(&out)
	want := []byte{
		0xe8, 0, 0, 0, 0,
		0x58, 0x81, 0xc0, 7, 0, 0, 0,
	}
	if !bytes.Equal(out.code, want) {
		t.Fatalf("address code = %x, want %x", out.code, want)
	}
}
