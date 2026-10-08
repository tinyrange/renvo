package linuxuser

import "fmt"

func (m *Memory) ReadPair(address uint64, size int) (uint64, uint64, error) {
	if size != 4 && size != 8 {
		return 0, 0, fmt.Errorf("invalid pair size")
	}
	if validRange(address, uint64(size)*2) && address%PageSize+uint64(size)*2 <= PageSize {
		p := m.lookupPage(address / PageSize)
		if p == nil || p.permissions&ReadPermission == 0 {
			return 0, 0, fmt.Errorf("unmapped or forbidden access at %#x", address)
		}
		offset := address % PageSize
		return pageRead(p, offset, size), pageRead(p, offset+uint64(size), size), nil
	}
	if err := m.Check(address, uint64(size)*2, ReadPermission); err != nil {
		return 0, 0, err
	}
	first := m.readChecked(address, size)
	second := m.readChecked(address+uint64(size), size)
	return first, second, nil
}
func (m *Memory) WritePair(address uint64, size int, first, second uint64) error {
	if size != 4 && size != 8 {
		return fmt.Errorf("invalid pair size")
	}
	if validRange(address, uint64(size)*2) && address%PageSize+uint64(size)*2 <= PageSize {
		p := m.lookupPage(address / PageSize)
		if p == nil || p.permissions&WritePermission == 0 {
			return fmt.Errorf("unmapped or forbidden access at %#x", address)
		}
		version, err := m.nextVersion()
		if err != nil {
			return err
		}
		m.markWritten(p, version)
		offset := address % PageSize
		pageWrite(p, offset, size, first)
		pageWrite(p, offset+uint64(size), size, second)
		return nil
	}
	if err := m.Check(address, uint64(size)*2, WritePermission); err != nil {
		return err
	}
	// Commit once, including version allocation, so even generation exhaustion
	// cannot leave the first half of a pair written on an error.
	var data [16]byte
	for i := 0; i < size; i++ {
		data[i] = byte(first >> (i * 8))
		data[size+i] = byte(second >> (i * 8))
	}
	return m.commit(address, data[:size*2])
}
