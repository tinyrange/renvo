//go:build !rtg

package testfront

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGeneratePackageWritesRunnableTestingMain(t *testing.T) {
	srcDir := t.TempDir()
	writeTestFile(t, srcDir, "calc.go", `package calc

func add(a int, b int) int {
	return a + b
}
`)
	writeTestFile(t, srcDir, "calc_test.go", `package calc

import checktest "testing"

func TestAdd(t *checktest.T) {
	if add(2, 3) != 5 {
		t.Fatalf("bad sum")
	}
}

func Testlower(t *checktest.T) {
	t.Fatalf("should not be discovered")
}
`)

	result, err := GeneratePackage(srcDir)
	if err != nil {
		t.Fatalf("GeneratePackage failed: %v", err)
	}
	if result.Package != "calc" {
		t.Fatalf("package = %q, want calc", result.Package)
	}
	if !reflect.DeepEqual(result.Tests, []string{"TestAdd"}) {
		t.Fatalf("tests = %#v, want TestAdd", result.Tests)
	}
	if !generatedFileContains(result.Files, "calc.go", []byte("package main")) {
		t.Fatalf("generated source did not rewrite package: %#v", result.Files)
	}
	if !generatedFileContains(result.Files, "rtgtest_calc_testsrc.go", []byte("func TestAdd")) {
		t.Fatalf("generated test source missing or still has _test.go name: %#v", result.Files)
	}
	if !generatedFileContains(result.Files, "rtg_testmain.go", []byte("testing.Main")) {
		t.Fatalf("generated runner missing testing.Main")
	}

	outDir := t.TempDir()
	if err := WritePackage(outDir, result); err != nil {
		t.Fatalf("WritePackage failed: %v", err)
	}
	writeTestFile(t, outDir, "go.mod", "module generated.test\n")
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = outDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated runner failed: %v\n%s", err, string(output))
	}
	if !bytes.Contains(output, []byte("PASS")) {
		t.Fatalf("generated runner output = %q, want PASS", string(output))
	}
}

func TestGeneratePackageRejectsNoTests(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "calc.go", `package calc

func add(a int, b int) int { return a + b }
`)
	_, err := GeneratePackage(dir)
	if !errors.Is(err, ErrNoTests) {
		t.Fatalf("GeneratePackage error = %v, want ErrNoTests", err)
	}
}

func TestGeneratePackageRejectsExternalTests(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "calc.go", `package calc

func add(a int, b int) int { return a + b }
`)
	writeTestFile(t, dir, "calc_test.go", `package calc_test

import "testing"

func TestAdd(t *testing.T) {}
`)
	_, err := GeneratePackage(dir)
	if !errors.Is(err, ErrExternalTests) {
		t.Fatalf("GeneratePackage error = %v, want ErrExternalTests", err)
	}
}

func TestGeneratePackageRejectsInvalidTestSignature(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "calc.go", `package calc

func add(a int, b int) int { return a + b }
`)
	writeTestFile(t, dir, "calc_test.go", `package calc

func TestAdd() {}
`)
	_, err := GeneratePackage(dir)
	if err == nil || !strings.Contains(err.Error(), "invalid test signature") {
		t.Fatalf("GeneratePackage error = %v, want invalid test signature", err)
	}
}

func generatedFileContains(files []GeneratedFile, name string, text []byte) bool {
	for _, file := range files {
		if file.Name == name && bytes.Contains(file.Data, text) {
			return true
		}
	}
	return false
}

func writeTestFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data = strings.TrimLeft(data, "\n")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
