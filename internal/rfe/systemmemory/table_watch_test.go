package systemmemory

import "testing"

func TestFullTLBFlushClearsOnlyTableWatchesAndRevokesNativeDescriptors(t *testing.T) {
	m := New(make([]byte, 3*4096), func() bool { return true }, func(a uint64, exec bool) (uint64, bool, error) { return a, true, nil })
	m.MarkPageTable(0)
	m.MarkPageTable(4096)
	code, err := m.CodeVersion(4096)
	if err != nil {
		t.Fatal(err)
	}
	m.NativeRefill(0)
	ctx := m.NativeContext()
	if ctx.Pages[0].Permissions&4 == 0 {
		t.Fatal("live table writable without barrier")
	}
	before := m.CodeContext()
	m.ClearPageTableWatches()
	if ctx.Pages[0].Data != nil || m.CodeContext() == before {
		t.Fatal("old native authority survived watch retirement")
	}
	before = m.CodeContext()
	m.BeforeWrite(0, 1)
	if m.CodeContext() != before {
		t.Fatal("retired table still invalidated code")
	}
	m.NativeRefill(0)
	if ctx.Pages[0].Permissions&4 != 0 {
		t.Fatal("ordinary reused RAM still write-protected")
	}
	m.NativeRefill(4096)
	if ctx.Pages[1].Permissions&4 == 0 {
		t.Fatal("executable alias lost protection")
	}
	m.BeforeWrite(4096, 1)
	if after, err := m.CodeVersion(4096); err != nil || after == code {
		t.Fatal("surviving code proof not revoked on mutation", err)
	}
	m.MarkPageTable(0)
	m.NativeRefill(0)
	if ctx.Pages[0].Permissions&4 == 0 {
		t.Fatal("reused active table did not regain barrier")
	}
}
