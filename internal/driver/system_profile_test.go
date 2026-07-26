package driver

import (
	"bytes"
	"strings"
	"testing"

	"renvo.dev/internal/load"
)

const testSystemProfile = `system "small-linux-amd64" {
    target = "linux/amd64"
    binary = 2MiB
    arena = 32MiB
}
`

func TestParseSystemProfile(t *testing.T) {
	profile, message, ok := parseSystemProfile([]byte(testSystemProfile))
	if !ok {
		t.Fatalf("parseSystemProfile failed: %s", message)
	}
	if profile.Name != "small-linux-amd64" || profile.Target != "linux/amd64" ||
		profile.BinaryLimit != 2*1024*1024 || profile.ArenaSize != 32*1024*1024 {
		t.Fatalf("profile = %#v", profile)
	}
}

func TestParseSystemProfileRejectsIncompleteAndUnknownProfiles(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{source: `target = "linux/amd64"`, want: "system declaration"},
		{source: `system "" { target = "linux/amd64" binary = 1MiB arena = 1MiB }`, want: "system name"},
		{source: `system "small" { binary = 1MiB arena = 1MiB }`, want: "missing target"},
		{source: `system "small" { target = "linux/amd64" arena = 1MiB }`, want: "missing binary"},
		{source: `system "small" { target = "linux/amd64" binary = 1MiB }`, want: "missing arena"},
		{source: `system "small" { target = "linux/amd64" binary = 1MiB arena = 128B }`, want: "arena must"},
		{source: `system "small" { target = "linux/amd64" binary = 1MiB arena = 1MiB stack = 1KiB }`, want: "unknown system field"},
	}
	for i := 0; i < len(tests); i++ {
		_, message, ok := parseSystemProfile([]byte(tests[i].source))
		if ok || !strings.Contains(message, tests[i].want) {
			t.Errorf("parseSystemProfile(%q) = ok=%v message=%q, want %q", tests[i].source, ok, message, tests[i].want)
		}
	}
}

func TestBuildFromFSLoadsSystemProfile(t *testing.T) {
	files := append(driverTestFiles(), load.SourceFile{Path: "/repo/case/small.rtg", Src: []byte(testSystemProfile)})
	result := BuildFromFS([]string{"-system", "small.rtg", "-o", "app", "./cmd/app"}, "/repo/case", "/std", memorySourceFS{files: files})
	if !result.Ok {
		t.Fatalf("BuildFromFS failed: %#v", result.Diagnostic)
	}
	options := result.Options
	if options.SystemName != "small-linux-amd64" || options.Target != "linux/amd64" ||
		options.BinaryLimit != 2*1024*1024 || options.ArenaSize != 32*1024*1024 {
		t.Fatalf("resolved options = %#v", options)
	}
}

func TestBuildFromFSRejectsInvalidSystemOptions(t *testing.T) {
	files := append(driverTestFiles(), load.SourceFile{Path: "/repo/case/bad.rtg", Src: []byte(`system "bad" { target = "linux/mips" binary = 1MiB arena = 1MiB }`)})
	tests := []struct {
		args []string
		code string
	}{
		{args: []string{"-system", "missing.rtg", "-o", "app", "./cmd/app"}, code: "RENVO-OPTION-020"},
		{args: []string{"-system", "bad.rtg", "-o", "app", "./cmd/app"}, code: "RENVO-OPTION-021"},
		{args: []string{"-system", "bad.rtg", "-t", "linux/amd64", "-o", "app", "./cmd/app"}, code: "RENVO-OPTION-022"},
		{args: []string{"-system", "bad.rtg", "-arena-size", "65536", "-o", "app", "./cmd/app"}, code: "RENVO-OPTION-023"},
	}
	for i := 0; i < len(tests); i++ {
		result := BuildFromFS(tests[i].args, "/repo/case", "/std", memorySourceFS{files: files})
		if result.Ok || result.Diagnostic.Code != tests[i].code {
			t.Errorf("BuildFromFS(%q) = %#v, want %s", tests[i].args, result, tests[i].code)
		}
	}
}

type systemProfileBackend struct {
	binary    []byte
	target    string
	arenaSize int
}

func (b *systemProfileBackend) CompileUnit([]byte, string, bool, bool) BackendResult {
	return BackendResult{Diagnostic: Diagnostic{Phase: "backend", Code: "TEST-SYSTEM-001", Message: "arena-aware entrypoint was not used"}}
}

func (b *systemProfileBackend) CompileUnitWithArena(_ []byte, target string, _ bool, _ bool, arenaSize int) BackendResult {
	b.target = target
	b.arenaSize = arenaSize
	return BackendResult{Binary: b.binary, Ok: true}
}

func TestCompileEnforcesSystemProfile(t *testing.T) {
	profile := `system "tiny" { target = "linux/amd64" binary = 6B arena = 64KiB }`
	files := append(driverTestFiles(), load.SourceFile{Path: "/repo/case/tiny.rtg", Src: []byte(profile)})
	fs := memorySourceFS{files: files}

	backend := &systemProfileBackend{binary: []byte("PASS\n")}
	result := CompileFromFS([]string{"-system", "tiny.rtg", "-o", "app", "./cmd/app"}, "/repo/case", "/std", fs, backend)
	if !result.Ok || !bytes.Equal(result.Binary, backend.binary) {
		t.Fatalf("within-budget compile = %#v", result)
	}
	if backend.target != "linux/amd64" || backend.arenaSize != 64*1024 {
		t.Fatalf("backend target/arena = %q/%d", backend.target, backend.arenaSize)
	}

	backend.binary = []byte("TOO LARGE")
	result = CompileFromFS([]string{"-system", "tiny.rtg", "-o", "app", "./cmd/app"}, "/repo/case", "/std", fs, backend)
	if result.Ok || result.Error != CompileErrSystem || result.Diagnostic.Code != "RENVO-SYSTEM-002" || len(result.Binary) != 0 {
		t.Fatalf("over-budget compile = %#v", result)
	}
	for _, want := range []string{"tiny", "9 bytes", "6 bytes", "3 bytes"} {
		if !strings.Contains(result.Diagnostic.Message, want) {
			t.Errorf("diagnostic %q missing %q", result.Diagnostic.Message, want)
		}
	}
}

func TestSystemBinaryLimitIncludesBrowserPackaging(t *testing.T) {
	profile := `system "tiny-browser" { target = "browser/wasm32" binary = 100B arena = 64KiB }`
	files := append(driverTestFiles(), load.SourceFile{Path: "/repo/case/browser.rtg", Src: []byte(profile)})
	backend := &systemProfileBackend{binary: []byte{0, 'a', 's', 'm'}}
	result := CompileFromFS([]string{"-system", "browser.rtg", "-o", "app.html", "./cmd/app"}, "/repo/case", "/std", memorySourceFS{files: files}, backend)
	if result.Ok || result.Diagnostic.Code != "RENVO-SYSTEM-002" || backend.target != "wasi/wasm32" {
		t.Fatalf("browser system compile = %#v; backend target = %q", result, backend.target)
	}
}
