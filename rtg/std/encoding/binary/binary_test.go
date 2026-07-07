package binary

import "testing"

func TestByteOrders(t *testing.T) {
	buf := make([]byte, 8)
	LittleEndian.PutUint32(buf, 0x11223344)
	if buf[0] != 0x44 || LittleEndian.Uint32(buf) != 0x11223344 {
		t.Fatalf("little endian uint32 failed: %x", buf[:4])
	}
	BigEndian.PutUint64(buf, 0x0102030405060708)
	if buf[0] != 1 || BigEndian.Uint64(buf) != 0x0102030405060708 {
		t.Fatalf("big endian uint64 failed: %x", buf)
	}
}

func TestVarints(t *testing.T) {
	buf := make([]byte, 10)
	n := PutUvarint(buf, 300)
	u, used := Uvarint(buf[:n])
	if u != 300 || used != n {
		t.Fatalf("uvarint = %d/%d want 300/%d", u, used, n)
	}
	n = PutVarint(buf, -12345)
	v, used := Varint(buf[:n])
	if v != -12345 || used != n {
		t.Fatalf("varint = %d/%d want -12345/%d", v, used, n)
	}
	if got := AppendUvarint(nil, 17); len(got) != 1 || got[0] != 17 {
		t.Fatalf("AppendUvarint failed: %v", got)
	}
}
