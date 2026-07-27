package main

import (
	"bytes"
	"testing"
)

func TestGeneratedAmd64InstructionBindings(t *testing.T) {
	tests := []struct {
		name string
		emit func(*RTGEmitter)
		want []byte
	}{
		{
			name: "move rdx to rax",
			emit: func(out *RTGEmitter) {
				rtgX8664Mov64(out, rtgX8664RAX, rtgX8664RDX)
			},
			want: []byte{0x48, 0x89, 0xd0},
		},
		{
			name: "add r9 to r8",
			emit: func(out *RTGEmitter) {
				rtgX8664Add64(out, rtgX8664R8, rtgX8664R9)
			},
			want: []byte{0x4d, 0x01, 0xc8},
		},
		{
			name: "load from rsp displacement",
			emit: func(out *RTGEmitter) {
				rtgX8664Load64(out, rtgX8664RAX, RTGAddress{
					Base: rtgX8664RSP, Displacement: 8, Scale: 1,
				})
			},
			want: []byte{0x48, 0x8b, 0x44, 0x24, 0x08},
		},
		{
			name: "move zero to rax",
			emit: func(out *RTGEmitter) {
				rtgX8664X86MoveImmediate(out, rtgX8664RAX, 0)
			},
			want: []byte{0x31, 0xc0},
		},
	}
	for _, test := range tests {
		var assembly renvoAsm
		emitter := renvoRTGEmitter(&assembly)
		test.emit(&emitter)
		if !bytes.Equal(assembly.code, test.want) {
			t.Errorf("%s = % x, want % x", test.name, assembly.code, test.want)
		}
	}
}

func TestGeneratedAmd64RelativeAddressPreservesAddend(t *testing.T) {
	var assembly renvoAsm
	emitter := renvoRTGEmitter(&assembly)
	label := emitter.NewLabel()
	rtgX8664Load64(&emitter, rtgX8664RAX, RTGAddress{Target: label, Addend: 3})
	emitter.Byte(0xaa)
	emitter.Mark(label)
	emitter.Patch()
	want := []byte{0x48, 0x8b, 0x05, 0x04, 0x00, 0x00, 0x00, 0xaa}
	if !bytes.Equal(assembly.code, want) {
		t.Fatalf("relative address = % x, want % x", assembly.code, want)
	}
}
