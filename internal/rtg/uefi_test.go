package rtg

import (
	"strings"
	"testing"
)

func TestUEFIAMD64ExternalDefinition(t *testing.T) {
	const filename = "../../backends/uefi_amd64.rtg"
	resolved := Resolve(parseDefinitionFile(t, filename))
	if !resolved.Ok {
		t.Fatalf("resolve %s: %#v", filename, resolved.Diagnostics)
	}
	if len(resolved.Targets) != 1 {
		t.Fatalf("target count = %d, want one", len(resolved.Targets))
	}
	target := resolved.Targets[0]
	if target.Descriptor.Name != "uefi/amd64" || target.Descriptor.OS != "uefi" ||
		target.Descriptor.ISA != "amd64" || target.Descriptor.OutputKind != "uefi-pe" {
		t.Fatalf("unexpected target descriptor: %#v", target.Descriptor)
	}
	generated := GeneratePreparedBackend(resolved, target.Descriptor.Name)
	if !generated.Ok {
		t.Fatalf("generate prepared backend: %#v", generated.Diagnostics)
	}
	for _, want := range []string{"PackageUefiImage(", "PackageUefiEntryStart("} {
		if !strings.Contains(string(generated.Source), want) {
			t.Errorf("prepared backend omitted %q", want)
		}
	}
}
