package systemmemory

import "fmt"

const directWindowSize = uint64(64 << 20)

// directMemory owns physical backing plus at most eight non-executable VA
// reservations. The owner is single-threaded; no mapping can change during a
// native call. Each admitted page is an exact Resolve-checked physical alias.
type directMemory interface {
	refill(address, offset, permissions uint64)
	forget()
	forgetPage(address uint64)
	window() (guest, host, size uint64)
	close() error
}

// NewDirect allocates physical RAM before exposing any slices. Unsupported
// hosts use ordinary Go RAM. Close must follow closure of all native engines;
// every borrowed Data slice becomes invalid at Close. There is deliberately no
// finalizer: a caller may retain a physical slice independently of this owner.
func NewDirect(size int, active func() bool, resolve Resolve) (*RAM, error) {
	if size <= 0 || size&4095 != 0 {
		return nil, fmt.Errorf("RAM size must be a positive page multiple")
	}
	data, direct, err := newDirectRAM(size)
	if err != nil {
		return nil, err
	}
	m := New(data, active, resolve)
	m.direct = direct
	return m, nil
}

// DirectAvailable reports host backing capability, not permission to access a
// guest address. Every view remains Resolve-checked and protection-enforced.
func (m *RAM) DirectAvailable() bool { return m != nil && m.direct != nil }

// ProfileDirectWindow is a read-only diagnostic of the selected cached view,
// not an architectural mapping probe or an access grant. It returns no host
// pointer and never changes cache selection or translation state.
func (m *RAM) ProfileDirectWindow() [2]uint64 {
	if m == nil || m.direct == nil {
		return [2]uint64{}
	}
	guest, _, size := m.direct.window()
	return [2]uint64{guest, size}
}

// Close revokes every native view before releasing physical RAM. It is an
// explicit end-of-memory lifetime boundary, not an interpreter/JIT toggle.
func (m *RAM) Close() error {
	m.disabled = true
	m.ForgetData()
	if m.direct == nil {
		return nil
	}
	err := m.direct.close()
	m.direct = nil
	m.Data = nil
	return err
}
