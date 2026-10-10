package linuxuser

import "fmt"

// DirectWindowSize is a virtual-address reservation, not a committed-memory
// allowance. Map still enforces MemoryLimit across the complete address space.
const DirectWindowSize uint64 = 64 << 20

type directMemory interface {
	page(uint64, uint8) *[4096]byte
	protect(uint64, uint64, uint8)
	unmap(uint64, uint64)
	window(uint64) (uint64, uint64, uint64)
	close() error
}

// NewDirectMemory opts into fault-assisted native data accesses. Unsupported
// hosts return an error; NewMemory remains the portable checked implementation.
func NewDirectMemory() (*Memory, error) {
	direct, err := newDirectMemory()
	if err != nil {
		return nil, err
	}
	m := NewMemory()
	m.versions.direct = direct
	return m, nil
}

func (m *Memory) selectDirect(address uint64) {
	if m.versions.direct == nil {
		return
	}
	guest, host, size := m.versions.direct.window(address)
	m.versions.native.DirectGuest, m.versions.native.DirectHost, m.versions.native.DirectSize = guest, host, size
}
func (m *Memory) clearDirect() {
	m.versions.native.DirectGuest, m.versions.native.DirectHost, m.versions.native.DirectSize = 0, 0, 0
}

// Close invalidates all aliases and descriptors before releasing host views.
// Like Map/Protect/Unmap and native calls, it requires serialized ownership.
func (m *Memory) Close() error {
	if m.versions.closed {
		return nil
	}
	m.versions.closed = true
	m.clearDirect()
	m.versions.native.ClearLinks()
	for number := range m.pages {
		m.forgetPage(number)
		delete(m.pages, number)
	}
	if m.versions.direct != nil {
		return m.versions.direct.close()
	}
	return nil
}
func directUnavailable() error {
	return fmt.Errorf("direct guest mappings require Linux/amd64 with cgo fault recovery")
}

// Close releases this process's guest memory after its engines have stopped.
func (p *Process) Close() error { return p.Memory.Close() }
