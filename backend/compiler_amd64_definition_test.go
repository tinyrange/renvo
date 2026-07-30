package main

import (
	"bytes"
	"testing"
)

func TestGeneratedAMD64DirectEmitterBindings(t *testing.T) {
	rax := RTGRegister{Code: 0, Valid: true}
	rcx := RTGRegister{Code: 1, Valid: true}
	rdi := RTGRegister{Code: 7, Valid: true}
	r8 := RTGRegister{Code: 8, Valid: true}
	r12 := RTGRegister{Code: 12, Valid: true}

	tests := []struct {
		name string
		emit func(*renvoAsm)
		want []byte
	}{
		{
			name: "move",
			emit: func(out *renvoAsm) { rtgX8664DirectMove(out, rax, rcx) },
			want: []byte{0x48, 0x89, 0xc8},
		},
		{
			name: "load extended register and SIB base",
			emit: func(out *renvoAsm) {
				rtgX8664DirectLoadU8(out, r8, renvoRTGAddress{
					Base: r12, Displacement: 16, Scale: 1,
				})
			},
			want: []byte{0x45, 0x0f, 0xb6, 0x44, 0x24, 0x10},
		},
		{
			name: "store byte register",
			emit: func(out *renvoAsm) {
				rtgX8664DirectStoreU8(out, renvoRTGAddress{Base: rdi, Scale: 1}, r8)
			},
			want: []byte{0x44, 0x88, 0x07},
		},
		{
			name: "zero immediate",
			emit: func(out *renvoAsm) { rtgX8664DirectMoveImmediate(out, rax, 0) },
			want: []byte{0x31, 0xc0},
		},
		{
			name: "condition",
			emit: func(out *renvoAsm) {
				rtgX8664DirectSetCondition(out, RTGCondition{SetOpcode: 0x94}, r8)
			},
			want: []byte{0x41, 0x0f, 0x94, 0xc0},
		},
		{
			name: "fixed syscall",
			emit: func(out *renvoAsm) { rtgX8664DirectHostSyscall(out) },
			want: []byte{0x0f, 0x05},
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

func TestGeneratedAMD64DirectRelocations(t *testing.T) {
	var out renvoAsm
	label := renvoAsmNewLabel(&out)
	rtgX8664DirectJump(&out, label)
	renvoAsmMarkLabel(&out, label)
	rtgX8664PatchRelocations(&out)
	want := []byte{0xe9, 0, 0, 0, 0}
	if !bytes.Equal(out.code, want) {
		t.Fatalf("jump code = %x, want %x", out.code, want)
	}
}
