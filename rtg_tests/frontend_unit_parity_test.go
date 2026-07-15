package rtg_tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"j5.nz/rtg/rtgunit"
)

func TestFrontendCanonicalUnitCoreParity(t *testing.T) {
	if os.Getenv(selfHostTestsEnv) != "1" {
		t.Skipf("set %s=1 to compare host and self-hosted linked units", selfHostTestsEnv)
	}
	root := repoRoot(t)
	host := frontendCompiler(t, root)
	stage3 := selfHostedFrontendCompiler(t, root)
	cases := []string{
		"rtg_tests/quick/build_constraints/000_buildconstraints",
		"rtg_tests/regressions/imported_alias_method",
		"rtg_tests/regressions/std_bytes_split_api",
	}
	for _, relative := range cases {
		relative := relative
		t.Run(filepath.Base(relative), func(t *testing.T) {
			dir := filepath.Join(root, filepath.FromSlash(relative))
			hostUnit := emitFrontendUnit(t, host, dir, filepath.Join(t.TempDir(), "host.rtgu"))
			stage3Unit := emitFrontendUnit(t, stage3, dir, filepath.Join(t.TempDir(), "stage3.rtgu"))
			if !reflect.DeepEqual(coreUnit(hostUnit), coreUnit(stage3Unit)) {
				t.Fatalf("host and stage3 core linked units differ\nhost source:\n%s\nstage3 source:\n%s", rtgunit.Source(hostUnit), rtgunit.Source(stage3Unit))
			}
		})
	}
}

type canonicalUnitCore struct {
	Package    string
	ImportPath string
	Text       []byte
	Tokens     []byte
	Decls      []rtgunit.Decl
	Funcs      []rtgunit.Func
}

func coreUnit(program rtgunit.Program) canonicalUnitCore {
	return canonicalUnitCore{
		Package:    program.Package,
		ImportPath: program.ImportPath,
		Text:       program.Text,
		Tokens:     program.Tokens,
		Decls:      program.Decls,
		Funcs:      program.Funcs,
	}
}

func emitFrontendUnit(t *testing.T, frontend frontendConfig, dir string, output string) rtgunit.Program {
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
	if !bytes.HasPrefix(data, []byte("RTGU")) {
		t.Fatalf("linked unit has invalid header %x", data)
	}
	program, err := rtgunit.Unmarshal(data)
	if err != nil {
		t.Fatalf("decode linked unit: %v", err)
	}
	return program
}
