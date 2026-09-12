package rfe

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testSource(name string, deps ...Dependency) []byte {
	m := Manifest{Format: 1, Name: name, Import: "renvo.dev/rfe/" + name, Requires: deps}
	b, _ := json.Marshal(m)
	return append(append([]byte("RFE 1\n-- rfe.json --\n"), b...), '\n')
}
func TestDependencyResolution(t *testing.T) {
	cpu := testSource("cpu")
	parsed, err := Decode(cpu)
	if err != nil {
		t.Fatal(err)
	}
	root := testSource("machine", Dependency{Name: "cpu", SHA256: parsed.Digest})
	packages, err := Resolve(root, mapLoader(map[string][]byte{"cpu": cpu}))
	if err != nil || len(packages) != 2 || packages[0].Manifest.Name != "cpu" {
		t.Fatalf("resolution: %v", err)
	}
	for _, available := range []map[string][]byte{nil, {"cpu": append(cpu, '\n')}, {"cpu": testSource("different")}} {
		if _, err := Resolve(root, mapLoader(available)); err == nil {
			t.Fatal("accepted missing or substituted dependency")
		}
	}
	a := testSource("a", Dependency{Name: "b"})
	b := testSource("b", Dependency{Name: "a"})
	if _, err := Resolve(a, mapLoader(map[string][]byte{"a": a, "b": b})); err == nil {
		t.Fatal("accepted dependency cycle")
	}
}
func TestLoweringDiagnostics(t *testing.T) {
	header := "package demo\n//rfe:word 16\n//rfe:state 8\n"
	for _, body := range []string{
		"//rfe:instruction 0177770 0005200\nfunc inc(opcode uint64){state[8]=0}",
		"//rfe:instruction 0177770 0005200\nfunc inc(opcode uint64){state[opcode]=0}",
		"//rfe:instruction 0177770 0005200\nfunc inc(opcode uint64){state[0]=hostCall()}",
		"//rfe:instruction 0177770 0005200\nfunc inc(opcode uint64){for {}}",
		"//rfe:instruction 0177770 0005201\nfunc inc(opcode uint64){state[0]=0}",
		"//rfe:instruction 0177770 0005200\nfunc inc(opcode uint64){state[0]=0}\n//rfe:instruction 0177770 0005200\nfunc other(opcode uint64){state[1]=0}",
	} {
		if _, err := GenerateLowering([]byte(header + body)); err == nil {
			t.Fatalf("accepted invalid lowering: %s", body)
		}
	}
}

func TestEmulatorSourcePackages(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	available := map[string][]byte{}
	for _, name := range []string{"pdp11", "v7-user", "pdp11-machine"} {
		data, err := os.ReadFile(filepath.Join(root, "emulators", name+".rfe"))
		if err != nil {
			t.Fatal(err)
		}
		available[name] = data
	}
	source := testSource("test-suite", Dependency{Name: "v7-user"}, Dependency{Name: "pdp11-machine"})
	packages, err := Resolve(source, mapLoader(available))
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	if err = Workspace(workspace, root, packages); err != nil {
		t.Fatal(err)
	}
	// Prove dependency imports resolve to archive contents, not similarly named
	// source packages in the parent checkout.
	data, err := os.ReadFile(filepath.Join(workspace, "packages", "v7-user", "process.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("renvo.dev/rfe/workspace/packages/pdp11")) {
		t.Fatal("CPU import did not relocate")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "-count=1", "./packages/...")
	cmd.Dir = workspace
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=off", "GOSUMDB=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("RFE package tests: %v\n%s", err, out)
	}
}
func TestWorkspaceRejectsSourceCollision(t *testing.T) {
	s := string(testSource("demo")) + "-- instructions.lower --\npackage demo\n//rfe:word 8\n//rfe:state 1\n//rfe:instruction 255 0\nfunc nop(opcode uint64){state[0]=0}\n-- instructions_generated.go --\npackage demo\n"
	p, err := Decode([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	if err = Workspace(t.TempDir(), "/unused", []*Package{p}); err == nil || !strings.Contains(err.Error(), "unique") {
		t.Fatal("accepted generated source collision", err)
	}
}

func mapLoader(files map[string][]byte) func(string) ([]byte, error) {
	return func(name string) ([]byte, error) { return files[name], nil }
}
