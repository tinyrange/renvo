package frontend_tests

import (
	"bytes"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

// Compile the self-hosted frontend entirely in memory so this also checks the
// renvo-tagged embed adapter on hosts that cannot execute Linux artifacts.
func TestFrontendMinimalAndFullBundleCompile(t *testing.T) {
	root := repoRoot(t)
	for _, tags := range []string{"", "renvo_bundle"} {
		name := "minimal"
		if tags != "" {
			name = "full"
		}
		t.Run(name, func(t *testing.T) {
			args := []string{"-t", "linux/amd64", "-s", "-arena-size", "134217728", "-o", "renvo"}
			if tags != "" {
				args = append(args, "-tags", tags)
			}
			args = append(args, "./cmd/renvo")
			result := driver.CompileFromFSWithModuleCache(args, root, "/std", "", driver.OSFS{}, backendcompiled.Backend{})
			if !result.Ok {
				t.Fatalf("self-hosted frontend compilation failed: %+v", result.Diagnostic)
			}
			if !bytes.HasPrefix(result.Binary, []byte("\x7fELF")) {
				t.Fatal("self-hosted frontend is not an ELF binary")
			}
			t.Logf("%s frontend payload: %d bytes", name, len(result.Binary))
			if int64(len(result.Binary)) > frontendPayloadMax {
				t.Fatalf("frontend payload=%dB > %dB", len(result.Binary), frontendPayloadMax)
			}
		})
	}
}
