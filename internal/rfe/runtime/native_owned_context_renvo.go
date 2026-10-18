//go:build renvo && !browser

package runtime

func (n *Native) AdmitCodeVersionedContext(m *MemoryContext) bool { return false }
