package systemmemory

import "testing"

func TestHardwareTableMetadataPreservesCodeProofsExceptExecutableAliases(t *testing.T) {
	m := New(make([]byte, 3*4096), func() bool { return true }, func(a uint64, exec bool) (uint64, bool, error) { return a, true, nil })
	m.MarkPageTable(0)
	m.NativeRefill(8192)
	ctx := m.NativeContext()
	before := m.CodeContext()
	m.BeforePageTableMetadata(0, 8)
	if m.CodeContext() != before || ctx.Pages[2].Data != nil {
		t.Fatal("metadata changed code proof or retained stale data authority")
	}
	code, err := m.CodeVersion(0)
	if err != nil {
		t.Fatal(err)
	}
	before = m.CodeContext()
	m.BeforePageTableMetadata(0, 8)
	if m.CodeContext() == before {
		t.Fatal("executable table alias did not invalidate")
	}
	if after, err := m.CodeVersion(0); err != nil || after == code {
		t.Fatal("metadata mutation preserved executable bytes", err)
	}
}
