package driver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreprocessCCommandUsesNormalizedCompilerOptions(t *testing.T) {
	dir := t.TempDir()
	include := filepath.Join(dir, "include")
	if err := os.Mkdir(include, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(include, "value.h"), []byte("#define HEADER_VALUE 3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "forced.h"), []byte("#define FORCED_VALUE 4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte("#include <value.h>\nVALUE HEADER_VALUE FORCED_VALUE\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	args := NormalizeCCompilerCommand([]string{
		"renvo", "cc", "-E", "-P", "-nostdinc", "-Iinclude", "-include", "forced.h",
		"-DVALUE=2", "-U__GNUC__", "-Wp,-MMD,main.d", "-MT", "main.i", "main.c",
	})
	if !CPreprocessCommandRequested(args) {
		t.Fatalf("normalized command was not recognized: %#v", args)
	}
	result := PreprocessCCommand(args, dir, OSFS{})
	if !result.Ok || string(result.Source) != "2 3 4\n" || result.Output != "-" ||
		result.DependencyFile != "main.d" || len(result.DependencyData) == 0 {
		t.Fatalf("preprocess result = %#v, source %q", result, result.Source)
	}
}

func TestPreprocessCCommandRejectsUnknownOption(t *testing.T) {
	args := NormalizeCCompilerCommand([]string{"renvo", "cc", "-E", "-P", "-fpretend-cpp", "main.c"})
	result := PreprocessCCommand(args, ".", OSFS{})
	if result.Ok || result.Option != "-fpretend-cpp" {
		t.Fatalf("unknown preprocessor option result = %#v", result)
	}
}

func TestPreprocessCCommandPreservesExplicitEmptyDefine(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte("before EMPTY after\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	args := NormalizeCCompilerCommand([]string{"renvo", "cc", "-E", "-P", "-DEMPTY=", "main.c"})
	result := PreprocessCCommand(args, dir, OSFS{})
	if !result.Ok || string(result.Source) != "before after\n" {
		t.Fatalf("empty define result = %#v, source %q", result, result.Source)
	}
}
