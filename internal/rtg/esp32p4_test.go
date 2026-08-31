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
	if len(resolved.Targets) != 2 {
		t.Fatalf("target count = %d, want two", len(resolved.Targets))
	}
	var target ResolvedTarget
	var compilerHost ResolvedTarget
	for _, candidate := range resolved.Targets {
		switch candidate.Descriptor.Name {
		case "esp32p4/riscv32":
			target = candidate
		case "esp32p4/compiler-host":
			compilerHost = candidate
		}
	}
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
		"AUIPC/JALR covers the full 32-bit PC-relative range",
		"second &= 0x000fffff",
	} {
		if !strings.Contains(source, contract) {
			t.Errorf("prepared backend omitted %q", contract)
		}
	}
	if compilerHost.Descriptor.Name == "" ||
		compilerHost.Descriptor.OS != "esp32p4" ||
		compilerHost.Descriptor.ISA != "riscv32" ||
		compilerHost.Descriptor.OutputKind != "elf" {
		t.Fatalf("unexpected compiler-host descriptor: %#v", compilerHost.Descriptor)
	}
	generated = GeneratePreparedBackend(resolved, compilerHost.Descriptor.Name)
	if !generated.Ok {
		t.Fatalf("generate compiler-host backend: %#v", generated.Diagnostics)
	}
	for _, contract := range []string{
		"CompilerDataAddress = 0x48000000",
		"CompilerDataLimit = 0x4a000000",
		"CompilerHostCommand = 0xffff1000",
		"ArenaDefault:33292288",
	} {
		if !strings.Contains(string(generated.Source), contract) {
			t.Errorf("prepared compiler-host backend omitted %q", contract)
		}
	}
}
