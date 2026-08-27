package rtg

import (
	"strings"
	"testing"
)

func TestRP2ExternalDefinitions(t *testing.T) {
	for _, test := range []struct {
		filename string
		target   string
		kind     string
	}{
		{"../../backends/rp2.rtg", "rp2/thumb", "uf2"},
		{"../../backends/rp2_debug.rtg", "rp2-debug/thumb", "elf"},
	} {
		resolved := Resolve(parseDefinitionFile(t, test.filename))
		if !resolved.Ok {
			t.Fatalf("resolve %s: %#v", test.filename, resolved.Diagnostics)
		}
		if len(resolved.Targets) != 1 {
			t.Fatalf("%s target count = %d, want one", test.filename, len(resolved.Targets))
		}
		target := resolved.Targets[0]
		if target.Descriptor.Name != test.target || target.Descriptor.OS != "rp2" ||
			target.Descriptor.ISA != "thumb" || target.Descriptor.WordBits != 32 ||
			target.Descriptor.PointerBits != 32 || target.Descriptor.OutputKind != test.kind {
			t.Fatalf("unexpected target descriptor: %#v", target.Descriptor)
		}
		generated := GeneratePreparedBackend(resolved, target.Descriptor.Name)
		if !generated.Ok {
			t.Fatalf("generate %s: %#v", test.target, generated.Diagnostics)
		}
		source := string(generated.Source)
		for _, contract := range []string{
			"PackagePatchRelocations(",
			"PackagePatchAbsolute(",
			"RTGResolveAbsoluteRelocation(",
			"WordBits:32,PointerBits:32",
		} {
			if !strings.Contains(source, contract) {
				t.Errorf("%s prepared backend omitted %q", test.target, contract)
			}
		}
	}
}
