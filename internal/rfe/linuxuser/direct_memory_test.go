//go:build !renvo && linux && amd64 && cgo

package linuxuser

import "testing"

func TestDirectMappingLifecycle(t *testing.T) {
	m, err := NewDirectMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	alias := *m
	ctx := m.NativeContext()
	// Separate windows and a ninth, checked-only window exercise sparse VA
	// fallback without changing the physical guest-memory allowance.
	for i := uint64(0); i < 9; i++ {
		a := i*DirectWindowSize + PageSize
		if err = m.Map(a, PageSize, 3); err != nil {
			t.Fatal(err)
		}
		if err = m.Write(a, 8, i+1); err != nil {
			t.Fatal(err)
		}
		if got, e := alias.Read(a, 8, false); e != nil || got != i+1 {
			t.Fatal(i, got, e)
		}
	}
	if err = m.Map(0, MemoryLimit, 3); err == nil {
		t.Fatal("budget/overlap accepted")
	}
	before := m.versions.epoch
	if err = m.Protect(PageSize, PageSize, 7); err != nil {
		t.Fatal(err)
	}
	stamp, err := m.CodeVersion(PageSize)
	if err != nil || stamp.Epoch <= before {
		t.Fatal("data-to-code generation", stamp, err)
	}
	if err = m.Write(PageSize, 8, 91); err != nil {
		t.Fatal(err)
	}
	newer, err := m.CodeVersion(PageSize)
	if err != nil || newer.Epoch <= stamp.Epoch {
		t.Fatal("executable write generation", newer, err)
	}
	if err = m.Unmap(PageSize, PageSize); err != nil {
		t.Fatal(err)
	}
	if _, err = m.Read(PageSize, 1, false); err == nil {
		t.Fatal("unmap still readable")
	}
	if err = m.Map(PageSize, PageSize, 7); err != nil {
		t.Fatal(err)
	}
	remapped, err := m.CodeVersion(PageSize)
	if err != nil || remapped.Epoch <= newer.Epoch {
		t.Fatal("remap reused generation", remapped, err)
	}
	if got, e := m.Read(PageSize, 8, false); e != nil || got != 0 {
		t.Fatal("remap leaked bytes", got, e)
	}
	if err = alias.Close(); err != nil {
		t.Fatal(err)
	}
	if ctx.DirectSize != 0 {
		t.Fatal("closed window retained")
	}
	if _, err = m.Read(PageSize, 1, false); err == nil {
		t.Fatal("copy retained closed alias")
	}
	if err = m.Map(PageSize, PageSize, 3); err == nil {
		t.Fatal("closed owner reopened")
	}
	if err = m.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestDirectUnownedGoFault(t *testing.T) {
	m, err := NewDirectMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	// Installing our signal handler must not swallow Go's recoverable nil fault.
	recovered := false
	func() {
		defer func() { recovered = recover() != nil }()
		var p *int
		*p = 1
	}()
	if !recovered {
		t.Fatal("Go fault did not chain to the previous handler")
	}
}
