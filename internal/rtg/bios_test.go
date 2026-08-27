package rtg

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestBIOS8086ExternalDefinition(t *testing.T) {
	const filename = "../../backends/msdos.rtg"
	resolved := Resolve(parseDefinitionFile(t, filename))
	if !resolved.Ok {
		t.Fatalf("resolve %s: %#v", filename, resolved.Diagnostics)
	}
	var target *ResolvedTarget
	for i := range resolved.Targets {
		if resolved.Targets[i].Descriptor.Name == "bios/8086" {
			target = &resolved.Targets[i]
			break
		}
	}
	if target == nil {
		t.Fatal("definition does not export bios/8086")
	}
	if target.Descriptor.OS != "bios" || target.Descriptor.ISA != "8086" ||
		target.Descriptor.OutputKind != "bios-disk" || target.Descriptor.ArenaDefault != 16384 {
		t.Fatalf("unexpected target descriptor: %#v", target.Descriptor)
	}
	generated := GeneratePreparedBackend(resolved, target.Descriptor.Name)
	if !generated.Ok {
		t.Fatalf("generate prepared backend: %#v", generated.Diagnostics)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "bios_8086_generated.go", generated.Source, 0); err != nil {
		t.Fatalf("generated BIOS backend is invalid Go: %v", err)
	}
	for _, want := range []string{
		"PackageBiosBootSector(sectors int) []byte",
		"renvoRTGRejectImageSize(true, memorySize, 0xf000)",
		"return rtgMsdosComBios8086PackageBiosImage(out)",
	} {
		if !strings.Contains(string(generated.Source), want) {
			t.Errorf("prepared BIOS backend omitted %q", want)
		}
	}

	// The same definition exports the next-stage, freestanding target. Checking
	// it from the already-resolved document avoids parsing the large shared 8086
	// architecture twice in the package suite.
	for i := 0; i < len(resolved.Targets); i++ {
		descriptor := resolved.Targets[i].Descriptor
		if descriptor.Name != "freestanding/amd64" {
			continue
		}
		if descriptor.OutputKind != "flat-memory" || descriptor.PointerBits != 64 ||
			!containsString(descriptor.Capabilities, "in_place_entry") {
			t.Fatalf("long-mode descriptor = %#v", descriptor)
		}
		return
	}
	t.Fatal("definition does not export freestanding/amd64")
}

func containsString(values []string, wanted string) bool {
	for i := 0; i < len(values); i++ {
		if values[i] == wanted {
			return true
		}
	}
	return false
}
