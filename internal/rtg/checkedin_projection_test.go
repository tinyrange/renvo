//go:build !renvo

package rtg

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// Renaming public identities must not change the generated physical glue.
// Exercise every recipe through parsing/resolution, not a dispatch-only mock.
func TestCheckedInProductionProjectionIsDefinitionBound(t *testing.T) {
	for _, name := range sortedNativeTargetNames() {
		t.Run(name, func(t *testing.T) {
			filename := nativeDefinitionEntrypoints[name]
			source, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			const renamed = "custom/board"
			modified := bytes.Replace(source, []byte("target "+name+" {"), []byte("target "+renamed+" {"), 1)
			if bytes.Equal(source, modified) {
				t.Fatal("target rename did not change the fixture")
			}
			resolved := Resolve(ParseImports(modified, filename, testFilesystemImportLoader{}))
			if !resolved.Ok {
				t.Fatalf("resolve renamed definition: %#v", resolved.Diagnostics)
			}
			// These are descriptor/display names, not encoder or format contracts.
			resolved.Targets[0].Arch.Name = "private_encoder"
			resolved.Targets[0].Descriptor.OS = "private_environment"
			generated := GenerateCheckedInTargetProjection(resolved, renamed, "main")
			baseline := GenerateCheckedInTargetProjection(resolveNativeTarget(t, name), name, "main")
			if !generated.Ok || !baseline.Ok {
				t.Fatalf("projection failed: renamed=%#v baseline=%#v", generated.Diagnostics, baseline.Diagnostics)
			}
			_, got, gotOK := strings.Cut(string(generated.Source), "package main\n")
			_, want, wantOK := strings.Cut(string(baseline.Source), "package main\n")
			if !gotOK || !wantOK || got != want {
				t.Fatal("public identity changed physical projection output")
			}
		})
	}
}

func TestCheckedInProductionProjectionRejectsInvalidBinding(t *testing.T) {
	const filename = "../../backend/definitions/linux_amd64.rtg"
	source, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, binding, diagnostic string }{
		{"missing", "", "requires a production_projection binding"},
		{"unknown", "production_projection = missing_recipe", "unknown production projection"},
		{"wrong_encoder", "production_projection = elf_eabi32_process", "requires encoder export"},
		{"wrong_format", "production_projection = pe_win64_process", "requires a declarative PE executable image"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			modified := bytes.Replace(source, []byte("production_projection = elf_sysv64_process"), []byte(tc.binding), 1)
			if bytes.Equal(source, modified) {
				t.Fatal("binding mutation did not change the fixture")
			}
			resolved := Resolve(ParseImports(modified, filename, testFilesystemImportLoader{}))
			if !resolved.Ok {
				t.Fatalf("resolve fixture: %#v", resolved.Diagnostics)
			}
			generated := GenerateCheckedInTargetProjection(resolved, "linux/amd64", "main")
			if generated.Ok || len(generated.Source) != 0 || len(generated.Diagnostics) != 1 ||
				!strings.Contains(generated.Diagnostics[0].Message, tc.diagnostic) {
				t.Fatalf("invalid production binding was not rejected: %#v", generated)
			}
		})
	}
}
