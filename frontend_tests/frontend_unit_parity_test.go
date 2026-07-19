package frontend_tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"renvo.dev/backend/unit"
)

func TestFrontendCanonicalUnitCoreParity(t *testing.T) {
	if os.Getenv(selfHostTestsEnv) != "1" {
		t.Skipf("set %s=1 to compare host and self-hosted linked units", selfHostTestsEnv)
	}
	root := repoRoot(t)
	host := frontendCompiler(t, root)
	stage3 := selfHostedFrontendCompiler(t, root)
	cases := []string{
		"frontend_tests/quick/build_constraints/000_buildconstraints",
		"frontend_tests/regressions/imported_alias_method",
		"frontend_tests/regressions/interface_dynamic_dispatch",
		"frontend_tests/regressions/type_assertion_switch_semantics",
		"frontend_tests/regressions/accepted_language_semantics",
		"frontend_tests/regressions/semantic_map_linking",
		"frontend_tests/regressions/std_path_api",
		"frontend_tests/extended/closures/000_closures",
		"frontend_tests/extended/function_values/000_functionvalues",
		"frontend_tests/extended/defer_panic_recover/000_deferpanicrecover",
		"frontend_tests/extended/multi_package/000_multipackage",
		"frontend_tests/extended/package_init/000_packageinit",
	}
	for _, relative := range cases {
		relative := relative
		t.Run(filepath.Base(relative), func(t *testing.T) {
			dir := filepath.Join(root, filepath.FromSlash(relative))
			hostUnit := emitFrontendUnit(t, host, dir, filepath.Join(t.TempDir(), "host.unit"))
			stage3Unit := emitFrontendUnit(t, stage3, dir, filepath.Join(t.TempDir(), "stage3.unit"))
			if !bytes.Equal(hostUnit.data, stage3Unit.data) {
				t.Fatalf("host and stage3 canonical linked units differ\nhost source:\n%s\nstage3 source:\n%s", unit.Source(hostUnit.program), unit.Source(stage3Unit.program))
			}
		})
	}
}

type emittedFrontendUnit struct {
	data    []byte
	program unit.Program
}

func emitFrontendUnit(t *testing.T, frontend frontendConfig, dir string, output string) emittedFrontendUnit {
	t.Helper()
	cmd := exec.Command(frontend.compiler, "-emit-unit", "-t", frontend.target, "-o", output, "./cmd/app")
	cmd.Dir = dir
	cmd.Env = frontendCommandEnv(frontend.env, dir)
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("emit linked unit: %v\n%s", err, combined)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("RNVO")) {
		t.Fatalf("linked unit has invalid header %x", data)
	}
	program, err := unit.Unmarshal(data)
	if err != nil {
		t.Fatalf("decode linked unit: %v", err)
	}
	return emittedFrontendUnit{data: data, program: program}
}
