package linuxuser

import "testing"

func TestPairFastPathBytesPermissionsVersionsAndAtomicity(t *testing.T) {
	m := NewMemory()
	if err := m.Map(4096, 8192, ReadPermission|WritePermission|ExecutePermission); err != nil {
		t.Fatal(err)
	}
	alias := *m
	for _, size := range []int{4, 8} {
		for _, offset := range []uint64{0, 1, 4079, 4080, 4087, 4088, 4095} {
			address := 4096 + offset
			before := m.versions.epoch
			if err := alias.WritePair(address, size, 0x0123456789abcdef, 0xfedcba9876543210); err != nil {
				t.Fatal(err)
			}
			if m.versions.epoch != before+1 || m.CodeContext().Epoch != before+1 {
				t.Fatal("pair did not commit one code generation")
			}
			first, second, err := m.ReadPair(address, size)
			mask := ^uint64(0)
			if size == 4 {
				mask = 0xffffffff
			}
			if err != nil || first != 0x0123456789abcdef&mask || second != 0xfedcba9876543210&mask {
				t.Fatal("pair values", size, offset, first, second, err)
			}
			data, err := m.ReadBytes(address, uint64(size*2))
			if err != nil {
				t.Fatal(err)
			}
			for i, value := range []uint64{0x0123456789abcdef, 0xfedcba9876543210} {
				for j := 0; j < size; j++ {
					if data[i*size+j] != byte(value>>uint(8*j)) {
						t.Fatal("pair byte order", size, offset, data)
					}
				}
			}
		}
	}
	for _, permissions := range []uint8{ReadPermission, WritePermission, 0} {
		if err := alias.Protect(4096, 4096, permissions); err != nil {
			t.Fatal(err)
		}
		_, _, readErr := m.ReadPair(4097, 8)
		before := m.versions.epoch
		writeErr := m.WritePair(4097, 8, 13, 17)
		if (readErr == nil) != (permissions&ReadPermission != 0) || (writeErr == nil) != (permissions&WritePermission != 0) {
			t.Fatal("pair permission cache", permissions, readErr, writeErr)
		}
		if writeErr != nil && m.versions.epoch != before {
			t.Fatal("failed pair allocated version")
		}
	}
	if err := m.Protect(4096, 4096, 3); err != nil {
		t.Fatal(err)
	}
	if err := m.Protect(8192, 4096, ReadPermission); err != nil {
		t.Fatal(err)
	}
	beforeBytes := m.pages[1].data
	before := m.versions.epoch
	if err := alias.WritePair(8184, 8, 1, 2); err == nil || beforeBytes != m.pages[1].data || m.versions.epoch != before {
		t.Fatal("cross-page partial pair")
	}
	m.versions.epoch = ^uint64(0)
	if err := alias.WritePair(4097, 8, 1, 2); err == nil || beforeBytes != m.pages[1].data {
		t.Fatal("same-page pair wrote before version allocation")
	}
	for _, address := range []uint64{AddressLimit - 8, AddressLimit, ^uint64(0)} {
		if _, _, err := m.ReadPair(address, 8); err == nil {
			t.Fatal("pair range bypass", address)
		}
		if err := m.WritePair(address, 8, 1, 2); err == nil {
			t.Fatal("pair write range bypass", address)
		}
	}
}
