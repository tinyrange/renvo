package frontend_tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestFrontendSuiteTempCleanup(t *testing.T) {
	for _, mode := range []string{"pass", "fail", "interrupt", "terminate"} {
		t.Run(mode, func(t *testing.T) {
			if runtime.GOOS == "windows" && (mode == "interrupt" || mode == "terminate") {
				t.Skip("Windows does not support sending these signals with os.Process.Signal")
			}
			dir := t.TempDir()
			manifest := filepath.Join(dir, "paths")
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, executable, "-test.run=^TestFrontendSuiteTempHelper$")
			cmd.Env = append(os.Environ(), "RENVO_TEMP_CLEANUP_HELPER="+mode, "RENVO_TEMP_CLEANUP_MANIFEST="+manifest)
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			waited := false
			defer func() {
				if !waited {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			}()
			var data []byte
			for {
				data, err = os.ReadFile(manifest)
				if err == nil {
					break
				}
				select {
				case <-ctx.Done():
					t.Fatal("helper did not create its directories")
				case <-time.After(10 * time.Millisecond):
				}
			}
			paths := strings.Fields(string(data))
			if len(paths) != 2 {
				t.Fatalf("manifest = %q", data)
			}
			if mode == "interrupt" || mode == "terminate" {
				for _, path := range paths {
					if _, err := os.Stat(path); err != nil {
						t.Fatal("directory missing before signal:", err)
					}
				}
				var sig os.Signal = os.Interrupt
				if mode == "terminate" {
					sig = syscall.SIGTERM
				}
				if err := cmd.Process.Signal(sig); err != nil {
					t.Fatal(err)
				}
			}
			err = cmd.Wait()
			waited = true
			want := 0
			switch mode {
			case "fail":
				want = 1
			case "interrupt":
				want = 130
			case "terminate":
				want = 143
			}
			if cmd.ProcessState.ExitCode() != want {
				t.Fatalf("exit=%d want=%d: %v", cmd.ProcessState.ExitCode(), want, err)
			}
			for _, path := range paths {
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					t.Errorf("suite directory remains: %s (%v)", path, err)
				}
			}
		})
	}
}

func TestFrontendSuiteTempHelper(t *testing.T) {
	mode := os.Getenv("RENVO_TEMP_CLEANUP_HELPER")
	if mode == "" {
		return
	}
	var paths []string
	for _, prefix := range []string{"renvo-frontend-corpus-", "renvo-frontend-selfhost-"} {
		dir, err := frontendSuiteTempDir(prefix)
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, dir)
		if err := os.WriteFile(filepath.Join(dir, "test-artifact"), []byte("temporary"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(os.Getenv("RENVO_TEMP_CLEANUP_MANIFEST"), []byte(strings.Join(paths, "\n")), 0600); err != nil {
		t.Fatal(err)
	}
	switch mode {
	case "fail":
		t.Fatal("intentional helper failure")
	case "interrupt", "terminate":
		time.Sleep(30 * time.Second)
	}
}
