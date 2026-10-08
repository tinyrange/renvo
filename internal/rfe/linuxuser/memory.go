package linuxuser

import (
	"encoding/binary"
	"fmt"
	"renvo.dev/internal/rfe/engine"
)

const (
	PageSize          uint64 = 4096
	AddressLimit      uint64 = 1 << 48
	MemoryLimit       uint64 = 64 << 20
	ReadPermission    uint8  = 1
	WritePermission   uint8  = 2
	ExecutePermission uint8  = 4
)

type page struct {
	data        [4096]byte
	permissions uint8
	epoch       uint64
}
type Memory struct {
	pages    map[uint64]*page
	versions *memoryVersions
}

func NewMemory() *Memory {
	return &Memory{pages: map[uint64]*page{}, versions: &memoryVersions{identity: &engine.CodeIdentity{}}}
}
func validRange(address, length uint64) bool {
	return address < AddressLimit && length <= AddressLimit-address
}
func pageRange(address, length uint64) bool {
	return length != 0 && address%PageSize == 0 && length%PageSize == 0 && validRange(address, length)
}

func (m *Memory) Map(address, length uint64, permissions uint8) error {
	if !pageRange(address, length) || permissions&^uint8(7) != 0 || length > MemoryLimit || uint64(len(m.pages))*PageSize > MemoryLimit-length {
		return fmt.Errorf("invalid or over-budget mapping")
	}
	for at := address; at < address+length; at += PageSize {
		if m.pages[at/PageSize] != nil {
			return fmt.Errorf("mapping overlaps at %#x", at)
		}
	}
	version, err := m.nextVersion()
	if err != nil {
		return err
	}
	for at := address; at < address+length; at += PageSize {
		m.pages[at/PageSize] = &page{permissions: permissions, epoch: version}
	}
	if permissions&ExecutePermission != 0 {
		m.versions.codeEpoch = version
	}
	return nil
}
func (m *Memory) Protect(address, length uint64, permissions uint8) error {
	if !pageRange(address, length) || permissions&^uint8(7) != 0 || length > MemoryLimit {
		return fmt.Errorf("invalid protection range")
	}
	for at := address; at < address+length; at += PageSize {
		if m.pages[at/PageSize] == nil {
			return fmt.Errorf("unmapped protection range")
		}
	}
	version, err := m.nextVersion()
	if err != nil {
		return err
	}
	for at := address; at < address+length; at += PageSize {
		if (m.pages[at/PageSize].permissions|permissions)&ExecutePermission != 0 {
			m.versions.codeEpoch = version
		}
		m.versions.native.Forget(at / PageSize)
		m.pages[at/PageSize].permissions = permissions
		m.pages[at/PageSize].epoch = version
	}
	return nil
}
func (m *Memory) Unmap(address, length uint64) error {
	if !pageRange(address, length) || length > MemoryLimit {
		return fmt.Errorf("invalid unmap range")
	}
	executable := false
	for at := address; at < address+length; at += PageSize {
		if p := m.lookupPage(at / PageSize); p != nil && p.permissions&ExecutePermission != 0 {
			executable = true
		}
	}
	if executable {
		version, err := m.nextVersion()
		if err != nil {
			return err
		}
		m.versions.codeEpoch = version
	}
	for at := address; at < address+length; at += PageSize {
		m.forgetPage(at / PageSize)
		delete(m.pages, at/PageSize)
	}
	return nil
}
func (m *Memory) Check(address, length uint64, permission uint8) error {
	if !validRange(address, length) || length > MemoryLimit {
		return fmt.Errorf("invalid access at %#x", address)
	}
	if length == 0 {
		return nil
	}
	end := address + length
	for at := address; at < end; {
		p := m.lookupPage(at / PageSize)
		if p == nil || p.permissions&permission != permission {
			return fmt.Errorf("unmapped or forbidden access at %#x", at)
		}
		at = (at/PageSize + 1) * PageSize
	}
	return nil
}

// scalarPage checks the complete scalar access. A nil page with nil error
// means a cross-page access; the caller then uses the full-range slow path.
func (m *Memory) scalarPage(address uint64, size int, permission uint8) (*page, error) {
	if size != 1 && size != 2 && size != 4 && size != 8 {
		return nil, fmt.Errorf("invalid access size")
	}
	if !validRange(address, uint64(size)) {
		return nil, fmt.Errorf("invalid access at %#x", address)
	}
	if address%PageSize+uint64(size) > PageSize {
		return nil, nil
	}
	p := m.lookupPage(address / PageSize)
	if p == nil || p.permissions&permission != permission {
		return nil, fmt.Errorf("unmapped or forbidden access at %#x", address)
	}
	return p, nil
}
func pageRead(p *page, offset uint64, size int) uint64 {
	switch size {
	case 1:
		return uint64(p.data[offset])
	case 2:
		return uint64(binary.LittleEndian.Uint16(p.data[offset : offset+2]))
	case 4:
		return uint64(binary.LittleEndian.Uint32(p.data[offset : offset+4]))
	default:
		return binary.LittleEndian.Uint64(p.data[offset : offset+8])
	}
}
func pageWrite(p *page, offset uint64, size int, value uint64) {
	switch size {
	case 1:
		p.data[offset] = byte(value)
	case 2:
		binary.LittleEndian.PutUint16(p.data[offset:offset+2], uint16(value))
	case 4:
		binary.LittleEndian.PutUint32(p.data[offset:offset+4], uint32(value))
	default:
		binary.LittleEndian.PutUint64(p.data[offset:offset+8], value)
	}
}

// readChecked is called only after a complete permission/range precheck.
// Each page is looked up once, not once for each byte.
func (m *Memory) readChecked(address uint64, size int) uint64 {
	var value uint64
	for done := 0; done < size; {
		offset := address % PageSize
		count := size - done
		if uint64(count) > PageSize-offset {
			count = int(PageSize - offset)
		}
		p := m.lookupPage(address / PageSize)
		for i := 0; i < count; i++ {
			value |= uint64(p.data[offset+uint64(i)]) << uint((done+i)*8)
		}
		done += count
		address += uint64(count)
	}
	return value
}
func (m *Memory) Read(address uint64, size int, execute bool) (uint64, error) {
	permission := ReadPermission
	if execute {
		permission = ExecutePermission
	}
	p, err := m.scalarPage(address, size, permission)
	if err != nil {
		return 0, err
	}
	if p != nil {
		return pageRead(p, address%PageSize, size), nil
	}
	if err := m.Check(address, uint64(size), permission); err != nil {
		return 0, err
	}
	return m.readChecked(address, size), nil
}
func (m *Memory) Write(address uint64, size int, value uint64) error {
	p, err := m.scalarPage(address, size, WritePermission)
	if err != nil {
		return err
	}
	if p != nil {
		version, err := m.nextVersion()
		if err != nil {
			return err
		}
		m.markWritten(p, version)
		pageWrite(p, address%PageSize, size, value)
		return nil
	}
	if err := m.Check(address, uint64(size), WritePermission); err != nil {
		return err
	}
	var data [8]byte
	for i := 0; i < size; i++ {
		data[i] = byte(value >> uint(i*8))
	}
	return m.commit(address, data[:size])
}
func (m *Memory) ReadBytes(address, length uint64) ([]byte, error) {
	if length > 1<<20 {
		return nil, fmt.Errorf("transfer exceeds budget")
	}
	if err := m.Check(address, length, ReadPermission); err != nil {
		return nil, err
	}
	out := make([]byte, int(length))
	for done := uint64(0); done < length; {
		at := address + done
		offset := at % PageSize
		count := length - done
		if count > PageSize-offset {
			count = PageSize - offset
		}
		p := m.lookupPage(at / PageSize)
		copy(out[done:done+count], p.data[offset:offset+count])
		done += count
	}
	return out, nil
}
func (m *Memory) WriteBytes(address uint64, data []byte) error {
	if err := m.Check(address, uint64(len(data)), WritePermission); err != nil {
		return err
	}
	return m.commit(address, data)
}

// copyIn is only used after complete validation, or by the ELF loader before
// exposing a new address space. Loader writes happen before any code stamps are
// observed; all writes to a live space must go through commit. This helper never
// allocates a page or changes permissions.
func (m *Memory) copyIn(address uint64, data []byte) {
	for len(data) > 0 {
		offset := address % PageSize
		count := len(data)
		if uint64(count) > PageSize-offset {
			count = int(PageSize - offset)
		}
		p := m.lookupPage(address / PageSize)
		copy(p.data[offset:offset+uint64(count)], data[:count])
		address += uint64(count)
		data = data[count:]
	}
}
func pageUp(value uint64) uint64 { return (value + PageSize - 1) &^ (PageSize - 1) }
