package rtg_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type frontendDiagnosticCase struct {
	name       string
	files      map[string]string
	wantCode   string
	wantFile   string
	wantDetail string
}

func TestFrontendStructuredDiagnostics(t *testing.T) {
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

	cases := []frontendDiagnosticCase{
		{
			name:       "syntax",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main( {\n"},
			wantCode:   "RTG-PARSE-001",
			wantFile:   "cmd/app/main.go",
			wantDetail: "source syntax is invalid",
		},
		{
			name:       "unresolved_import",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nimport _ \"github.com/example/missing\"\n\nfunc main() {}\n"},
			wantCode:   "RTG-LOAD-008",
			wantFile:   "cmd/app/main.go",
			wantDetail: "unresolved import github.com/example/missing",
		},
		{
			name: "import_cycle",
			files: map[string]string{
				"cmd/app/main.go": "package main\n\nimport _ \"example.com/diagnostic/lib\"\n\nfunc main() {}\n",
				"lib/lib.go":      "package lib\n\nimport _ \"example.com/diagnostic/cmd/app\"\n",
			},
			wantCode:   "RTG-LOAD-011",
			wantFile:   "lib/lib.go",
			wantDetail: "import cycle detected",
		},
		{
			name:       "return_count",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc value() int { return }\nfunc main() { _ = value() }\n"},
			wantCode:   "RTG-CHECK-007",
			wantFile:   "cmd/app/main.go",
			wantDetail: "return value count does not match function results",
		},
		{
			name:       "assignment_type",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { var value int; value = \"wrong\"; _ = value }\n"},
			wantCode:   "RTG-CHECK-008",
			wantFile:   "cmd/app/main.go",
			wantDetail: "assignment value is not assignable to its destination",
		},
		{
			name:       "excluded_goroutine",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc work() {}\nfunc main() { go work() }\n"},
			wantCode:   "RTG-CHECK-009",
			wantFile:   "cmd/app/main.go",
			wantDetail: "feature is excluded from the current frontend scope",
		},
	}

	for _, frontend := range frontends {
		frontend := frontend
		for _, tc := range cases {
			tc := tc
			t.Run(frontend.name+"/"+tc.name, func(t *testing.T) {
				runFrontendDiagnosticCase(t, frontend.config, tc, nil)
			})
		}
	}
}

func TestFrontendBackendDiagnosticPreservesDetail(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("backend failure helper uses a POSIX shell script")
	}
	root := repoRoot(t)
	frontend := frontendCompiler(t, root)
	dir := t.TempDir()
	backend := filepath.Join(dir, "backend-failure")
	if err := os.WriteFile(backend, []byte("#!/bin/sh\necho 'intentional backend failure' >&2\nexit 23\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	tc := frontendDiagnosticCase{
		name:       "backend",
		files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() {}\n"},
		wantCode:   "RTG-BACKEND-003",
		wantDetail: "intentional backend failure",
	}
	runFrontendDiagnosticCase(t, frontend, tc, []string{"RTG_BACKEND=" + backend})
}

func runFrontendDiagnosticCase(t *testing.T, frontend frontendConfig, tc frontendDiagnosticCase, envOverride []string) {
	t.Helper()
	if frontend.compiler == "" {
		t.Fatal("frontend compiler is unavailable")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/diagnostic\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, source := range tc.files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	output := filepath.Join(dir, "app")
	cmd := exec.Command(frontend.compiler, "-t", frontend.target, "-s", "-o", output, "./cmd/app")
	cmd.Dir = dir
	env := append([]string(nil), frontend.env...)
	for _, override := range envOverride {
		env = setFrontendEnv(env, override)
	}
	cmd.Env = frontendCommandEnv(env, dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("frontend unexpectedly accepted %s", tc.name)
	}
	text := string(out)
	if !strings.Contains(text, "error "+tc.wantCode+" ") {
		t.Fatalf("diagnostic = %q, want stable code %s", text, tc.wantCode)
	}
	if !strings.Contains(text, tc.wantDetail) {
		t.Fatalf("diagnostic = %q, want detail %q", text, tc.wantDetail)
	}
	if tc.wantFile != "" {
		wantPath := filepath.Join(dir, filepath.FromSlash(tc.wantFile)) + ":"
		if !strings.Contains(text, wantPath) {
			t.Fatalf("diagnostic = %q, want source location beginning %q", text, wantPath)
		}
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("failed compilation left output %q (stat error %v)", output, statErr)
	}
}

func setFrontendEnv(env []string, item string) []string {
	key := envKey(item)
	for i := 0; i < len(env); i++ {
		if envKey(env[i]) == key {
			env[i] = item
			return env
		}
	}
	return append(env, item)
}
