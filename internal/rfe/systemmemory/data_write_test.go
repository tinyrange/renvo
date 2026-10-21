package systemmemory

import "testing"

func TestProvenDataWriteRetainsCodeProofButNotTableAuthority(t *testing.T) {
	m := New(make([]byte, 8192), func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	proof, err := m.CodeVersion(0)
	if err != nil {
		t.Fatal(err)
	}
	m.NativeRefill(0)
	ctx := m.NativeContext()
	if ctx.Pages[0].Permissions&4 == 0 {
		t.Fatal("mixed page lost native write barrier")
	}
	before := m.CodeContext()
	m.BeforeNonCodeWrite(1024, 8)
	after, err := m.CodeVersion(0)
	if err != nil || before != m.CodeContext() || after != proof {
		t.Fatal("non-code data changed executable proof")
	}
	m.MarkPageTable(0)
	m.NativeRefill(4096)
	before = m.CodeContext()
	m.BeforeNonCodeWrite(1024, 8)
	if before == m.CodeContext() || ctx.Pages[1].Data != nil {
		t.Fatal("page-table mutation retained native mapping authority")
	}
	after, err = m.CodeVersion(0)
	if err != nil || after != proof {
		t.Fatal("non-code table bytes changed executable proof")
	}
}
