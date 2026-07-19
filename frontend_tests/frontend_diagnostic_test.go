package frontend_tests

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
			wantCode:   "RENVO-PARSE-001",
			wantFile:   "cmd/app/main.go",
			wantDetail: "source syntax is invalid",
		},
		{
			name:       "excluded_generics_declaration",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc identity[T any](value T) T { return value }\nfunc main() {}\n"},
			wantCode:   "RENVO-PARSE-002",
			wantFile:   "cmd/app/main.go",
			wantDetail: "generics are not supported by RENVO",
		},
		{
			name:       "excluded_generics_instantiation",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc identity[T any](value T) T { return value }\nfunc main() { _ = identity[int](1) }\n"},
			wantCode:   "RENVO-PARSE-002",
			wantFile:   "cmd/app/main.go",
			wantDetail: "generics are not supported by RENVO",
		},
		{
			name:       "unresolved_import",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nimport _ \"github.com/example/missing\"\n\nfunc main() {}\n"},
			wantCode:   "RENVO-LOAD-008",
			wantFile:   "cmd/app/main.go",
			wantDetail: "unresolved import github.com/example/missing",
		},
		{
			name:       "excluded_cgo",
			files:      map[string]string{"cmd/app/main.go": "package main\n\n/* #include <stdlib.h> */\nimport \"C\"\n\nfunc main() {}\n"},
			wantCode:   "RENVO-LOAD-019",
			wantFile:   "cmd/app/main.go",
			wantDetail: "cgo is not supported by RENVO",
		},
		{
			name:       "excluded_cgo_only_file",
			files:      map[string]string{"cmd/app/main.go": "//go:build cgo\n\npackage main\n\nfunc main() {}\n"},
			wantCode:   "RENVO-LOAD-019",
			wantFile:   "cmd/app/main.go",
			wantDetail: "cgo is not supported by RENVO",
		},
		{
			name:       "unavailable_standard_package",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nimport _ \"time\"\n\nfunc main() {}\n"},
			wantCode:   "RENVO-LOAD-020",
			wantFile:   "cmd/app/main.go",
			wantDetail: "standard library package time is not included in this RENVO build",
		},
		{
			name:       "embed_pattern",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nimport _ \"embed\"\n\n//go:embed missing.txt\nvar value string\n\nfunc main() {}\n"},
			wantCode:   "RENVO-LOAD-018",
			wantFile:   "cmd/app/main.go",
			wantDetail: "invalid go:embed directive or pattern: missing.txt",
		},
		{
			name: "import_cycle",
			files: map[string]string{
				"cmd/app/main.go": "package main\n\nimport _ \"example.com/diagnostic/lib\"\n\nfunc main() {}\n",
				"lib/lib.go":      "package lib\n\nimport _ \"example.com/diagnostic/cmd/app\"\n",
			},
			wantCode:   "RENVO-LOAD-011",
			wantFile:   "lib/lib.go",
			wantDetail: "import cycle detected",
		},
		{
			name:       "return_count",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc value() int { return }\nfunc main() { _ = value() }\n"},
			wantCode:   "RENVO-CHECK-007",
			wantFile:   "cmd/app/main.go",
			wantDetail: "return value count does not match function results",
		},
		{
			name:       "assignment_type",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { var value int; value = \"wrong\"; _ = value }\n"},
			wantCode:   "RENVO-CHECK-008",
			wantFile:   "cmd/app/main.go",
			wantDetail: "assignment value is not assignable to its destination",
		},
		{
			name:       "assignment_type_bool_from_int",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { var value bool; value = 1; _ = value }\n"},
			wantCode:   "RENVO-CHECK-008",
			wantFile:   "cmd/app/main.go",
			wantDetail: "assignment value is not assignable to its destination",
		},
		{
			name:       "undefined_identifier",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { print(missing) }\n"},
			wantCode:   "RENVO-CHECK-029",
			wantFile:   "cmd/app/main.go",
			wantDetail: "undefined identifier",
		},
		{
			name:       "undefined_type",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nvar value Missing\nfunc main() { _ = value }\n"},
			wantCode:   "RENVO-CHECK-029",
			wantFile:   "cmd/app/main.go",
			wantDetail: "undefined identifier",
		},
		{
			name:       "invalid_inferred_assignment",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { var value int; other := \"bad\"; value = other; _ = value }\n"},
			wantCode:   "RENVO-CHECK-008",
			wantFile:   "cmd/app/main.go",
			wantDetail: "assignment value is not assignable to its destination",
		},
		{
			name:       "invalid_operands",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { value := 1 + true; _ = value }\n"},
			wantCode:   "RENVO-CHECK-030",
			wantFile:   "cmd/app/main.go",
			wantDetail: "invalid operation for operand types",
		},
		{
			name:       "invalid_call_argument_type",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc consume(value int) {}\nfunc main() { consume(\"bad\") }\n"},
			wantCode:   "RENVO-CHECK-016",
			wantFile:   "cmd/app/main.go",
			wantDetail: "call argument is not assignable to its parameter",
		},
		{
			name:       "invalid_return_type",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc value() int { return \"bad\" }\nfunc main() { _ = value() }\n"},
			wantCode:   "RENVO-CHECK-031",
			wantFile:   "cmd/app/main.go",
			wantDetail: "return value is not assignable to the function result",
		},
		{
			name:       "unterminated_string",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { print(\"unterminated\n) }\n"},
			wantCode:   "RENVO-PARSE-001",
			wantFile:   "cmd/app/main.go",
			wantDetail: "source syntax is invalid",
		},
		{
			name:       "excluded_goroutine",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc work() {}\nfunc main() { go work() }\n"},
			wantCode:   "RENVO-CHECK-017",
			wantFile:   "cmd/app/main.go",
			wantDetail: "goroutines are not supported by RENVO",
		},
		{
			name:       "excluded_channel",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nvar values chan int\nfunc main() { _ = <-values }\n"},
			wantCode:   "RENVO-CHECK-018",
			wantFile:   "cmd/app/main.go",
			wantDetail: "channels are not supported by RENVO",
		},
		{
			name:       "excluded_select",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { select {} }\n"},
			wantCode:   "RENVO-CHECK-019",
			wantFile:   "cmd/app/main.go",
			wantDetail: "select statements are not supported by RENVO",
		},
		{
			name: "unused_import",
			files: map[string]string{
				"cmd/app/main.go": "package main\n\nimport \"example.com/diagnostic/lib\"\n\nfunc main() {}\n",
				"lib/lib.go":      "package lib\n",
			},
			wantCode:   "RENVO-CHECK-010",
			wantFile:   "cmd/app/main.go",
			wantDetail: "import is not used",
		},
		{
			name:       "non_function_call",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { x := 1; x() }\n"},
			wantCode:   "RENVO-CHECK-011",
			wantFile:   "cmd/app/main.go",
			wantDetail: "called expression is not a function",
		},
		{
			name:       "assignment_target",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { 1 = 2 }\n"},
			wantCode:   "RENVO-CHECK-012",
			wantFile:   "cmd/app/main.go",
			wantDetail: "left side of assignment is not assignable",
		},
		{
			name:       "assignment_count",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { a, b := 1; _, _ = a, b }\n"},
			wantCode:   "RENVO-CHECK-013",
			wantFile:   "cmd/app/main.go",
			wantDetail: "assignment count does not match",
		},
		{
			name:       "break_placement",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { break }\n"},
			wantCode:   "RENVO-CHECK-014",
			wantFile:   "cmd/app/main.go",
			wantDetail: "break is not inside a loop or switch",
		},
		{
			name:       "continue_placement",
			files:      map[string]string{"cmd/app/main.go": "package main\n\nfunc main() { continue }\n"},
			wantCode:   "RENVO-CHECK-015",
			wantFile:   "cmd/app/main.go",
			wantDetail: "continue is not inside a loop",
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
		wantCode:   "RENVO-BACKEND-003",
		wantDetail: "intentional backend failure",
	}
	runFrontendDiagnosticCase(t, frontend, tc, []string{"RENVO_BACKEND=" + backend})
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
