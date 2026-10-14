package systemmemory

import "testing"

func TestReclaimPreservesOtherArenaExecutableAliases(t *testing.T) {
	domain := 0
	m := New(make([]byte, 3*4096), func() bool { return true }, func(a uint64, execute bool) (uint64, bool, error) { return a, true, nil })
	m.Domain = func() int { return domain }
	m.MultipleArenas = true
	// Both arenas translated the same physical instruction page.
	if _, err := m.CodeVersion(0); err != nil {
		t.Fatal(err)
	}
	domain = 1
	before, err := m.CodeVersion(0)
	if err != nil {
		t.Fatal(err)
	}
	m.MarkPageTable(8192)
	domain = 0
	m.ReclaimCode()
	domain = 1
	m.NativeRefill(0)
	if m.NativeContext().Pages[0].Permissions&4 == 0 {
		t.Fatal("user arena reclamation unguarded surviving kernel code alias")
	}
	m.BeforeWrite(0, 1)
	after, err := m.CodeVersion(0)
	if err != nil || before == after {
		t.Fatal("physical alias write did not invalidate surviving kernel translation", err)
	}
	// Closing the final code owner releases code barriers, not page-table ones.
	m.ReclaimCode()
	m.NativeRefill(0)
	if m.NativeContext().Pages[0].Permissions&4 != 0 {
		t.Fatal("code barrier survived final arena reclamation")
	}
	m.NativeRefill(8192)
	if m.NativeContext().Pages[2].Permissions&4 == 0 {
		t.Fatal("arena reclamation dropped page-table barrier")
	}
}

func TestMappedAndPhysicalKernelShareReclaimOwner(t *testing.T) {
	domain := 0
	m := New(make([]byte, 4096), func() bool { return true }, func(a uint64, execute bool) (uint64, bool, error) { return a, true, nil })
	m.Domain = func() int { return domain }
	m.MultipleArenas = true
	for _, d := range []int{0, 1, 2} {
		domain = d
		if _, err := m.CodeVersion(0); err != nil {
			t.Fatal(err)
		}
	}
	domain = 2
	m.ReclaimCode()
	domain = 0
	m.NativeRefill(0)
	if m.NativeContext().Pages[0].Permissions&4 == 0 {
		t.Fatal("kernel reclamation dropped surviving user barrier")
	}
	m.ReclaimCode()
	m.NativeRefill(0)
	if m.NativeContext().Pages[0].Permissions&4 != 0 {
		t.Fatal("mapped kernel proof survived closing the shared kernel arena")
	}
}
