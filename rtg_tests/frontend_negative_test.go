package rtg_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type negativeExpectation struct {
	Phase   string `json:"phase"`
	Code    string `json:"code"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

func TestFrontendNegativeCorpus(t *testing.T) {
	root := repoRoot(t)
	frontends := []struct {
		name   string
		config frontendConfig
	}{{name: "host", config: frontendCompiler(t, root)}}
	if os.Getenv(selfHostTestsEnv) == "1" {
		frontends = append(frontends, struct {
			name   string
			config frontendConfig
		}{name: "stage3", config: selfHostedFrontendCompiler(t, root)})
	}

	negativeRoot := filepath.Join(root, "rtg_tests", "negative")
	entries, err := os.ReadDir(negativeRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, frontend := range frontends {
		frontend := frontend
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			entry := entry
			t.Run(frontend.name+"/"+entry.Name(), func(t *testing.T) {
				runNegativeCorpusCase(t, frontend.config, filepath.Join(negativeRoot, entry.Name()))
			})
		}
	}
}

func runNegativeCorpusCase(t *testing.T, frontend frontendConfig, dir string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "expect.json"))
	if err != nil {
		t.Fatal(err)
	}
	var want negativeExpectation
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	if want.Phase == "" || want.Code == "" || want.File == "" || want.Line < 1 || want.Column < 1 || want.Message == "" {
		t.Fatalf("incomplete negative expectation: %#v", want)
	}

	output := filepath.Join(t.TempDir(), "app")
	cmd := exec.Command(frontend.compiler, "-t", frontend.target, "-s", "-o", output, "./cmd/app")
	cmd.Dir = dir
	env := append([]string(nil), frontend.env...)
	sentinel := ""
	if runtime.GOOS != "windows" && len(frontend.env) > 0 {
		sentinel = filepath.Join(t.TempDir(), "backend-invoked")
		backend := filepath.Join(t.TempDir(), "reject-backend")
		script := "#!/bin/sh\n: > '" + sentinel + "'\nexit 99\n"
		if err := os.WriteFile(backend, []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		env = setFrontendEnv(env, "RTG_BACKEND="+backend)
	}
	cmd.Env = frontendCommandEnv(env, dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("negative corpus case unexpectedly compiled")
	}
	wantPrefix := fmt.Sprintf("%s:%d:%d: error %s (%s): %s", filepath.Join(dir, filepath.FromSlash(want.File)), want.Line, want.Column, want.Code, want.Phase, want.Message)
	if !strings.Contains(string(out), wantPrefix) {
		t.Fatalf("diagnostic = %q, want %q", string(out), wantPrefix)
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("negative compilation left output %q (stat error %v)", output, statErr)
	}
	if sentinel != "" {
		if _, statErr := os.Stat(sentinel); !os.IsNotExist(statErr) {
			t.Fatalf("frontend invoked backend for rejected source (stat error %v)", statErr)
		}
	}
}
