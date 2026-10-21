package systemmemory

import (
	"fmt"
	"renvo.dev/internal/rfe/engine"
	"testing"
)

func TestCodeProofRevalidationChecksBytesPermissionsAndNewPhysicalAliases(t *testing.T) {
	frame := uint64(0)
	execute := true
	data := make([]byte, 8192)
	copy(data, []byte{1, 2, 3, 4})
	copy(data[4096:], data[:4])
	m := New(data, func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) {
		if x && !execute {
			return 0, false, fmt.Errorf("execute denied")
		}
		return frame + (a & 4095), true, nil
	})
	m.EnableOwnedMemory()
	m.EnableCodeProofs()
	p := m.ProveCode([]engine.CodeRange{{0x1000, 0x1004}}, m.CodeMutation())
	if p == nil {
		t.Fatal("initial proof")
	}
	m.InvalidateMappings()
	if p.Valid() {
		t.Fatal("full mapping flush left valid proof")
	}
	if !p.Revalidate() {
		t.Fatal("identical code required recompilation")
	}
	frame = 4096
	m.InvalidatePage(0x1000)
	if !p.Revalidate() {
		t.Fatal("identical mapped frame could not reuse body")
	}
	m.BeforeWrite(0, 1)
	data[0] = 9
	if !p.Valid() {
		t.Fatal("old physical alias still owned proof")
	}
	m.BeforeWrite(4096, 1)
	data[4096] = 9
	if p.Valid() || p.Revalidate() {
		t.Fatal("changed opcode reused old body")
	}
	data[4096] = 1
	if !p.Revalidate() {
		t.Fatal("restored identical bytes could not be checked")
	}
	execute = false
	m.InvalidatePage(0x1000)
	if p.Revalidate() {
		t.Fatal("execute revocation was skipped")
	}
	execute = true
	if !p.Revalidate() {
		t.Fatal("restored execute mapping")
	}
	p.Close()
	if p.Revalidate() || len(m.proofs.live) != 0 {
		t.Fatal("released proof resurrected")
	}
}
