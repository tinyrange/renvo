//go:build !renvo

package testfront

import (
	"os/exec"
	"testing"
)

func TestGeneratePackagePreservesApplicationMain(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "main.go", `package main
var calls int
var __renvo_application_main = 7
func main() { calls++ }
type field struct { main int }
type method struct{}
func (method) main() int { return 9 }
`)
	writeTestFile(t, dir, "main_test.go", `package main
import "testing"
func TestMainBinding(t *testing.T) {
 if calls != 0 { t.Fatal("application ran before tests") }
 f := main; f(); main()
 if calls != 2 || __renvo_application_main != 7 { t.Fatal("application binding") }
 value := field{main: 3}
 if value.main != 3 || (method{}).main() != 9 { t.Fatal("member binding") }
 main := func() int { return 11 }
 if main() != 11 { t.Fatal("shadowing local") }
}
`)
	result, err := GeneratePackage(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := WritePackage(out, result); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, out, "go.mod", "module generated.test\n")
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = out
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated main-package tests: %v\n%s", err, output)
	}
}
