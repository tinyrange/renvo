package main

import (
	"bytes"
	"testing"
)

func TestGeneratedAmd64InstructionBindings(t *testing.T) {
	tests := []struct {
		name string
		emit func(*renvoAsm)
		want []byte
	}{
		{
			name: "move zero to primary",
			emit: func(out *renvoAsm) {
				renvoAmd64AsmMovRaxImm(out, 0)
			},
			want: []byte{0x31, 0xc0},
		},
		{
			name: "load secondary displacement",
			emit: func(out *renvoAsm) {
				renvoAmd64AsmLoadRaxMemRdxDisp(out, 8)
			},
			want: []byte{0x48, 0x8b, 0x42, 0x08},
		},
		{
			name: "store primary displacement",
			emit: func(out *renvoAsm) {
				renvoAmd64AsmStoreRaxMemRdxDisp(out, 256)
			},
			want: []byte{0x48, 0x89, 0x82, 0x00, 0x01, 0x00, 0x00},
		},
		{
			name: "compare primary with zero",
			emit: func(out *renvoAsm) {
				renvoAmd64AsmCmpRaxImm8(out, 0)
			},
			want: []byte{0x48, 0x85, 0xc0},
		},
	}
	for _, test := range tests {
		var assembly renvoAsm
		test.emit(&assembly)
		if !bytes.Equal(assembly.code, test.want) {
			t.Errorf("%s = % x, want % x", test.name, assembly.code, test.want)
		}
	}
}

func TestGeneratedAmd64BSSAddressRecordsRelocation(t *testing.T) {
	var assembly renvoAsm
	renvoAmd64AsmMovR10BssAddr(&assembly, 17)
	if !bytes.Equal(assembly.code, []byte{0x4c, 0x8d, 0x15, 0, 0, 0, 0}) {
		t.Fatalf("address bytes = % x", assembly.code)
	}
	if len(assembly.absRelocs) != 3 || assembly.absRelocs[0] != 3 ||
		assembly.absRelocs[1] != 17 || assembly.absRelocs[2] != renvoAbsBssReloc {
		t.Fatalf("address relocations = %#v", assembly.absRelocs)
	}
}
