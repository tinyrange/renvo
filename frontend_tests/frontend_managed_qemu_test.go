package frontend_tests

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// This opt-in test deliberately fails missing adapters/emulators. It is not an
// arena smoke and must never turn an unsupported row into a skipped/pass result.
func TestManagedHeapQEMU(t *testing.T) {
	if os.Getenv("RENVO_QEMU_MANAGED") != "1" {
		t.Skip("opt-in Linux managed QEMU matrix")
	}
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "tools/qemu-smoke/targets.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Emulator  string   `json:"emulator"`
		Target    string   `json:"target"`
		Backend   string   `json:"backend"`
		Arguments []string `json:"arguments"`
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("empty QEMU matrix")
	}
	dir := t.TempDir()
	compiler := filepath.Join(dir, "renvo")
	command := func(cwd string, env []string, name string, args ...string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = cwd
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		return out, err
	}
	if out, err := command(root, os.Environ(), "go", "build", "-o", compiler, "./cmd/renvo"); err != nil {
		t.Fatalf("build driver: %v\n%s", err, out)
	}
	write := func(t *testing.T, path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(t, filepath.Join(dir, "go.mod"), "module managedqemu\nrequire renvo.dev v0.0.0\nreplace renvo.dev => "+root+"\n")
	source := filepath.Join(dir, "main.go")
	write(t, source, `package main
import "renvo.dev/x/runtime"
var global []byte
type node struct { value uintptr; next *node }
func chain() *node { var head *node; for i:=0;i<24;i++ { head=&node{value:uintptr(i+1),next:head} }; return head }
func checkChain(head *node) { for n:=uintptr(24);n>0;n-- { if head==nil||head.value!=n { panic("lost graph") }; head=head.next }; if head!=nil { panic("bad graph") } }
func waste(){b:=make([]byte,8192);b[8191]=91}
func tooLarge()(oom bool){defer func(){if recover()!=nil{oom=true}}();b:=make([]byte,1024*1024);b[0]=1;return false}
func main(){
 graph:=chain()
 global=make([]byte,8192);global[7999]=73
 live:=make([]byte,8192);live[8191]=42;interior:=&live[8191];live=nil
 for i:=0;i<240;i++{waste()}
 runtime.Collect()
 checkChain(graph)
 if global[7999]!=73||*interior!=42{panic("lost roots")}
 if !tooLarge(){panic("missing OOM")}
 runtime.Collect()
 for i:=0;i<80;i++{waste()}
 checkChain(graph)
 if global[7999]!=73||*interior!=42{panic("lost roots after OOM")}
 println("PASS")
}`)
	maintained := map[string]string{"aarch64": "linux_aarch64.rtg", "arm": "linux_arm.rtg", "i386": "linux_386.rtg", "x86_64": "linux_amd64.rtg"}
	// Preserve relative imports for maintained definitions in the isolated fixture.
	files, err := filepath.Glob(filepath.Join(root, "backend/definitions/*.rtg"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(dir, filepath.Base(file)), string(data))
	}

	type result struct {
		Emulator string `json:"emulator"`
		Target   string `json:"target"`
		Passed   bool   `json:"passed"`
	}
	results := make([]result, 0, len(rows))
	for _, row := range rows {
		ok := t.Run(row.Emulator, func(t *testing.T) {
			qemu, err := exec.LookPath("qemu-" + row.Emulator)
			if err != nil {
				t.Fatal(err)
			}
			original := filepath.Join(root, row.Backend)
			if row.Backend == "" {
				filename, ok := maintained[row.Emulator]
				if !ok {
					t.Fatalf("no maintained definition for %s", row.Emulator)
				}
				original = filepath.Join(root, "backend/definitions", filename)
			}
			raw, err := os.ReadFile(original)
			if err != nil {
				t.Fatal(err)
			}
			text := string(raw)
			marker := "target " + row.Target + " {"
			if strings.Count(text, marker) != 1 {
				t.Fatalf("expected one declaration for %s", row.Target)
			}
			start := strings.Index(text, marker) + len(marker)
			end := strings.Index(text[start:], "\n}")
			if end < 0 {
				t.Fatal("unterminated target declaration")
			}
			body := text[start : start+end]
			if strings.Contains(body, "memory_strategy = arena") {
				body = strings.Replace(body, "memory_strategy = arena", "memory_strategy = desktop", 1)
			} else if !strings.Contains(body, "memory_strategy = desktop") {
				body = "\n\tmemory_strategy = desktop" + body
			}
			text = text[:start] + body + text[start+end:]
			definition := filepath.Join(dir, "managed_candidate_"+row.Emulator+".rbe")
			write(t, definition, text)
			executable := filepath.Join(dir, row.Emulator)
			build := func(limit string) {
				t.Helper()
				out, err := command(dir, os.Environ(), compiler, "-backend", definition, "-t", row.Target, "-heap-limit", limit, "-s", "-o", executable, source)
				if err != nil {
					t.Fatalf("managed compile (%s): %v\n%s", limit, err, out)
				}
			}
			run := func(override, want string, wantCode int) {
				t.Helper()
				args := append([]string{}, row.Arguments...)
				args = append(args, executable)
				env := []string{}
				if override != "" {
					env = append(env, "RENVO_HEAP_LIMIT="+override)
				}
				out, err := command(dir, env, qemu, args...)
				code := 0
				if err != nil {
					exit, ok := err.(*exec.ExitError)
					if !ok {
						t.Fatalf("QEMU execution: %v\n%s", err, out)
					}
					code = exit.ExitCode()
				}
				if code != wantCode || string(out) != want {
					t.Fatalf("override %q: exit %d, output %q; want exit %d, output %q", override, code, out, wantCode, want)
				}
			}
			build("64KiB")
			run("", "PASS\n", 0)
			run("1", "panic: out of memory\n", 2)
			run("invalid", "renvo: invalid RENVO_HEAP_LIMIT\n", 2)
			run("9223372036854775808", "renvo: invalid RENVO_HEAP_LIMIT\n", 2)
			if strings.Contains(text, "managed_word_bits = 32") {
				run("1TiB", "renvo: invalid RENVO_HEAP_LIMIT\n", 2)
			}
			build("1")
			run("", "panic: out of memory\n", 2)
			run("64KiB", "PASS\n", 0)
		})
		results = append(results, result{Emulator: row.Emulator, Target: row.Target, Passed: ok})
	}
	report, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	reportDir := filepath.Join(root, "sandbox/agent-profiles")
	if err := os.MkdirAll(reportDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "managed-qemu.json"), append(report, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
