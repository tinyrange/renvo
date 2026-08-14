package rtg

import (
	"strings"
	"testing"
)

func TestESP32C6ExternalDefinition(t *testing.T) {
	const filename = "../../backends/esp32c6.rtg"
	resolved := Resolve(parseDefinitionFile(t, filename))
	if !resolved.Ok {
		t.Fatalf("resolve %s: %#v", filename, resolved.Diagnostics)
	}
	if len(resolved.Targets) != 1 {
		t.Fatalf("target count = %d, want one", len(resolved.Targets))
	}
	target := resolved.Targets[0]
	if target.Descriptor.Name != "esp32c6/riscv32" ||
		target.Descriptor.OS != "esp32c6" ||
		target.Descriptor.ISA != "riscv32" ||
		target.Descriptor.WordBits != 32 ||
		target.Descriptor.PointerBits != 32 ||
		target.Descriptor.OutputKind != "elf" {
		t.Fatalf("unexpected target descriptor: %#v", target.Descriptor)
	}
	if len(target.Descriptor.Aliases) != 1 ||
		target.Descriptor.Aliases[0] != "m5nanoc6/riscv32" {
		t.Fatalf("aliases = %#v, want m5nanoc6/riscv32",
			target.Descriptor.Aliases)
	}
	generated := GeneratePreparedBackend(resolved, target.Descriptor.Name)
	if !generated.Ok {
		t.Fatalf("generate prepared backend: %#v", generated.Diagnostics)
	}
	source := string(generated.Source)
	for _, contract := range []string{
		"func rtgEsp32c6Esp32c6PackageImage(",
		"func rtgEsp32c6Esp32c6PackageClearBSS(",
		"func rtgEsp32c6Esp32c6PackageUsbSerialInitialize(",
		"RTGBSSEndAddress(4)",
		"RTGResolveAbsoluteRelocation(",
		"0x60008064, 0x50d83aa1",
		"0x6000f04c, 4",
		"UsbSerialEndpoint = 0x6000f000",
		"UsbSerialInterruptRaw = ",
		"AppDescriptorAddress = 0x42010020",
		"WordBits:32,PointerBits:32",
		"func rtgEsp32c6Riscv32PackageMoveImmediate(",
		"if setcc == 150 { return rtgEsp32c6ULE }",
		"if setcc == 151 { return rtgEsp32c6UGT }",
	} {
		if !strings.Contains(source, contract) {
			t.Errorf("prepared backend omitted %q", contract)
		}
	}
}
