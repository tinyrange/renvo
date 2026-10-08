package linuxuser

import "testing"

func TestPageCacheAliasesPermissionsRemapAndCollision(t *testing.T) {
	m := NewMemory()
	const at uint64 = 0x1000
	const collision uint64 = at + 64*PageSize
	rwx := ReadPermission | WritePermission | ExecutePermission
	if err := m.Map(at, PageSize, rwx); err != nil {
		t.Fatal(err)
	}
	if err := m.Map(collision, PageSize, rwx); err != nil {
		t.Fatal(err)
	}
	if err := m.Write(at, 8, 0x1234); err != nil {
		t.Fatal(err)
	}
	if err := m.Write(collision, 8, 0x5678); err != nil {
		t.Fatal(err)
	}
	alias := *m
	for _, address := range []uint64{at, collision, at, collision, at} {
		want := uint64(0x1234)
		if address == collision {
			want = 0x5678
		}
		if got, err := alias.Read(address, 8, false); err != nil || got != want {
			t.Fatal("collision/alias read", got, err)
		}
	}
	if err := alias.Protect(at, PageSize, WritePermission); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Read(at, 8, false); err == nil {
		t.Fatal("cached read bypassed protection")
	}
	if _, err := m.Read(at, 4, true); err == nil {
		t.Fatal("cached instruction bypassed protection")
	}
	if _, err := m.CodeVersion(at); err == nil {
		t.Fatal("cached code stamp bypassed protection")
	}
	if err := m.Protect(at, PageSize, ReadPermission); err != nil {
		t.Fatal(err)
	}
	if err := alias.Write(at, 8, 99); err == nil {
		t.Fatal("cached write bypassed protection")
	}
	// Warm through one owner; unmap through the other must discard detached data.
	if got, err := m.Read(at, 8, false); err != nil || got != 0x1234 {
		t.Fatal(got, err)
	}
	if err := alias.Unmap(at, PageSize); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Read(at, 8, false); err == nil {
		t.Fatal("detached cached page after alias unmap")
	}
	if err := alias.Map(at, PageSize, rwx); err != nil {
		t.Fatal(err)
	}
	if got, err := m.Read(at, 8, false); err != nil || got != 0 {
		t.Fatal("remap reused old contents", got, err)
	}
	// Cross-page slow paths must not turn a cached first page into a partial store.
	if err := m.Write(at+PageSize-8, 8, 0xabcdef); err != nil {
		t.Fatal(err)
	}
	if err := alias.Write(at+PageSize-4, 8, ^uint64(0)); err == nil {
		t.Fatal("cross-page write accepted unmapped second page")
	}
	if got, err := m.Read(at+PageSize-8, 8, false); err != nil || got != 0xabcdef {
		t.Fatal("partial faulted write", got, err)
	}
}
