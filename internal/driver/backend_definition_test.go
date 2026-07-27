package driver

import (
	"testing"

	backendunit "renvo.dev/backend/unit"
	"renvo.dev/internal/load"
	internalunit "renvo.dev/internal/unit"
)

const testBackendDefinition = `definition 1
unit acme
implements direct_emitter_v1
arch a64 {
	alias = "aarch64"
	endian = little
	word_bits = 64
	pointer_bits = 64
}
abi acme_abi { arch = a64 }
runtime acme_runtime { operations = [print] }
format acme_image { address_bits = 64 }
target acme/aarch64 {
	arch = a64
	abi = acme_abi
	runtime = acme_runtime
	executable = acme_image
	aliases = ["acme/arm64"]
	build_tags = [linux, aarch64, arm64]
	capabilities = [hosted, executable]
}
`

func TestBackendDefinitionResolvesBeforeSourceSelection(t *testing.T) {
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/acme.rtg", Src: []byte(testBackendDefinition)},
		{Path: "/repo/case/cmd/app/main_linux_arm64.go", Src: []byte("package main\nfunc appMain() int { return 42 }\n")},
		{Path: "/repo/case/cmd/app/main_windows_amd64.go", Src: []byte("this file must not parse")},
	}
	result := BuildFromFSWithBackend([]string{
		"-backend", "acme.rtg",
		"-t", "acme/arm64",
		"-emit-unit",
		"-o", "app.unit",
		"./cmd/app",
	}, "/repo/case", "/std", memorySourceFS{files: files})
	if !result.Ok {
		t.Fatalf("BuildFromFS failed: %#v", result.Diagnostic)
	}
	if result.Options.Target != "acme/aarch64" {
		t.Fatalf("resolved options = %#v", result.Options)
	}
	binding, ok := internalunit.ReadTargetBinding(result.Unit)
	if !ok || binding.Target != "acme/aarch64" ||
		binding.DescriptorVersion != 1 {
		t.Fatalf("binding = %#v, ok %v", binding, ok)
	}
	if _, err := backendunit.Unmarshal(result.Unit); err != nil {
		t.Fatalf("ordinary backend reader rejected optional binding tags: %v", err)
	}
}

func TestBackendDefinitionRejectsUnknownSelectedTarget(t *testing.T) {
	files := append(driverTestFiles(), load.SourceFile{Path: "/repo/case/acme.rtg", Src: []byte(testBackendDefinition)})
	result := BuildFromFSWithBackend([]string{
		"-backend", "acme.rtg",
		"-t", "acme/missing",
		"-o", "app",
		"./cmd/app",
	}, "/repo/case", "/std", memorySourceFS{files: files})
	if result.Ok || result.Diagnostic.Code != "RENVO-OPTION-027" {
		t.Fatalf("BuildFromFS = %#v", result)
	}
}

func TestBackendDefinitionAcceptsSystemTargetDefault(t *testing.T) {
	files := append(driverTestFiles(),
		load.SourceFile{Path: "/repo/case/acme.rtg", Src: []byte(testBackendDefinition)},
		load.SourceFile{Path: "/repo/case/system.rtg", Src: []byte(`system "acme-small" { target = "acme/arm64" binary = 1MiB arena = 1MiB }`)},
	)
	result := BuildFromFSWithBackend([]string{
		"-backend", "acme.rtg",
		"-system", "system.rtg",
		"-emit-unit",
		"-o", "app.unit",
		"./cmd/app",
	}, "/repo/case", "/std", memorySourceFS{files: files})
	if !result.Ok {
		t.Fatalf("BuildFromFS failed: %#v", result.Diagnostic)
	}
	if result.Options.Target != "acme/aarch64" || result.Options.SystemName != "acme-small" {
		t.Fatalf("resolved options = %#v", result.Options)
	}
}
