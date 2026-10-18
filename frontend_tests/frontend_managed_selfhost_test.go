package frontend_tests

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

func TestManagedHeapSelfHost(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("Linux x86-64 managed runtime")
	}
	root := repoRoot(t)
	dir := t.TempDir()
	target := "linux/amd64-desktop"
	args := []string{"-t", target, "-heap-limit", selfHostedFrontendArenaSize, "-s", "-o", "compiler", "./cmd/renvo"}
	built := driver.CompileFromFS(args, root, filepath.Join(root, "std"), driver.OSFS{}, backendcompiled.Backend{})
	if !built.Ok {
		t.Fatalf("stage1: %#v", built.Diagnostic)
	}
	stage1 := filepath.Join(dir, "stage1")
	if err := os.WriteFile(stage1, built.Binary, 0700); err != nil {
		t.Fatal(err)
	}
	run := func(binary string, args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = root
		cmd.Env = []string{"PWD=" + root, "RENVO_STDROOT=" + filepath.Join(root, "std")}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v: %v\n%s", filepath.Base(binary), args, err, out)
		}
		return out
	}
	assertManaged := func(binary string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, "--help")
		cmd.Env = []string{"RENVO_HEAP_LIMIT=1"}
		out, err := cmd.CombinedOutput()
		exit, ok := err.(*exec.ExitError)
		if !ok || exit.ExitCode() != 2 || string(out) != "panic: out of memory\n" {
			t.Fatalf("%s is not a managed executable: %v %q", binary, err, out)
		}
	}
	assertManaged(stage1)
	run(stage1, "--help")
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module managedselfhost\nrequire renvo.dev v0.0.0\nreplace renvo.dev => "+root+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	smoke := filepath.Join(dir, "smoke.go")
	if err := os.WriteFile(smoke, []byte(`package main;import "renvo.dev/x/runtime";func waste(){b:=make([]byte,8192);b[8191]=21};func main(){b:=make([]byte,8192);b[8191]=73;for i:=0;i<80;i++{waste()};runtime.Collect();if b[8191]!=73{panic("root")};println("PASS")}`), 0600); err != nil {
		t.Fatal(err)
	}
	smokeBin := filepath.Join(dir, "smoke")
	run(stage1, "-t", target, "-heap-limit", "64KiB", "-s", "-o", smokeBin, smoke)
	if got := run(smokeBin); string(got) != "PASS\n" {
		t.Fatalf("smoke: %q", got)
	}
	stage2 := filepath.Join(dir, "stage2")
	stage3 := filepath.Join(dir, "stage3")
	for _, pair := range [][2]string{{stage1, stage2}, {stage2, stage3}} {
		run(pair[0], "-t", target, "-heap-limit", selfHostedFrontendArenaSize, "-s", "-o", pair[1], "./cmd/renvo")
		assertManaged(pair[1])
	}
	second, err := os.ReadFile(stage2)
	if err != nil {
		t.Fatal(err)
	}
	third, err := os.ReadFile(stage3)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(second, third) {
		t.Fatal("managed stage2/stage3 fixed point differs")
	}
	run(stage3, "-t", target, "-heap-limit", "64KiB", "-s", "-o", smokeBin, smoke)
	if got := run(smokeBin); string(got) != "PASS\n" {
		t.Fatalf("stage3 smoke: %q", got)
	}
	// Exercise the embedded backend API's managed range, not only its legacy
	// arena-compatible middle. A large cap is not an eager reservation.
	run(stage3, "-t", target, "-heap-limit", "2GiB", "-s", "-o", smokeBin, smoke)
	if got := run(smokeBin); string(got) != "PASS\n" {
		t.Fatalf("stage3 large budget: %q", got)
	}
	run(stage3, "-t", target, "-heap-limit", "1", "-s", "-o", smokeBin, smoke)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, smokeBin)
	cmd.Env = []string{}
	out, err := cmd.CombinedOutput()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 2 || string(out) != "panic: out of memory\n" {
		t.Fatalf("stage3 tiny compiled budget: %v %q", err, out)
	}
}
