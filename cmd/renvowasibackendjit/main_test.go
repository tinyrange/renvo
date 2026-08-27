package main

import (
	"testing"

	"renvo.dev/internal/rtg"
)

func TestTargetManifestDescribesVMCompilerAndOutput(t *testing.T) {
	var definition [32]byte
	definition[0] = 0xab
	got := targetManifest(rtg.TargetDescriptor{
		Name: "example/riscv32", OutputKind: "elf", BuildTags: []string{"example"},
		Definition: definition, Version: rtg.DescriptorVersion,
	})
	if got.BackendFormat != "vm32" || got.Output != "app.elf" ||
		got.Definition[:2] != "ab" || got.DescriptorVersion != rtg.DescriptorVersion {
		t.Fatalf("manifest = %#v", got)
	}
}

func TestOutputNameFollowsTargetImage(t *testing.T) {
	for _, test := range []struct{ target, kind, want string }{
		{"example/wasm32", "wasm", "app.wasm"},
		{"example/vm32", "rnvm", "app.rnvb"},
		{"msdos/8086", "dos-com", "app.com"},
		{"msdos/8086-mz", "dos-mz", "app.exe"},
		{"uefi/amd64", "uefi-pe", "BOOTX64.EFI"},
		{"windows/custom", "pe", "app.exe"},
		{"example/riscv32", "elf", "app.elf"},
		{"unixv7/pdp11", "v7-a.out", "app"},
		{"example/raw", "raw", "app"},
	} {
		if got := outputName(test.target, test.kind); got != test.want {
			t.Errorf("outputName(%q, %q) = %q, want %q", test.target, test.kind, got, test.want)
		}
	}
}
