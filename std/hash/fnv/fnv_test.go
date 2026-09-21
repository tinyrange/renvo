package fnv

import "testing"

func TestFNVVectors(t *testing.T) {
	a := New32a()
	a.Write([]byte("he"))
	a.Write([]byte("llo"))
	if a.Sum32() != 0x4f9f2cab {
		t.Fatal("FNV-1a32")
	}
	sum := a.Sum([]byte{7})
	if len(sum) != 5 || sum[0] != 7 || sum[1] != 0x4f || sum[4] != 0xab {
		t.Fatal("sum encoding")
	}
	a.Reset()
	if a.Sum32() != 2166136261 {
		t.Fatal("reset")
	}
	b := New32()
	b.Write([]byte("hello"))
	if b.Sum32() != 0xb6fa7167 {
		t.Fatal("FNV-1 32")
	}
	c := New64a()
	c.Write([]byte("hello"))
	if c.Sum64() != 0xa430d84680aabd0b {
		t.Fatal("FNV-1a 64")
	}
	d := New64()
	d.Write([]byte("hello"))
	if d.Sum64() != 0x7b495389bdbdd4c7 {
		t.Fatal("FNV-1 64")
	}
}
