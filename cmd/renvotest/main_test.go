//go:build !renvo

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunGeneratesPackage(t *testing.T) {
	srcDir := t.TempDir()
	writeCmdTestFile(t, srcDir, "math.go", `package math

func one() int { return 1 }
`)
	writeCmdTestFile(t, srcDir, "math_test.go", `package math

import "testing"

func TestOne(t *testing.T) {
	if one() != 1 {
		t.Fatalf("bad value")
	}
}
`)
	outDir := t.TempDir()
	if code := run([]string{"-o", outDir, srcDir}); code != 0 {
		t.Fatalf("run exit = %d, want 0", code)
	}
	if _, err := os.Stat(filepath.Join(outDir, "renvo_testmain.go")); err != nil {
		t.Fatalf("generated runner missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "renvotest_math_testsrc.go")); err != nil {
		t.Fatalf("generated test source missing: %v", err)
	}
}

func TestRunRejectsInvalidArgs(t *testing.T) {
	if code := run([]string{}); code != 2 {
		t.Fatalf("missing output exit = %d, want 2", code)
	}
	if code := run([]string{"-o", t.TempDir()}); code != 2 {
		t.Fatalf("missing package exit = %d, want 2", code)
	}
}

func writeCmdTestFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
