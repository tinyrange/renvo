//go:build !renvo

package driver

import (
	"os"
	"path/filepath"
	"renvo.dev/internal/load"
	"testing"
)

func TestBundleEnvironmentDefaults(t *testing.T) {
	if got := StdRootFromEnv(nil); got != "/std" {
		t.Errorf("standard library root = %q", got)
	}
	wantCache := ""
	if renvoBundledExtrasEnabled {
		wantCache = "/modules"
	}
	if got := ModuleCacheFromEnv(nil); got != wantCache {
		t.Errorf("module cache = %q, want %q", got, wantCache)
	}
	if got := StdRootFromEnv([]string{"RENVO_STDROOT=/custom/std"}); got != "/custom/std" {
		t.Errorf("standard library override = %q", got)
	}
	if got := ModuleCacheFromEnv([]string{"RENVO_MODCACHE=/custom/modules"}); got != "/custom/modules" {
		t.Errorf("module cache override = %q", got)
	}
}

func TestDefaultBundleResolvesImplicitConcurrency(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app\n\ngo 1.22\n"), 0644); err != nil {
		t.Fatal(err)
	}
	source := []byte("package main\nfunc main() { values:=make(chan int,1); values<-7; print(<-values) }\n")
	if err := os.WriteFile(filepath.Join(root, "main.go"), source, 0644); err != nil {
		t.Fatal(err)
	}
	result := BuildFromFSWithModuleCache([]string{"-t", "linux/amd64", "-o", "app", "."}, root, "/std", "", OSFS{})
	if !result.Ok {
		t.Fatalf("default bundle concurrency build failed: %#v", result.Diagnostic)
	}
}

func TestMinimalBundleHashesCustomSources(t *testing.T) {
	if renvoBundledExtrasEnabled {
		t.Skip("full bundle retains its existing immutable-source optimization")
	}
	for _, path := range []string{"/std/fmt/fmt.go", "/modules/example.com/lib@v1.0.0/lib.go"} {
		beforeA, beforeB := embeddedBuildFingerprint("/app", Options{}, []load.SourceFile{{Path: path, Src: []byte("package lib; const Value = 1")}})
		afterA, afterB := embeddedBuildFingerprint("/app", Options{}, []load.SourceFile{{Path: path, Src: []byte("package lib; const Value = 2")}})
		if beforeA == afterA && beforeB == afterB {
			t.Errorf("same-length edit to caller-provided source %q did not invalidate the cache", path)
		}
	}
}

func TestDefaultBundleThroughOSFS(t *testing.T) {
	fs := OSFS{}
	for _, path := range []string{"/std", "/std/fmt", "/std/fmt/fmt_renvo.go"} {
		if !fs.PathExists(path) {
			t.Errorf("missing embedded path %q", path)
		}
	}
	if data, ok := fs.ReadFile("/std/fmt/fmt_renvo.go"); !ok || len(data) == 0 {
		t.Fatal("default standard library source is unavailable")
	}
}
