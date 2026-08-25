package rtg

import (
	"strings"
	"testing"
)

func TestESP32P4ExternalDefinition(t *testing.T) {
	const filename = "../../backends/esp32p4.rtg"
	resolved := Resolve(parseDefinitionFile(t, filename))
	if !resolved.Ok {
		t.Fatalf("resolve %s: %#v", filename, resolved.Diagnostics)
	}
	if len(resolved.Targets) != 1 {
		t.Fatalf("target count = %d, want one", len(resolved.Targets))
	}
	target := resolved.Targets[0]
	if target.Descriptor.Name != "esp32p4/riscv32" ||
		target.Descriptor.OS != "esp32p4" ||
		target.Descriptor.ISA != "riscv32" ||
		target.Descriptor.WordBits != 32 ||
		target.Descriptor.PointerBits != 32 ||
		target.Descriptor.OutputKind != "elf" {
		t.Fatalf("unexpected target descriptor: %#v", target.Descriptor)
	}
	generated := GeneratePreparedBackend(resolved, target.Descriptor.Name)
	if !generated.Ok {
		t.Fatalf("generate prepared backend: %#v", generated.Diagnostics)
	}
	source := string(generated.Source)
	for _, contract := range []string{
		"func rtgEsp32p4Esp32p4PackageImage(",
		"RTGBSSEndAddress(4)",
		"0x500c2064, 0x50d83aa1",
		"0x50116018, 0x50d83aa1",
		"0x500d204c, 4",
		"UsbSerialEndpoint = 0x500d2000",
		"AppDescriptorAddress = 0x40010020",
		"BssLimit = 0x4ff2a000",
	} {
		if !strings.Contains(source, contract) {
			t.Errorf("prepared backend omitted %q", contract)
		}
	}
}
