package binary

import (
	"bytes"
	host "encoding/binary"
	"testing"
)

func TestAppendOrdersPreservePrefixesAndGrowth(t *testing.T) {
	for _, capacity := range []int{0, 1, 7, 32} {
		for _, value := range []uint64{0, 1, 0x0123456789abcdef, ^uint64(0)} {
			initial := make([]byte, 0, capacity)
			initial = append(initial, 0x5a)
			big := BigEndian.AppendUint16(initial, uint16(value))
			big = BigEndian.AppendUint32(big, uint32(value))
			big = BigEndian.AppendUint64(big, value)
			want := host.BigEndian.AppendUint16([]byte{0x5a}, uint16(value))
			want = host.BigEndian.AppendUint32(want, uint32(value))
			want = host.BigEndian.AppendUint64(want, value)
			if !bytes.Equal(big, want) {
				t.Fatalf("big append cap=%d value=%x: %x != %x", capacity, value, big, want)
			}
			little := LittleEndian.AppendUint16(initial, uint16(value))
			little = LittleEndian.AppendUint32(little, uint32(value))
			little = LittleEndian.AppendUint64(little, value)
			want = host.LittleEndian.AppendUint16([]byte{0x5a}, uint16(value))
			want = host.LittleEndian.AppendUint32(want, uint32(value))
			want = host.LittleEndian.AppendUint64(want, value)
			if !bytes.Equal(little, want) {
				t.Fatalf("little append cap=%d value=%x: %x != %x", capacity, value, little, want)
			}
		}
	}
}
