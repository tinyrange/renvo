package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSyncExpectations(t *testing.T) {
	root := t.TempDir()
	backend := filepath.Join(root, "backend", "tests")
	frontend := filepath.Join(root, "frontend_tests", "quick", "sample")
	for _, dir := range []string{backend, filepath.Join(frontend, "cmd", "app")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(backend, "sample.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(frontend, "go.mod"), []byte("module example.com/sample\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tier := range []string{"extended", "regressions", "std_compat"} {
		if err := os.MkdirAll(filepath.Join(root, "frontend_tests", tier), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := syncExpectations(root, false); err == nil {
		t.Fatal("missing expectations unexpectedly passed")
	}
	if count, err := syncExpectations(root, true); err != nil || count != 2 {
		t.Fatalf("write count=%d err=%v", count, err)
	}
	if count, err := syncExpectations(root, false); err != nil || count != 2 {
		t.Fatalf("check count=%d err=%v", count, err)
	}
	backendExpected := filepath.Join(backend, "sample.expected")
	stderrExpected := []byte(`{"stdout":"","stderr":"PASS\n","exit_code":0}`)
	if err := os.WriteFile(backendExpected, stderrExpected, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := syncExpectations(root, false); err != nil {
		t.Fatalf("check structured backend expectation: %v", err)
	}
	if _, err := syncExpectations(root, true); err != nil {
		t.Fatalf("preserve structured backend expectation: %v", err)
	}
	if data, err := os.ReadFile(backendExpected); err != nil || string(data) != string(stderrExpected) {
		t.Fatalf("structured expectation changed: data=%q err=%v", string(data), err)
	}
}

func TestCheckedInExpectations(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := syncExpectations(root, false); err != nil {
		t.Fatal(err)
	}
}
