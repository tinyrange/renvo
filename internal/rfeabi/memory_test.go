package rfeabi

import "testing"

func TestOwnedMemoryCollisionGrowthRemovalAndSharedPhysicalWatches(t *testing.T) {
	ram := make([]byte, 512*4096)
	clock := uint64(1)
	watch := NewWriteWatches(len(ram))
	m := NewOwnedMemory(ram, &clock, watch)
	for i := uint64(0); i < 512; i++ {
		m.Fill(1+i*64, i*4096, 3)
	}
	for i := uint64(0); i < 512; i++ {
		p, ok := m.Lookup(1 + i*64)
		if !ok || p.Data != (*[4096]byte)(ram[i*4096:(i+1)*4096]) {
			t.Fatal("collision evicted live mapping", i)
		}
	}
	for i := uint64(0); i < 512; i += 2 {
		m.Forget(1 + i*64)
	}
	for i := uint64(0); i < 512; i++ {
		_, ok := m.Lookup(1 + i*64)
		if ok != (i&1 != 0) {
			t.Fatal("cluster deletion", i, ok)
		}
	}
	watch.Mark(4095, 2, 1)
	watch.Mark(8192, 4, 2)
	if !watch.Intersects(4094, 4, 1) || watch.Intersects(8191, 1, 2) || !watch.Intersects(8193, 1, 2) {
		t.Fatal("physical byte mask")
	}
	watch.Clear(1)
	if watch.Intersects(4095, 2, 1) || !watch.Intersects(8192, 4, 2) {
		t.Fatal("independent code/table watches")
	}
}
