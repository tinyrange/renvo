package rtg

import (
	"bytes"
	"testing"
)

func TestGenerationIndicesDoNotRetainMutableDocuments(t *testing.T) {
	resolved := resolveNativeTarget(t, "linux/amd64")
	arch := resolved.Targets[0].Arch.Name
	unit := resolved.Document.Unit
	before := GenerateArchitectureBackend(resolved, arch, "generated")
	if !before.Ok {
		t.Fatalf("baseline generation: %#v", before.Diagnostics)
	}
	oldSymbol := "rtg" + exportedName(unit) + "RAX"
	if !bytes.Contains(before.Source, []byte(oldSymbol)) {
		t.Fatalf("missing baseline symbol %q", oldSymbol)
	}
	resolved.Document.Unit = "changed_generation_unit"
	after := GenerateArchitectureBackend(resolved, arch, "generated")
	if !after.Ok {
		t.Fatalf("edited generation: %#v", after.Diagnostics)
	}
	newSymbol := "rtg" + exportedName(resolved.Document.Unit) + "RAX"
	if !bytes.Contains(after.Source, []byte(newSymbol)) || bytes.Contains(after.Source, []byte(oldSymbol)) {
		t.Fatal("generation reused symbols from the earlier document")
	}
	resolved.Document.Unit = unit
	again := GenerateArchitectureBackend(resolved, arch, "generated")
	if !again.Ok || !bytes.Equal(before.Source, again.Source) {
		t.Fatal("generation retained edits across calls")
	}
}
