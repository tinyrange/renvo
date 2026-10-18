package frontend_tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/backendjit"
	"renvo.dev/internal/driver"
)

func TestManagedHeap(t *testing.T)               { testManagedHeap(t, false) }
func TestManagedHeapBuiltin(t *testing.T)        { testManagedHeap(t, true) }
func TestManagedHeapDefinitionPort(t *testing.T) { testManagedHeap(t, false, true) }

type managedBuiltinTestBackend struct{ backendcompiled.Backend }

func (managedBuiltinTestBackend) CompileSourceWithArena(source []byte, target string, strip bool, limit int) driver.BackendResult {
	data, ok := backendcompiled.RenvoCompileSourceToBytesWithOptions(source, target, backendcompiled.RenvoCompileOptions{StripSymbols: strip, ArenaSize: limit})
	return driver.BackendResult{Binary: data, Ok: ok}
}

type managedTestBackend interface {
	driver.Backend
	CompileSourceWithArena([]byte, string, bool, int) driver.BackendResult
}

func testManagedHeap(t *testing.T, builtin bool, custom ...bool) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("Linux x86-64 managed runtime")
	}
	root := repoRoot(t)
	definition := filepath.Join(root, "backend/definitions/linux_amd64_desktop.rbe")
	target := "linux/amd64-desktop"
	if len(custom) != 0 && custom[0] {
		definition = managedPortableDefinition(t, root, 4096)
		target = "testhost/portable64"
	}
	var backend managedTestBackend = managedBuiltinTestBackend{}
	if !builtin {
		backend = backendjit.New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"), t.TempDir(), backendcompiled.Backend{})
	}
	compile := func(t *testing.T, source, limit string) string {
		t.Helper()
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module heapcase\nrequire renvo.dev v0.0.0\nreplace renvo.dev => "+root+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		src := filepath.Join(dir, "main.go")
		if err := os.WriteFile(src, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		args := []string{"-t", target, "-s", "-o", "app"}
		if !builtin {
			args = append(args, "-backend", definition)
		}
		if limit != "" {
			args = append(args, "-heap-limit", limit)
		}
		args = append(args, src)
		compiled := driver.CompileFromFS(args, root, filepath.Join(root, "std"), driver.OSFS{}, backend)
		if !compiled.Ok {
			t.Fatalf("compile: %#v", compiled.Diagnostic)
		}
		executable := filepath.Join(dir, "app")
		if err := os.WriteFile(executable, compiled.Binary, 0700); err != nil {
			t.Fatal(err)
		}
		return executable
	}
	run := func(t *testing.T, executable, override, want string, success bool) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, executable)
		cmd.Env = []string{}
		if override != "" {
			if strings.HasPrefix(override, "RENVO_HEAP_LIMIT=") {
				cmd.Env = append(cmd.Env, override)
			} else {
				cmd.Env = append(cmd.Env, "RENVO_HEAP_LIMIT="+override)
			}
		}
		output, err := cmd.CombinedOutput()
		if (err == nil) != success || string(output) != want {
			t.Fatalf("run override %q: %v %q, want success=%v %q", override, err, output, success, want)
		}
	}
	for _, tc := range managedHeapStressCases {
		t.Run("stress-"+tc.name, func(t *testing.T) {
			executable := compile(t, tc.source, tc.limit)
			run(t, executable, "", "PASS\n", true)
		})
	}
	t.Run("roots-and-churn", func(t *testing.T) {
		executable := compile(t, `package main
import "renvo.dev/x/runtime"
type Node struct { next *Node; data []byte }
var root = &Node{data:make([]byte, 128)}
func waste() { b:=make([]byte,4096); for i:=range b { b[i]=99 } }
func interior() *byte { b:=make([]byte,800); b[311]=73; return &b[311] }
func main() {
 root.data[70]=42
 root.next=&Node{data:make([]byte,129)}
 root.next.data[128]=81
 root.next.next=root
 p:=interior()
 for i:=0;i<200;i++ { waste() }
 runtime.Collect()
 if root.data[70]!=42 || root.next.data[128]!=81 || root.next.next!=root || *p!=73 {panic("lost root")}
 b:=make([]byte,4096); for _,v:=range b {if v!=0 {panic("not zero")}}
 println("PASS")
}`, "64KiB")
		run(t, executable, "", "PASS\n", true)
	})
	t.Run("budget-and-environment", func(t *testing.T) {
		executable := compile(t, `package main
var hold []byte
func main(){ hold=make([]byte,90000); hold[89999]=42; if hold[89999]!=42 {panic("bad")}; println("PASS") }
`, "64KiB")
		run(t, executable, "", "panic: out of memory\n", false)
		run(t, executable, "128KiB", "PASS\n", true)
		run(t, executable, "32KiB", "panic: out of memory\n", false)
		for _, bad := range []string{"RENVO_HEAP_LIMIT=", "0", "-1", "12MB", "999999999999999999999999", "1KiBx"} {
			run(t, executable, bad, "renvo: invalid RENVO_HEAP_LIMIT\n", false)
		}
	})
	t.Run("recover-oom", func(t *testing.T) {
		executable := compile(t, `package main
var live []byte
func attempt() { defer func(){v:=recover();if v==nil {panic("missing oom")};println("PASS")}(); live=make([]byte,1000000); println("UNEXPECTED") }
func main(){attempt()}
`, "64KiB")
		run(t, executable, "", "PASS\n", true)
	})
	t.Run("uncapped-growth", func(t *testing.T) {
		executable := compile(t, `package main
var data [24][]byte
func main(){for i:=range data {data[i]=make([]byte,100000);data[i][99999]=byte(i)};for i:=range data {if data[i][99999]!=byte(i){panic("lost")}};println("PASS")}
`, "")
		run(t, executable, "", "PASS\n", true)
	})

	t.Run("large-budget-not-reservation", func(t *testing.T) {
		executable := compile(t, `package main;func main(){b:=make([]byte,16);b[15]=7;println("PASS")}`, "2GiB")
		run(t, executable, "", "PASS\n", true)
	})
	t.Run("exact-page-budget", func(t *testing.T) {
		executable := compile(t, `package main
var data []byte
func main(){data=make([]byte,4032);data[4031]=91;if data[4031]!=91{panic("last byte")};println("PASS")}
`, "4256")
		run(t, executable, "", "PASS\n", true)
		run(t, executable, "4255", "panic: out of memory\n", false)
		executable = compile(t, `package main
var data []byte
func main(){data=make([]byte,4033);println("UNEXPECTED")}
`, "4256")
		run(t, executable, "", "panic: out of memory\n", false)
	})
	t.Run("too-small-for-metadata", func(t *testing.T) {
		executable := compile(t, `package main;func main(){println("PASS")}`, "1")
		run(t, executable, "", "panic: out of memory\n", false)
	})
	t.Run("containers", func(t *testing.T) {
		executable := compile(t, `package main
import "renvo.dev/x/runtime"
var table map[string][]byte
var callback func() string
var box any
func setup(){
 table=make(map[string][]byte)
 text:=string([]byte{'h','e','l','l','o'})
 callback=func()string{return text}
 p:=new(int);*p=731;box=p
 var data []byte
 for i:=0;i<800;i++ {data=append(data,byte(i))}
 table[text]=data
}
func churn(){b:=make([]byte,5000);b[4999]=3}
func main(){setup();for i:=0;i<120;i++{churn()};runtime.Collect();if callback()!="hello"||*box.(*int)!=731||table["hello"][799]!=31{panic("lost container")};println("PASS")}
`, "128KiB")
		run(t, executable, "", "PASS\n", true)
	})
	t.Run("collection-is-explicit", func(t *testing.T) {
		result := backend.CompileSourceWithArena([]byte(`package main
func renvo_runtime_HeapCollect() {}
func appMain()int {before:=__renvoManagedState[10];renvo_runtime_HeapCollect();if __renvoManagedState[10]!=before+1 {return 1};print("PASS\n");return 0}
`), target, true, 65536)
		if !result.Ok {
			t.Fatalf("compile collection boundary: %#v", result.Diagnostic)
		}
		path := filepath.Join(t.TempDir(), "collect")
		if err := os.WriteFile(path, result.Binary, 0700); err != nil {
			t.Fatal(err)
		}
		run(t, path, "", "PASS\n", true)
	})
	// This fixture uses Linux/amd64 syscall numbers. Keep it out of the
	// all-platform corpus; both native prepared and bundled paths execute it.
	t.Run("syscall-regression", func(t *testing.T) {
		source, err := os.ReadFile(filepath.Join(root, "backend/tests/raw_syscall_mmap_six_arguments.go.txt"))
		if err != nil {
			t.Fatal(err)
		}
		result := backend.CompileSourceWithArena(source, target, true, 65536)
		if !result.Ok {
			t.Fatalf("compile raw syscall: %#v", result.Diagnostic)
		}
		path := filepath.Join(t.TempDir(), "syscall")
		if err := os.WriteFile(path, result.Binary, 0700); err != nil {
			t.Fatal(err)
		}
		run(t, path, "", "PASS\n", true)
	})
	t.Run("metadata-is-charged", func(t *testing.T) {
		executable := compile(t, `package main
var data []byte
func main(){data=make([]byte,4096);println("PASS")}
`, "4096")
		run(t, executable, "", "panic: out of memory\n", false)
	})
	t.Run("arena-ignores-managed-environment", func(t *testing.T) {
		result := backend.CompileSourceWithArena([]byte(`package main
func renvo_runtime_HeapCollect() {}
func appMain()int {b:=make([]byte,128);b[127]=42;renvo_runtime_HeapCollect();if b[127]!=42{return 1};print("PASS\n");return 0}
`), "linux/amd64", true, 65536)
		if !result.Ok {
			t.Fatalf("compile arena: %#v", result.Diagnostic)
		}
		path := filepath.Join(t.TempDir(), "arena")
		if err := os.WriteFile(path, result.Binary, 0700); err != nil {
			t.Fatal(err)
		}
		run(t, path, "invalid", "PASS\n", true)
	})
	t.Run("arena-hints-preserve-managed-lifetimes", func(t *testing.T) {
		source := []byte(`package main
func renvo_runtime_ArenaMark()int{return 0}
func renvo_runtime_ArenaReset(mark int){}
func renvo_runtime_ArenaDiscard(start int,end int){}
func renvo_runtime_ArenaPersistBytes(b []byte)[]byte{return b}
func renvo_runtime_HeapCollect(){}
func appMain()int{mark:=renvo_runtime_ArenaMark();b:=make([]byte,3000);b[2999]=72;p:=renvo_runtime_ArenaPersistBytes(b);renvo_runtime_ArenaReset(mark);renvo_runtime_ArenaDiscard(mark,mark+65536);renvo_runtime_HeapCollect();if mark!=0||p[2999]!=72||&p[0]!=&b[0]{return 1};print("PASS\n");return 0}
`)
		result := backend.CompileSourceWithArena(source, target, true, 65536)
		if !result.Ok {
			t.Fatalf("arena hints: %#v", result.Diagnostic)
		}
		path := filepath.Join(t.TempDir(), "hints")
		if err := os.WriteFile(path, result.Binary, 0700); err != nil {
			t.Fatal(err)
		}
		run(t, path, "", "PASS\n", true)
	})
}
