//go:build !renvo

package backendcompiled

import (
	"bytes"
	"testing"
)

func TestObjectIntegerBytesRespectStorageOrder(t *testing.T) {
	for _, tc := range []struct {
		size, order int
		want        []byte
	}{
		{1, renvoEndianLittle, []byte{0x88}},
		{2, renvoEndianLittle, []byte{0x88, 0x77}},
		{3, renvoEndianLittle, []byte{0x88, 0x77, 0x66}},
		{4, renvoEndianLittle, []byte{0x88, 0x77, 0x66, 0x55}},
		{8, renvoEndianLittle, []byte{0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11}},
		{1, renvoEndianBig, []byte{0x88}},
		{2, renvoEndianBig, []byte{0x77, 0x88}},
		{3, renvoEndianBig, []byte{0x66, 0x77, 0x88}},
		{4, renvoEndianBig, []byte{0x55, 0x66, 0x77, 0x88}},
		{8, renvoEndianBig, []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}},
	} {
		data := bytes.Repeat([]byte{0xaa}, tc.size+2)
		if !renvoStoreIntegerBytes(data, 1, tc.size, 0x1122334455667788, tc.order) {
			t.Fatal("valid constant rejected")
		}
		if !bytes.Equal(data[1:len(data)-1], tc.want) || data[0] != 0xaa || data[len(data)-1] != 0xaa {
			t.Errorf("size %d order %d bytes = %x, want sentinel + %x + sentinel", tc.size, tc.order, data, tc.want)
		}
	}
	for _, tc := range []struct{ offset, size, order int }{
		{-1, 4, renvoEndianLittle}, {0, 0, renvoEndianLittle}, {0, 9, renvoEndianBig},
		{6, 4, renvoEndianLittle}, {9, 1, renvoEndianBig}, {0, 4, 0},
		{int(^uint(0) >> 1), 8, renvoEndianLittle},
	} {
		data := bytes.Repeat([]byte{0xaa}, 8)
		if renvoStoreIntegerBytes(data, tc.offset, tc.size, 0, tc.order) || !bytes.Equal(data, bytes.Repeat([]byte{0xaa}, 8)) {
			t.Errorf("invalid range/order %+v accepted or mutated data", tc)
		}
	}
	context := renvoNewCompileContext(renvoTargetLinuxAmd64, false, false, false)
	data := make([]byte, 4)
	if !renvoStoreTargetConstant(context, data, 0, 4, 0x11223344) || !bytes.Equal(data, []byte{0x44, 0x33, 0x22, 0x11}) {
		t.Fatal("selected descriptor did not reach object serializer")
	}
	context.renvoTarget = -1
	if renvoStoreTargetConstant(context, data, 0, 4, 0) {
		t.Fatal("unknown descriptor encoded object bytes")
	}
}
