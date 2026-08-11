package rtg

import (
	"slices"
	"strings"
	"testing"
)

func TestESP32S3ExternalDefinition(t *testing.T) {
	const filename = "../../examples/m5sticks3/esp32s3.rtg"
	resolved := Resolve(parseDefinitionFile(t, filename))
	if !resolved.Ok {
		t.Fatalf("resolve %s: %#v", filename, resolved.Diagnostics)
	}
	if len(resolved.Targets) != 1 {
		t.Fatalf("target count = %d, want one", len(resolved.Targets))
	}
	target := resolved.Targets[0]
	if target.Descriptor.Name != "esp32s3/xtensa_lx7" ||
		target.Descriptor.OS != "esp32s3" ||
		target.Descriptor.ISA != "riscv32" ||
		target.Descriptor.WordBits != 32 ||
		target.Descriptor.PointerBits != 32 ||
		target.Descriptor.OutputKind != "elf" ||
		target.Descriptor.ArenaDefault != 131072 ||
		!slices.Contains(target.Descriptor.BuildTags, "tiny") {
		t.Fatalf("unexpected target descriptor: %#v", target.Descriptor)
	}
	generated := GeneratePreparedBackend(resolved, target.Descriptor.Name)
	if !generated.Ok {
		t.Fatalf("generate prepared backend: %#v", generated.Diagnostics)
	}
	source := string(generated.Source)
	for _, contract := range []string{
		"func rtgEsp32s3Esp32s3PackageUsbSerialInitialize(",
		"func rtgEsp32s3XtensaLx7PackageEmitLongJump(",
		"UsbSerialEndpoint = 0x60038000",
		"UsbSerialInterruptRaw = ",
		"UsbSerialProbeIterations = 65536",
		"if setcc == 150 { return rtgEsp32s3ULE }",
		"if setcc == 151 { return rtgEsp32s3UGT }",
	} {
		if !strings.Contains(source, contract) {
			t.Errorf("prepared backend omitted %q", contract)
		}
	}
}
