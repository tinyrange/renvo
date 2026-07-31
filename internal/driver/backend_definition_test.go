package driver

import (
	"strings"
	"testing"

	backendunit "renvo.dev/backend/unit"
	"renvo.dev/internal/load"
	"renvo.dev/internal/rtg"
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
	reject = [
		move, address,
		load.native, load.i32, load.u32, load.i16, load.u16, load.i8, load.u8,
		store.native, store.u32, store.u16, store.u8,
		add, subtract, multiply, bit_and, bit_or, bit_xor, compare, test,
		increment, decrement,
		shift_left_immediate, shift_right_unsigned_immediate, shift_right_signed_immediate,
		call, call_indirect, jump, jump_condition, set_condition, return, leave,
		host_syscall, move_immediate, variable_shift, signed_divide, copy_bytes
	]
}
abi acme_abi { arch = a64 }
runtime acme_runtime { operations = [print] }
format acme_image { address_bits = 64 }
target acme/aarch64 {
	family = native_v1
	os = linux
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
		binding.DescriptorVersion != rtg.DescriptorVersion {
		t.Fatalf("binding = %#v, ok %v", binding, ok)
	}
	if _, err := backendunit.Unmarshal(result.Unit); err != nil {
		t.Fatalf("ordinary backend reader rejected optional binding tags: %v", err)
	}
}

func TestBackendDefinitionResolvesRelativeImportsFromSourceFS(t *testing.T) {
	declarationAt := strings.Index(testBackendDefinition, "arch a64")
	if declarationAt < 0 {
		t.Fatal("test definition has no architecture declaration")
	}
	root := testBackendDefinition[:declarationAt] + `@import "machine/native.rtg"` + "\n"
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/definitions/acme.rtg", Src: []byte(root)},
		{Path: "/repo/case/definitions/machine/native.rtg", Src: []byte(testBackendDefinition[declarationAt:])},
		{Path: "/repo/case/cmd/app/main_linux_arm64.go", Src: []byte("package main\nfunc appMain() int { return 42 }\n")},
	}
	result := BuildFromFSWithBackend([]string{
		"-backend", "definitions/acme.rtg",
		"-t", "acme/aarch64",
		"-emit-unit",
		"-o", "app.unit",
		"./cmd/app",
	}, "/repo/case", "/std", memorySourceFS{files: files})
	if !result.Ok {
		t.Fatalf("imported backend BuildFromFS failed: %#v", result.Diagnostic)
	}
	if result.Options.Target != "acme/aarch64" {
		t.Fatalf("imported backend target = %q", result.Options.Target)
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

func TestBackendDefinitionAcceptsUnknownArchitectureDescriptor(t *testing.T) {
	definition := []byte(testBackendDefinition)
	definition = []byte(replaceDriverText(string(definition),
		`alias = "aarch64"`, `alias = "example64"`))
	files := append(driverTestFiles(),
		load.SourceFile{Path: "/repo/case/acme.rtg", Src: definition})
	result := BuildFromFSWithBackend([]string{
		"-backend", "acme.rtg",
		"-t", "acme/aarch64",
		"-emit-unit",
		"-o", "app.unit",
		"./cmd/app",
	}, "/repo/case", "/std", memorySourceFS{files: files})
	if !result.Ok {
		t.Fatalf("unknown architecture BuildFromFS = %#v", result)
	}
	if result.Options.Target != "acme/aarch64" {
		t.Fatalf("unknown architecture target = %q", result.Options.Target)
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

func replaceDriverText(source string, old string, replacement string) string {
	for i := 0; i+len(old) <= len(source); i++ {
		if source[i:i+len(old)] == old {
			return source[:i] + replacement + source[i+len(old):]
		}
	}
	return source
}
