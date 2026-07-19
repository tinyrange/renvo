package target

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestC89ExplicitProfileCompilesAndIsDeterministic(t *testing.T) {
	bits := 32
	if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
		bits = 64
	}
	profile := C89ExplicitProfile("host-check", false, bits, bits, CEndianLittle, "test-abi", "write", "result", "write")
	first, err := profile.RenderC89Preamble()
	if err != nil {
		t.Fatal(err)
	}
	second, err := profile.RenderC89Preamble()
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("C89 profile output is not deterministic")
	}
	if strings.Count(string(first), "RENVO_C_RUNTIME_WRITE") != 1 || !strings.Contains(string(first), "RENVO_C89_ASSERT(pointer_width") {
		t.Fatalf("profile contract is incomplete:\n%s", first)
	}
	compileC89(t, first, true)
}

func TestC89AutomaticProfileCompiles(t *testing.T) {
	profile := C89AutomaticProfile("compiler-selected", true, CEndianLittle, "host-c", "open", "close")
	source, err := profile.RenderC89Preamble()
	if err != nil {
		t.Fatal(err)
	}
	compileC89(t, source, true)
	for _, name := range []string{"gcc", "clang"} {
		compiler, lookupErr := exec.LookPath(name)
		if lookupErr != nil {
			continue
		}
		t.Run(name, func(t *testing.T) {
			compileC89With(t, compiler, source, true)
		})
	}
}

func TestC89MismatchedExplicitProfileNamesFailedAssumption(t *testing.T) {
	bits := 16
	if ^uint(0)>>63 == 1 {
		bits = 32
	}
	profile := C89ExplicitProfile("mismatch", false, 16, bits, CEndianLittle, "wrong-pointer")
	source, err := profile.RenderC89Preamble()
	if err != nil {
		t.Fatal(err)
	}
	output := compileC89(t, source, false)
	if !strings.Contains(output, "renvo_assumption_pointer_width") {
		t.Fatalf("mismatch did not name pointer-width assumption:\n%s", output)
	}
}

func TestC89ProfileRejectsNonC89Surface(t *testing.T) {
	profile := C89ExplicitProfile("surface", false, 32, 32, CEndianBig, "ilp32", "result")
	source, err := profile.RenderC89Preamble()
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"//", "_Static_assert", "stdint.h", "stdbool.h", " inline ", "..."} {
		if strings.Contains(string(source), forbidden) {
			t.Fatalf("generated profile contains non-C89 surface %q", forbidden)
		}
	}
}

func TestC89ProfileValidation(t *testing.T) {
	profile := C89ExplicitProfile("bad", false, 8, 32, CEndianLittle, "non-iso-int")
	if _, err := profile.RenderC89Preamble(); err == nil || !strings.Contains(err.Error(), "16, 32, or 64") {
		t.Fatalf("8-bit ISO C int profile error = %v", err)
	}
	profile = C89ExplicitProfile("bad", false, 32, 32, CEndianLittle, "abi", "not-valid")
	if _, err := profile.RenderC89Preamble(); err == nil || !strings.Contains(err.Error(), "runtime operation") {
		t.Fatalf("invalid operation error = %v", err)
	}
	profile = C89ExplicitProfile("non-ASCII-\u2603", false, 32, 32, CEndianLittle, "abi")
	if _, err := profile.RenderC89Preamble(); err == nil || !strings.Contains(err.Error(), "printable ASCII") {
		t.Fatalf("non-ASCII profile error = %v", err)
	}
	profile = C89ExplicitProfile("valid", false, 32, 32, CEndianLittle, "line\nbreak")
	if _, err := profile.RenderC89Preamble(); err == nil || !strings.Contains(err.Error(), "printable ASCII") {
		t.Fatalf("non-printable ABI error = %v", err)
	}
}

func compileC89(t *testing.T, preamble []byte, wantSuccess bool) string {
	t.Helper()
	compiler, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("C compiler not installed")
	}
	return compileC89With(t, compiler, preamble, wantSuccess)
}

func compileC89With(t *testing.T, compiler string, preamble []byte, wantSuccess bool) string {
	t.Helper()
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "profile.c")
	objectPath := filepath.Join(dir, "profile.o")
	source := append(append([]byte(nil), preamble...), []byte("\nint renvo_profile_fixture(void) { return RENVO_C_LANGUAGE_INT_BITS + RENVO_C_POINTER_BITS; }\n")...)
	if err := os.WriteFile(sourcePath, source, 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(compiler, "-std=c89", "-pedantic-errors", "-Wall", "-Werror", "-ffreestanding", "-c", sourcePath, "-o", objectPath)
	output, compileErr := command.CombinedOutput()
	if wantSuccess && compileErr != nil {
		t.Fatalf("strict C89 compilation failed: %v\n%s", compileErr, output)
	}
	if !wantSuccess && compileErr == nil {
		t.Fatal("mismatched C profile unexpectedly compiled")
	}
	return string(output)
}
