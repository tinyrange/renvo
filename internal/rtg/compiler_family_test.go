package rtg

import (
	"strings"
	"testing"
)

func TestCompilerFamilyContractsIgnoreNames(t *testing.T) {
	structured := strings.ReplaceAll(testMachineDefinition, "family = native_v1", "family = structured32")
	structured = strings.ReplaceAll(structured, "arch tiny64 {", "arch tiny64 {\n\tcompiler_family = structured32")
	structured = strings.ReplaceAll(structured, "format tiny_image {", "format tiny_image {\n\tcompiler_family = structured32\n")
	structured = strings.ReplaceAll(structured, "bits = 64", "bits = 32")
	cases := []struct {
		name, source, code string
	}{
		{"unknown-structured-machine", structured, ""},
		{"native-familiar-alias", strings.ReplaceAll(testMachineDefinition, `alias = "tiny"`, `alias = "wasm32"`), ""},
		{"native-familiar-output-kind", strings.ReplaceAll(testMachineDefinition, "format tiny_image {", "format tiny_image {\nkind = wasm\n"), ""},
		{"native-structured-output", strings.ReplaceAll(testMachineDefinition, "format tiny_image {", "format tiny_image {\ncompiler_family = structured32\n"), "RTG-RESOLVE-024"},
		{"structured-native-output", strings.ReplaceAll(structured, "format tiny_image {\n\tcompiler_family = structured32", "format tiny_image {"), "RTG-RESOLVE-025"},
		{"structured-native-object", strings.ReplaceAll(structured, "executable = tiny_image", "executable = tiny_image\nobject = native_object") + "\nformat native_object { address_bits = 32 }\n", "RTG-RESOLVE-025"},
		{"structured-wide-word", strings.ReplaceAll(structured, "word_bits = 32", "word_bits = 64"), "RTG-RESOLVE-025"},
		{"structured-wide-pointer", strings.ReplaceAll(structured, "pointer_bits = 32", "pointer_bits = 64"), "RTG-RESOLVE-025"},
		{"familiar-name-no-capability", strings.ReplaceAll(strings.ReplaceAll(testMachineDefinition, "family = native_v1", "family = structured32"), `alias = "tiny"`, `alias = "wasm32"`), "RTG-RESOLVE-025"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolved := Resolve(Parse([]byte(tc.source), "family.rtg"))
			if tc.code == "" {
				if !resolved.Ok {
					t.Fatalf("Resolve: %#v", resolved.Diagnostics)
				}
			} else if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, tc.code) {
				t.Fatalf("Resolve = %v, diagnostics %#v; want %s", resolved.Ok, resolved.Diagnostics, tc.code)
			}
		})
	}
}

func TestCompilerFamilyRejectsMalformedDeclarations(t *testing.T) {
	for _, declaration := range []string{"arch tiny64 {", "format tiny_image {"} {
		for _, field := range []string{
			"compiler_family = mystery",
			"compiler_family = native_v1 extra",
			"compiler_family { value = native_v1 }",
			"compiler_family = native_v1\ncompiler_family = native_v1",
		} {
			source := strings.ReplaceAll(testMachineDefinition, declaration, declaration+"\n"+field+"\n")
			resolved := Resolve(Parse([]byte(source), "invalid-family.rtg"))
			if resolved.Ok || !(hasDiagnosticCode(resolved.Diagnostics, "RTG-VALIDATE-135") || hasDiagnosticCode(resolved.Diagnostics, "RTG-VALIDATE-060")) {
				t.Fatalf("%s %s: %#v", declaration, field, resolved.Diagnostics)
			}
			generated := GenerateFixedBackend(resolved, "test/tiny64")
			if generated.Ok || len(generated.Source) != 0 {
				t.Fatalf("invalid family emitted source")
			}
		}
	}
}
