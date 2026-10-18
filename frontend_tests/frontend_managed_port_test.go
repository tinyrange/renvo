package frontend_tests

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/backendjit"
	"renvo.dev/internal/driver"
	"runtime"
	"strings"
	"testing"
	"time"
)

// This is a host-executable adapter, not an emulated new OS. Rename every
// policy identity and change mapping granularity solely through definitions.
func managedPortableDefinition(t *testing.T, root string, page int) string {
	t.Helper()
	dir := t.TempDir()
	files, err := filepath.Glob(filepath.Join(root, "backend/definitions/*.rtg"))
	if err != nil {
		t.Fatal(err)
	}
	names := strings.NewReplacer(
		"linux/amd64-desktop", "testhost/portable64",
		"arch x86_64 {", "arch portable64 {", "arch = x86_64", "arch = portable64",
		"abi sysv_x86_64 {", "abi portable_abi {", "abi = sysv_x86_64", "abi = portable_abi",
		"runtime linux_x86_64 {", "runtime portable_runtime {", "runtime = linux_x86_64", "runtime = portable_runtime",
		"format elf64_x86_64 {", "format portable_format {", "executable = elf64_x86_64", "executable = portable_format",
		"os = linux", "os = testhost",
		"const __renvoManagedPageSize = uintptr(4096)", fmt.Sprintf("const __renvoManagedPageSize = uintptr(%d)", page),
	)
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, filepath.Base(file)), []byte(names.Replace(string(data))), 0600); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, "portable.rbe")
	if err := os.WriteFile(path, []byte("@import \"linux_amd64.rtg\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestManagedHeapDefinitionGranularity(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("host executable adapter")
	}
	root := repoRoot(t)
	definition := managedPortableDefinition(t, root, 8192)
	backend := backendjit.New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"), t.TempDir(), backendcompiled.Backend{})
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte(`package main;func main(){b:=make([]byte,1);b[0]=42;if b[0]!=42{panic("value")};println("PASS")}`), 0600); err != nil {
		t.Fatal(err)
	}
	result := driver.CompileFromFS([]string{"-t", "testhost/portable64", "-backend", definition, "-heap-limit", "8352", "-s", "-o", "app", src}, root, filepath.Join(root, "std"), driver.OSFS{}, backend)
	if !result.Ok {
		t.Fatalf("compile: %#v", result.Diagnostic)
	}
	bin := filepath.Join(dir, "app")
	if err := os.WriteFile(bin, result.Binary, 0700); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		limit, want string
		ok          bool
	}{
		{"8352", "PASS\n", true}, {"8351", "panic: out of memory\n", false}, {"4256", "panic: out of memory\n", false},
	} {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, bin)
		cmd.Env = []string{"RENVO_HEAP_LIMIT=" + test.limit}
		out, err := cmd.CombinedOutput()
		cancel()
		if (err == nil) != test.ok || string(out) != test.want {
			t.Fatalf("limit %s: %v %q", test.limit, err, out)
		}
	}
}
