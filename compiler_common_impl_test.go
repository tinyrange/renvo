package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestBuildMetaHandlesMoreThanInitialFuncCapacity(t *testing.T) {
	src := []byte("package main\n")
	for i := 0; i < 1301; i++ {
		name := strconv.Itoa(i)
		src = append(src, []byte("func f"+name+"() int { return "+name+" }\n")...)
	}
	src = append(src, []byte("func appMain(args []string, env []string) int { return f1300() }\n")...)

	prog := rtgParseProgram(src)
	if !prog.ok {
		t.Fatalf("failed to parse generated source")
	}
	meta := rtgBuildMeta(&prog)
	if !meta.ok {
		t.Fatalf("failed to build metadata")
	}
	if len(meta.funcs) != 1302 {
		t.Fatalf("metadata function count = %d, want 1302", len(meta.funcs))
	}
}

func TestArbitrarySyscallLinuxAmd64Write(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("linux/amd64 syscall execution test requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	src := []byte(`package main

func syscall(num int, fd int, msg string, n int) int { return 0 }

func appMain(args []string, env []string) int {
	syscall(1, 1, "PASS\n", 5)
	return 0
}
`)
	data, ok := RtgCompileSourceToBytes(src, "linux/amd64")
	if !ok {
		t.Fatalf("RtgCompileSourceToBytes failed")
	}
	out := filepath.Join(t.TempDir(), "syscall-write")
	if err := os.WriteFile(out, data, 0755); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	cmd := exec.Command(out)
	got, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled syscall test failed: %v\n%s", err, string(got))
	}
	if string(got) != "PASS\n" {
		t.Fatalf("compiled syscall output = %q, want PASS", string(got))
	}
}

func TestLinkStaticAddsWindowsImport(t *testing.T) {
	src := []byte(`package main

// rtg:linkstatic user32.dll,MessageBeep
func messageBeep(kind int) int { return 0 }

func appMain(args []string, env []string) int {
	return messageBeep(0)
}
`)
	for _, target := range []string{"windows/amd64", "windows/386"} {
		target := target
		t.Run(target, func(t *testing.T) {
			data, ok := RtgCompileSourceToBytes(src, target)
			if !ok {
				t.Fatalf("RtgCompileSourceToBytes failed")
			}
			text := string(data)
			for _, want := range []string{"user32.dll", "MessageBeep"} {
				if !strings.Contains(text, want) {
					t.Fatalf("windows import table missing %q", want)
				}
			}
		})
	}
}
