package targetinfo

import (
	"reflect"
	"testing"
)

func TestDescriptorRecordPrimitives(t *testing.T) {
	// Exercise byte order, the 32-bit host limit, embedded NUL/non-ASCII
	// bytes, and the nil versus empty slice distinction in generated records.
	r := descriptorReader{data: "\x78\x56\x34\x12\xff\xff\xff\x7f\x04\x00\x00\x00a\x00\xc3\xa9\x00\x00\x00\x00\x01\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00"}
	if got := r.number(); got != 0x12345678 {
		t.Fatalf("byte order: %x", got)
	}
	if got := r.number(); got != 0x7fffffff {
		t.Fatalf("32-bit value: %x", got)
	}
	if got := r.text(); got != "a\x00é" {
		t.Fatalf("binary text: %q", got)
	}
	if got := r.list(); got != nil {
		t.Fatalf("nil list: %#v", got)
	}
	if got := r.list(); got == nil || len(got) != 0 {
		t.Fatalf("empty list: %#v", got)
	}
	if got := r.list(); !reflect.DeepEqual(got, []string{""}) {
		t.Fatalf("empty element: %#v", got)
	}
	if r.pos != len(r.data) {
		t.Fatalf("consumed %d of %d bytes", r.pos, len(r.data))
	}
}

func TestDescriptorIndexBounds(t *testing.T) {
	for _, index := range []int{-1, descriptorCount, descriptorCount + 1} {
		if got := descriptorAt(index); !reflect.DeepEqual(got, Descriptor{}) {
			t.Fatalf("descriptorAt(%d) = %#v", index, got)
		}
	}
}
