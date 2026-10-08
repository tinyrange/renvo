package linuxuser

import "testing"

func TestScalarAndBufferMemoryBoundaries(t *testing.T) {
	m := NewMemory()
	if err := m.Map(0x1000, 3*PageSize, 7); err != nil {
		t.Fatal(err)
	}
	original := make([]byte, 3*int(PageSize))
	for i := range original {
		original[i] = byte(i*37 + i/13)
		m.pages[1+uint64(i)/PageSize].data[uint64(i)%PageSize] = original[i]
	}
	offsets := []int{0, 1, 2, 3, 7, 15, 4088, 4089, 4090, 4091, 4092, 4093, 4094, 4095, 4096, 4097, 8188, 8191}
	for _, size := range []int{1, 2, 4, 8} {
		for _, offset := range offsets {
			for _, execute := range []bool{false, true} {
				want := uint64(0)
				for j := 0; j < size; j++ {
					want |= uint64(original[offset+j]) << uint(j*8)
				}
				got, err := m.Read(0x1000+uint64(offset), size, execute)
				if err != nil || got != want {
					t.Fatalf("size=%d offset=%d execute=%v got=%x want=%x err=%v", size, offset, execute, got, want, err)
				}
			}
		}
	}
	for _, size := range []int{1, 2, 4, 8} {
		for _, offset := range offsets {
			if err := m.WriteBytes(0x1000, original); err != nil {
				t.Fatal(err)
			}
			value := uint64(0xfedcba9876543210) ^ uint64(offset)
			want := append([]byte(nil), original...)
			for j := 0; j < size; j++ {
				want[offset+j] = byte(value >> uint(j*8))
			}
			if err := m.Write(0x1000+uint64(offset), size, value); err != nil {
				t.Fatal(err)
			}
			got, err := m.ReadBytes(0x1000, uint64(len(want)))
			if err != nil || string(got) != string(want) {
				t.Fatalf("scalar store damaged bytes size=%d offset=%d err=%v", size, offset, err)
			}
		}
	}
	for _, offset := range []int{0, 3, 4095} {
		for _, length := range []int{0, 1, 17, 4096, 4103} {
			if err := m.WriteBytes(0x1000, original); err != nil {
				t.Fatal(err)
			}
			payload := make([]byte, length)
			for i := range payload {
				payload[i] = byte(i * 11)
			}
			want := append([]byte(nil), original...)
			copy(want[offset:], payload)
			if err := m.WriteBytes(0x1000+uint64(offset), payload); err != nil {
				t.Fatal(err)
			}
			got, err := m.ReadBytes(0x1000, uint64(len(want)))
			if err != nil || string(got) != string(want) {
				t.Fatal("page-chunk buffer copy")
			}
		}
	}
	if err := m.Protect(0x2000, PageSize, ReadPermission); err != nil {
		t.Fatal(err)
	}
	before, _ := m.ReadBytes(0x1ff8, 16)
	stamp := m.CodeContext()
	for _, write := range []func() error{
		func() error { return m.Write(0x1ffc, 8, 0) },
		func() error { return m.WritePair(0x1ff8, 8, 0, 0) },
		func() error { return m.WriteBytes(0x1fff, []byte{0, 0}) },
	} {
		if err := write(); err == nil {
			t.Fatal("cross-page write ignored permission")
		}
		after, _ := m.ReadBytes(0x1ff8, 16)
		if string(after) != string(before) || m.CodeContext() != stamp {
			t.Fatal("failed access partially committed")
		}
	}
	if _, err := m.Read(0x1ffc, 8, true); err == nil {
		t.Fatal("execute permission checked only on first page")
	}
	if err := m.Unmap(0x2000, PageSize); err != nil {
		t.Fatal(err)
	}
	if _, _, err := m.ReadPair(0x1ff8, 8); err == nil {
		t.Fatal("pair read crossed missing page")
	}
	for _, size := range []int{-1, 0, 3, 16} {
		if _, err := m.Read(0x1000, size, false); err == nil {
			t.Fatal("invalid size read")
		}
		if err := m.Write(0x1000, size, 0); err == nil {
			t.Fatal("invalid size write")
		}
	}
	if _, err := m.Read(AddressLimit-4, 8, false); err == nil {
		t.Fatal("address range overflow")
	}
}
