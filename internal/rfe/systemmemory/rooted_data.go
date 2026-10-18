package systemmemory

// EnableRootedData is explicit single-threaded host-owner authority. The owner
// must tag BOTH full ordered translation roots and every privilege permission
// class, watch every data/execute walk's complete physical table dependencies,
// revoke descriptors before every PTE/control/permission change, and execute
// only the freshly selected domain/root context. It must not reuse an inactive
// context directly. Root selection alone does not mutate mappings.
//
// No extra page/root slots are allocated. A root slot's existing replacement,
// PTE barriers, watch changes, TLBI and global invalidation still revoke data.
// Direct VA reservations remain unrooted and keep the conservative behavior.
func (m *RAM) EnableRootedData() bool {
	if m.direct != nil || m.AddressSpacePair == nil || m.Domain == nil {
		return false
	}
	m.ForgetData()
	m.rootedData = true
	return true
}

// SelectPrivilege observes a checked domain/root transition, not a permission
// mutation. Only the explicit complete-walker owner may preserve separate old
// views. Generic owners and direct mappings retain their previous revocation.
func (m *RAM) SelectPrivilege() {
	if !m.rootedData || m.direct != nil || m.AddressSpacePair == nil || m.Domain == nil {
		m.ForgetData()
	}
}
