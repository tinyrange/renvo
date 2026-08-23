package driver

import (
	"os"
	"path/filepath"
	"strings"
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

func TestPreprocessCCommandReadsKbuildStandardInput(t *testing.T) {
	args := NormalizeCCompilerCommand([]string{
		"renvo", "cc", "-Wno-error", "-Wno-unused-macros", "-E", "-P", "-x", "c", "-",
	})
	if !CPreprocessCommandUsesStandardInput(args) {
		t.Fatalf("normalized standard-input command = %#v", args)
	}
	result := PreprocessCCommandWithInput(args, "/repo", memorySourceFS{}, []byte("#define VALUE 42\nVALUE\n"))
	if !result.Ok || string(result.Source) != "42\n" || result.Output != "-" {
		t.Fatalf("standard-input preprocess result = %#v, source %q", result, result.Source)
	}
}

func TestPreprocessCCommandDumpsMacrosForNullProbe(t *testing.T) {
	args := NormalizeCCompilerCommand([]string{"renvo", "cc", "-dM", "-E", "-x", "c", "/dev/null"})
	result := PreprocessCCommand(args, "/repo", memorySourceFS{})
	if !result.Ok || !strings.Contains(string(result.Source), "#define __GNUC__ 5\n") ||
		strings.Contains(string(result.Source), "__clang__") {
		t.Fatalf("null macro-dump result = %#v, source %q", result, result.Source)
	}
}

func TestPreprocessCCommandReportsI386DataModel(t *testing.T) {
	args := NormalizeCCompilerCommand([]string{"renvo", "cc", "-m32", "-funsigned-char", "-dM", "-E", "-x", "c", "/dev/null"})
	result := PreprocessCCommand(args, "/repo", memorySourceFS{})
	text := string(result.Source)
	if !result.Ok || !strings.Contains(text, "#define __i386__ 1\n") ||
		!strings.Contains(text, "#define __SIZEOF_POINTER__ 4\n") ||
		!strings.Contains(text, "#define __CHAR_UNSIGNED__ 1\n") ||
		strings.Contains(text, "#define __x86_64__ 1\n") || strings.Contains(text, "#define __LP64__ 1\n") {
		t.Fatalf("i386 macro dump = %#v, source %q", result, result.Source)
	}
}

func TestCCommandMacrosDescribeWindowsLLP64(t *testing.T) {
	options := Options{Target: "windows/amd64", Mode: ModeExecutable, CCompiler: true}
	macros := cCommandMacros(options)
	want := map[string]string{
		"_WIN32": "1", "_WIN64": "1", "__SIZEOF_LONG__": "4",
		"__SIZEOF_POINTER__": "8", "__STDC_HOSTED__": "1",
	}
	for name, value := range want {
		found := false
		for i := 0; i < len(macros); i++ {
			if macros[i].Name == name && macros[i].Value == value {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Windows macro %s=%s missing from %#v", name, value, macros)
		}
	}
	undefined := cCommandUndefined(options)
	for _, name := range []string{"__linux__", "__ELF__", "__LP64__"} {
		if findString(undefined, name) < 0 {
			t.Fatalf("incompatible Windows macro %s was not undefined: %#v", name, undefined)
		}
	}
}

func TestCCompilerTargetUsesILP32ForMicrocontrollerISAs(t *testing.T) {
	for _, test := range []struct {
		target string
		macro  string
	}{
		{target: "esp32c6/riscv32", macro: "__riscv"},
		{target: "esp32s3/xtensa_lx7", macro: "__XTENSA__"},
		{target: "esp32p4/riscv32", macro: "__riscv"},
	} {
		_, _, pointerBits := cCompilerTarget(test.target)
		if pointerBits != 32 {
			t.Fatalf("cCompilerTarget(%q) pointer width = %d, want 32", test.target, pointerBits)
		}
		found := false
		for _, macro := range cCommandMacros(Options{Target: test.target, Mode: ModeExecutable, CCompiler: true}) {
			if macro.Name == test.macro {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("C target macro %q missing for %q", test.macro, test.target)
		}
	}
}
