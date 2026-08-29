//go:build !renvo

package backendjit

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"hash/adler32"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/internal/apk"
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

func compileAndroidDEXExample(t *testing.T) []byte {
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
		"-t", "android/vm32",
		"-s",
		"-o", "classes.dex",
		filepath.Join(root, "backend", "tests", "bitwise_or_sets_high_bit.go"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			backendJITTestCacheDir, backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("Android/JVM CompilerJIT compile failed: %#v", result.Diagnostic)
	}
	return result.Binary
}

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

func TestCompilerJITAndroidDEXImage(t *testing.T) {
	dex := compileAndroidDEXExample(t)
	if len(dex) < 112 || !bytes.Equal(dex[:8], []byte("dex\n035\x00")) {
		t.Fatalf("Android/JVM output has no DEX magic: %x", dex[:min(len(dex), 16)])
	}
	if got := int(binary.LittleEndian.Uint32(dex[32:36])); got != len(dex) {
		t.Fatalf("DEX file_size = %d, want %d", got, len(dex))
	}
	if got := binary.LittleEndian.Uint32(dex[8:12]); got != adler32.Checksum(dex[12:]) {
		t.Fatalf("DEX Adler-32 = %#x, want %#x", got, adler32.Checksum(dex[12:]))
	}
	wantSHA1 := sha1.Sum(dex[32:])
	if !bytes.Equal(dex[12:32], wantSHA1[:]) {
		t.Fatal("DEX SHA-1 signature is invalid")
	}
	for _, required := range [][]byte{
		[]byte("LRenvoProgram;"),
		[]byte("Ldev/renvo/app/RenvoActivity;"),
		[]byte("Landroid/app/Activity;"),
	} {
		if !bytes.Contains(dex, required) {
			t.Errorf("generated DEX omits %q", required)
		}
	}
}

func TestRenvoBuiltAPKPackagerBuildsAndroidDEX(t *testing.T) {
	dex := compileAndroidDEXExample(t)
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	packager := driver.CompileFromFS([]string{
		"-t", hostTarget(), "-s", "-o", "renvoapk",
		filepath.Join(root, "cmd", "renvoapk"),
	}, root, filepath.Join(root, "std"), driver.OSFS{}, backendcompiled.Backend{})
	if !packager.Ok {
		t.Fatalf("Renvo packager compile failed: %#v", packager.Diagnostic)
	}
	directory := t.TempDir()
	executable := filepath.Join(directory, "renvoapk")
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	if err := os.WriteFile(executable, packager.Binary, 0o755); err != nil {
		t.Fatal(err)
	}
	dexPath := filepath.Join(directory, "classes.dex")
	if err := os.WriteFile(dexPath, dex, 0o644); err != nil {
		t.Fatal(err)
	}
	configSource := []byte("package=dev.renvo.jvmpackager\n" +
		"name=Renvo JVM Packager\nversion_code=1\nversion_name=1.0\n" +
		"min_sdk=24\ntarget_sdk=35\n")
	configPath := filepath.Join(directory, "app.conf")
	if err := os.WriteFile(configPath, configSource, 0o644); err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(directory, "app.apk")
	command := exec.Command(executable,
		"-dex", dexPath, "-config", configPath, "-o", outputPath)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("Renvo-built DEX packager failed: %v\n%s", err, output)
	}
	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	config, err := apk.ParseConfig(configSource)
	if err != nil {
		t.Fatal(err)
	}
	want, err := apk.BuildDEX(dex, config)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("Renvo-built DEX packager differs from host builder")
	}
}
