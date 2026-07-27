//go:build !renvo

package backendjit

import (
	"os"
	"path/filepath"
	"testing"

	"renvo.dev/internal/rtg"
)

func TestExcludedFamily(t *testing.T) {
	excluded := excludedFamily(rtg.TargetDescriptor{ISA: "amd64"})
	for _, name := range []string{"compiler_amd64_impl.go", "compiler_linux_amd64_impl.go", "compiler_windows_amd64_impl.go"} {
		if !excluded[name] {
			t.Errorf("%s was not excluded", name)
		}
	}
	if excluded["compiler_aarch64_impl.go"] {
		t.Fatal("unrelated backend was excluded")
	}
}

func TestPublishIsReusable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache", "entry.rtgb")
	if err := publish(path, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := publish(path, []byte("second")); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(source) != "second" {
		t.Fatalf("cache source = %q", source)
	}
}
