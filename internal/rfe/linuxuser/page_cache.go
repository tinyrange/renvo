package linuxuser

// This is a mapping lookup cache, not an access/permission cache. Every caller
// still checks current permissions on the returned page. Its owner is shared by
// memory aliases, so Unmap cannot leave a detached page reachable via a copy.
type pageCacheEntry struct {
	number uint64
	page   *page
}

func (m *Memory) lookupPage(number uint64) *page {
	entry := &m.versions.pageCache[number&63]
	if entry.page != nil && entry.number == number {
		m.fillNative(number, entry.page)
		return entry.page
	}
	p := m.pages[number]
	if p != nil {
		*entry = pageCacheEntry{number, p}
		m.fillNative(number, p)
	}
	return p
}

func (m *Memory) forgetPage(number uint64) {
	m.versions.native.Forget(number)
	entry := &m.versions.pageCache[number&63]
	if entry.number == number {
		*entry = pageCacheEntry{}
	}
}
