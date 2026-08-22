//go:build !renvo

package backendjit

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

func c89GeneratedByteArray(t *testing.T, source []byte, name string) []byte {
	t.Helper()
	prefix := []byte("static unsigned char " + name + "[] = {\n")
	start := bytes.Index(source, prefix)
	if start < 0 {
		t.Fatalf("generated C89 output omits %s byte array", name)
	}
	start += len(prefix)
	end := bytes.Index(source[start:], []byte("};\n"))
	if end < 0 {
		t.Fatalf("generated C89 output does not terminate %s byte array", name)
	}
	fields := strings.FieldsFunc(string(source[start:start+end]), func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t' || r == '\r'
	})
	result := make([]byte, 0, len(fields))
	for _, field := range fields {
		value, err := strconv.Atoi(field)
		if err != nil || value < 0 || value > 255 {
			t.Fatalf("invalid byte %q in generated %s array", field, name)
		}
		result = append(result, byte(value))
	}
	return result
}

func compileC89SourceArenaTags(t *testing.T, target string, sourcePath string, arenaSize int, tags string) ([]byte, string) {
	t.Helper()
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "backends", "c89.rtg")
	args := []string{
		"-backend", definition,
		"-t", target,
	}
	if arenaSize > 0 {
		args = append(args, "-arena-size", strconv.Itoa(arenaSize))
	}
	if tags != "" {
		args = append(args, "-tags", tags)
	}
	args = append(args, "-s", "-o", "program.c", sourcePath)
	result := driver.CompileFromFS(args, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			backendJITTestCacheDir, backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("C89 CompilerJIT compile failed: %#v", result.Diagnostic)
	}
	return result.Binary, root
}

func compileC89SourceArena(t *testing.T, target string, sourcePath string, arenaSize int) ([]byte, string) {
	t.Helper()
	return compileC89SourceArenaTags(t, target, sourcePath, arenaSize, "")
}

func compileC89Source(t *testing.T, target string, sourcePath string) ([]byte, string) {
	t.Helper()
	return compileC89SourceArena(t, target, sourcePath, 0)
}

func compileC89Unit(t *testing.T, root string, sourcePath string, tags string) []byte {
	t.Helper()
	args := []string{"-t", "linux/amd64", "-emit-unit", "-s", "-o", "program.rtgu"}
	if tags != "" {
		args = append(args, "-tags", tags)
	}
	args = append(args, sourcePath)
	result := driver.CompileFromFS(args, root, filepath.Join(root, "std"), driver.OSFS{}, backendcompiled.Backend{})
	if !result.Ok {
		t.Fatalf("unit compile failed for %s: %#v", sourcePath, result.Diagnostic)
	}
	return result.Binary
}

func compileC89Fixture(t *testing.T, target string, fixture string) ([]byte, string) {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return compileC89Source(t, target,
		filepath.Join(root, "internal", "backendjit", "testdata", fixture))
}

func compileC89Program(t *testing.T, target string) ([]byte, string) {
	t.Helper()
	return compileC89Fixture(t, target, "c89_semantics.go")
}

func TestCompilerJITC89TranslationUnit(t *testing.T) {
	first, _ := compileC89Program(t, "c89/hosted32")
	second, _ := compileC89Program(t, "c89/hosted32")
	if !bytes.Equal(first, second) {
		t.Fatal("C89 output is not deterministic")
	}
	for _, required := range [][]byte{
		[]byte("#include <limits.h>\n"),
		[]byte("CHAR_BIT == 8"),
		[]byte("sizeof(void *) * CHAR_BIT == 32"),
		[]byte("renvo_assumption_pointer_width"),
		[]byte("UINT_MAX == 4294967295UL"),
		[]byte("typedef unsigned int rgu32"),
		[]byte("int main(int argc, char **argv)"),
	} {
		if !bytes.Contains(first, required) {
			t.Errorf("generated C89 output omits %q", required)
		}
	}
	for _, forbidden := range [][]byte{
		[]byte("//"),
		[]byte("_Static_assert"),
		[]byte("stdint.h"),
		[]byte("stdbool.h"),
		[]byte("inline "),
		[]byte("long long"),
	} {
		if bytes.Contains(first, forbidden) {
			t.Errorf("generated C89 output contains forbidden construct %q", forbidden)
		}
	}
}

func TestCompilerJITC89IndirectCallsAreEmitted(t *testing.T) {
	source, root := compileC89Fixture(t, "c89/hosted32", "c89_semantics.go")
	functionValues, _ := compileC89Source(t, "c89/hosted32",
		filepath.Join(root, "backend", "tests", "c89_indirect_calls.go"))
	if !bytes.Contains(source, []byte("else if (op==37UL)")) {
		t.Fatal("generated C89 runtime omits indirect-call execution")
	}
	code := c89GeneratedByteArray(t, functionValues, "rgcod")
	if !bytes.Contains(code, []byte{37, 7}) {
		t.Fatal("compiled function-value program does not exercise an indirect call through the scratch register")
	}
}

func TestCompilerJITC89ScratchDecrementIsAliasSafe(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source, _ := compileC89Source(t, "c89/hosted32",
		filepath.Join(root, "backend", "tests", "c89_scratch_decrement.go"))
	code := c89GeneratedByteArray(t, source, "rgcod")
	if !bytes.Contains(code, []byte{45, 7}) {
		t.Fatal("compiled pointer decrement does not use the alias-safe scratch-register opcode")
	}
}

func TestCompilerJITC89Profiles(t *testing.T) {
	explicit, _ := compileC89Program(t, "c89/hosted32")
	automatic, _ := compileC89Program(t, "c89/hosted32-auto")
	freestanding, _ := compileC89Program(t, "c89/freestanding32")
	if !bytes.Contains(explicit, []byte("typedef char renvo_assumption_unsigned_int_width[")) {
		t.Fatal("explicit profile omits its unsigned-int assumption")
	}
	if bytes.Contains(explicit, []byte("#elif ULONG_MAX")) {
		t.Fatal("explicit profile unexpectedly selects a carrier automatically")
	}
	if !bytes.Contains(automatic, []byte("#elif ULONG_MAX == 4294967295UL")) {
		t.Fatal("automatic profile does not select from exact native widths")
	}
	if !bytes.Contains(freestanding, []byte("int rgmain(void)")) {
		t.Fatal("freestanding profile omits its external entry point")
	}
	if bytes.Contains(freestanding, []byte("int main(")) {
		t.Fatal("freestanding profile depends on hosted startup")
	}
}

func TestCompilerJITC89HostedWrite(t *testing.T) {
	source, _ := compileC89Fixture(t, "c89/hosted32", "c89_write.go")
	for _, required := range [][]byte{
		[]byte("#include <stdio.h>"),
		[]byte("fwrite("),
	} {
		if !bytes.Contains(source, required) {
			t.Errorf("hosted write output omits %q", required)
		}
	}
}

func TestCompilerJITC89HostedProcessRuntime(t *testing.T) {
	source, _ := compileC89Fixture(t, "c89/hosted32", "args_env.go")
	for _, required := range [][]byte{
		[]byte("int main(int argc, char **argv)"),
		[]byte("fopen("),
		[]byte("fread("),
		[]byte("fgetpos("),
		[]byte("opendir("),
		[]byte("readdir("),
		[]byte("chmod("),
		[]byte("getenv("),
	} {
		if !bytes.Contains(source, required) {
			t.Errorf("hosted process runtime omits %q", required)
		}
	}
}

func TestCompilerJITC89DockerMatrix(t *testing.T) {
	if os.Getenv("RENVO_C89_DOCKER") != "1" {
		t.Skip("set RENVO_C89_DOCKER=1 to compile and execute with every C89 container")
	}
	profiles := []struct {
		target string
		name   string
	}{
		{target: "c89/hosted32", name: "hosted-explicit.c"},
		{target: "c89/hosted32-auto", name: "hosted-auto.c"},
		{target: "c89/freestanding32", name: "freestanding-explicit.c"},
	}
	directory := t.TempDir()
	var root string
	paths := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		source, repositoryRoot := compileC89Program(t, profile.target)
		root = repositoryRoot
		path := filepath.Join(directory, profile.name)
		if err := os.WriteFile(path, source, 0o600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	writeSource, _ := compileC89Fixture(t, "c89/hosted32", "c89_write.go")
	writePath := filepath.Join(directory, "hosted-write.c")
	if err := os.WriteFile(writePath, writeSource, 0o600); err != nil {
		t.Fatal(err)
	}
	paths = append(paths, writePath)
	faultSource, _ := compileC89Fixture(t, "c89/hosted32", "c89_fault.go")
	faultPath := filepath.Join(directory, "hosted-fault.c")
	if err := os.WriteFile(faultPath, faultSource, 0o600); err != nil {
		t.Fatal(err)
	}
	paths = append(paths, faultPath)
	corpusDirectory := filepath.Join(directory, "corpus")
	if err := os.Mkdir(corpusDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	corpus := []string{
		"arithmetic_mod_remainder.go",
		"append_expansion_byte_overlap.go",
		"c89_function_zero_composite.go",
		"c89_host_syscall.go",
		"c89_indirect_calls.go",
		"c89_interface_struct_result.go",
		"c89_large_struct_arguments.go",
		"c89_scratch_decrement.go",
		"int64_uint64_32bit_operations.go",
		"unsafe_pointer_array_index.go",
		"variadic_functions_multiple_ints.go",
	}
	for _, name := range corpus {
		source, _ := compileC89Source(t, "c89/hosted32",
			filepath.Join(root, "backend", "tests", name))
		path := filepath.Join(corpusDirectory, name[:len(name)-3]+".c")
		if err := os.WriteFile(path, source, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	paths = append(paths, corpusDirectory)
	command := exec.Command(filepath.Join(root, "tools", "c89", "matrix"), paths...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("C89 Docker matrix failed: %v\n%s", err, output)
	}
	t.Logf("%s", output)
}

func TestCompilerJITC89DockerBootstrap(t *testing.T) {
	if os.Getenv("RENVO_C89_BOOTSTRAP_DOCKER") != "1" {
		t.Skip("set RENVO_C89_BOOTSTRAP_DOCKER=1 to bootstrap through every C89 container")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	write := func(name string, data []byte) string {
		t.Helper()
		path := filepath.Join(directory, name)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	backendSource, _ := compileC89SourceArena(t, "c89/hosted32",
		filepath.Join(root, "backend"), 128*1024*1024)
	frontendSource, _ := compileC89SourceArenaTags(t, "c89/hosted32",
		filepath.Join(root, "cmd", "renvo"), 256*1024*1024, "renvo_bundle")
	backendUnit := compileC89Unit(t, root, filepath.Join(root, "backend"), "")
	frontendUnit := compileC89Unit(t, root, filepath.Join(root, "cmd", "renvo"), "renvo_bundle")
	backendSmokeUnit := compileC89Unit(t, root,
		filepath.Join(root, "backend", "tests", "c89_scratch_decrement.go"), "")
	command := exec.Command(filepath.Join(root, "tools", "c89", "bootstrap"),
		write("backend.c", backendSource),
		write("frontend.c", frontendSource),
		write("backend.rtgu", backendUnit),
		write("frontend.rtgu", frontendUnit),
		write("backend-smoke.rtgu", backendSmokeUnit))
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("C89 Docker bootstrap failed: %v\n%s", err, output)
	}
	t.Logf("%s", output)
}
