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
}
