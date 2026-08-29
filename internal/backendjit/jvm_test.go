//go:build !renvo

package backendjit

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

func compileJVMExample(t *testing.T) ([]byte, string) {
	t.Helper()
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "backends", "jvm.rbe")
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "jvm/vm32",
		"-s",
		"-o", "RenvoProgram.class",
		filepath.Join(root, "examples", "jvm"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			backendJITTestCacheDir, backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("JVM CompilerJIT compile failed: %#v", result.Diagnostic)
	}
	return result.Binary, root
}

func TestCompilerJITJVMClassfileAndInterop(t *testing.T) {
	class, _ := compileJVMExample(t)
	if len(class) < 10 || !bytes.Equal(class[:4], []byte{0xca, 0xfe, 0xba, 0xbe}) {
		t.Fatalf("JVM output has no classfile magic: %x", class[:min(len(class), 16)])
	}
	if class[4] != 0 || class[5] != 0 || class[6] != 0 || class[7] != 49 {
		t.Fatalf("classfile version = %x, want 49.0", class[4:8])
	}
	for _, required := range [][]byte{
		[]byte("RenvoProgram"),
		[]byte("java/lang/System"),
		[]byte("java/lang/reflect/Method"),
		[]byte("reflectString"),
	} {
		if !bytes.Contains(class, required) {
			t.Errorf("generated class omits %q", required)
		}
	}
	for _, forbidden := range [][]byte{
		[]byte("javax/tools/JavaCompiler"),
		[]byte("java/nio/charset/StandardCharsets"),
		[]byte("java/util/Base64"),
		[]byte("public class RenvoProgram"),
	} {
		if bytes.Contains(class, forbidden) {
			t.Errorf("generated class contains forbidden source/runtime dependency %q", forbidden)
		}
	}
}

func TestCompilerJITJVMExecutesWhenJavaIsAvailable(t *testing.T) {
	java, err := exec.LookPath("java")
	if err != nil {
		t.Skip("java is not installed")
	}
	class, _ := compileJVMExample(t)
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "RenvoProgram.class"), class, 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(java, "-Xverify:all", "-cp", directory, "RenvoProgram")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("generated class failed JVM verification/execution: %v\n%s", err, output)
	}
	if !bytes.HasPrefix(output, []byte("Renvo on ")) || !bytes.Contains(output, []byte(" / Java ")) {
		t.Fatalf("generated JVM interop output = %q", output)
	}
}
