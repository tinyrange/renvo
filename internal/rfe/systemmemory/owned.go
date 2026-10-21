package systemmemory

import "renvo.dev/internal/rfeabi"

// EnableOwnedMemory installs exact shared physical watches and scalable native
// mapping tables. The owner must report every non-native mutation before write.
func (m *RAM) EnableOwnedMemory() { m.ownedWatches = rfeabi.NewWriteWatches(len(m.Data)) }
func (m *RAM) MarkCodeBytes(offset uint64, n int) {
	if m.ownedWatches != nil {
		flags := byte(1)
		if m.proofs != nil {
			flags = 4
		}
		m.ownedWatches.Mark(offset, n, flags)
	}
	for at := offset; at < offset+uint64(n); at++ {
		m.MarkCode(at)
	}
}
func (m *RAM) MarkPageTableEntry(offset uint64, n int) {
	if m.ownedWatches != nil {
		m.ownedWatches.Mark(offset, n, 2)
	}
	m.MarkPageTable(offset)
}
