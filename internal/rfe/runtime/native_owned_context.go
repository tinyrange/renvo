//go:build !renvo

package runtime

// AdmitCodeVersionedContext is a host-only capability for an owner implementing
// the same disjoint-RAM/code-generation contract as CompileCodeVersionedLoop.
// It does not admit an executable address or alter per-call pointer/permission
// checks. Root-specific contexts of the same checked owner remain distinct.
func (n *Native) AdmitCodeVersionedContext(m *MemoryContext) bool {
	if m == nil {
		return false
	}
	if n.codeVersionedContexts == nil {
		n.codeVersionedContexts = make(map[*MemoryContext]bool)
	}
	if !n.codeVersionedContexts[m] && len(n.codeVersionedContexts) >= 64 {
		return false
	}
	n.codeVersionedContexts[m] = true
	return true
}
