package driver

import (
	"renvo.dev/internal/load"
	"strings"
	"testing"
)

func TestManagedPolicyRejectsLateSystemTarget(t *testing.T) {
	profile := strings.Replace(testSystemProfile, `target = "linux/amd64"`, `target = "linux/amd64-desktop"`, 1)
	fs := memorySourceFS{files: []load.SourceFile{{Path: "/repo/desktop.rtg", Src: []byte(profile)}}}
	got := ResolveFSOptions([]string{"-system", "desktop.rtg", "-o", "app", "main.go"}, "/repo", fs)
	if got.Ok || got.Error != ParseErrHeapLimit {
		t.Fatalf("profile bypassed desktop policy: %#v", got)
	}
	// An explicit arena target remains a supported override of a profile default.
	got = ResolveFSOptions([]string{"-system", "desktop.rtg", "-t", "linux/amd64", "-o", "app", "main.go"}, "/repo", fs)
	if !got.Ok || got.MemoryStrategy != "" {
		t.Fatalf("arena override: %#v", got)
	}
}

func TestManagedPolicyUsesResolvedDefinitionNotPublicName(t *testing.T) {
	// Public names are not allocator contracts: this definition is an ordinary
	// arena target despite sharing the built-in desktop target's spelling.
	definition := strings.ReplaceAll(testBackendDefinition, "acme/aarch64", "linux/amd64-desktop")
	fs := memorySourceFS{files: []load.SourceFile{{Path: "/repo/custom.rtg", Src: []byte(definition)}}}
	got := resolveBackendBuildOptions([]string{"-backend", "custom.rtg", "-t", "linux/amd64-desktop", "-arena-size", "65536", "-o", "app", "main.go"}, "/repo", fs)
	if !got.options.Ok || got.options.MemoryStrategy != "" || allocationBudget(got.options) != 65536 {
		t.Fatalf("builtin policy overrode definition: %#v", got.options)
	}
}
