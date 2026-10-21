package rtg

import (
	"bytes"
	"testing"
)

func TestGenerationIndicesDoNotRetainMutableDocuments(t *testing.T) {
	for _, tc := range []struct {
		name     string
		generate func(ResolveResult) GenerateResult
	}{
		{"architecture", func(r ResolveResult) GenerateResult {
			return GenerateArchitectureBackend(r, r.Targets[0].Arch.Name, "generated")
		}},
		{"prepared", func(r ResolveResult) GenerateResult {
			return GeneratePreparedBackend(r, "linux/amd64")
		}},
		{"fixed", func(r ResolveResult) GenerateResult {
			return GenerateFixedBackend(r, "linux/amd64")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resolved := resolveNativeTarget(t, "linux/amd64")
			unit := resolved.Document.Unit
			before := tc.generate(resolved)
			if !before.Ok {
				t.Fatalf("baseline generation: %#v", before.Diagnostics)
			}
			oldSymbol := "rtg" + exportedName(unit) + "RAX"
			if !bytes.Contains(before.Source, []byte(oldSymbol)) {
				t.Fatalf("missing baseline symbol %q", oldSymbol)
			}
			resolved.Document.Unit = "changed_generation_unit"
			after := tc.generate(resolved)
			if !after.Ok {
				t.Fatalf("edited generation: %#v", after.Diagnostics)
			}
			newSymbol := "rtg" + exportedName(resolved.Document.Unit) + "RAX"
			if !bytes.Contains(after.Source, []byte(newSymbol)) || bytes.Contains(after.Source, []byte(oldSymbol)) {
				t.Fatal("generation reused symbols from the earlier document")
			}
			resolved.Document.Unit = unit
			again := tc.generate(resolved)
			if !again.Ok || !bytes.Equal(before.Source, again.Source) {
				t.Fatal("generation retained edits across calls")
			}
		})
	}
}
