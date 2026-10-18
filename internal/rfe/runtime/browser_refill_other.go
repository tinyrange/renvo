//go:build !renvo || !browser

package runtime

func (n *Native) BeginFastRefill(m *MemoryContext, refill func(*MemoryContext) bool) bool {
	return false
}
func (n *Native) EndFastRefill() {}
