package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/std/vm"
)

func TestVM32FrontendGenericsWideValues(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("executes Linux AMD64 output")
	}
	root := repoRoot(t)
	frontend := frontendCompiler(t, root)
	imagePath := filepath.Join(t.TempDir(), "frontend.rnvb")
	command := frontendCommand(frontend, "-t", "vm/vm32", "-arena-size", "201326592", "-s", "-o", imagePath, "./cmd/renvo")
	command.Dir = root
	command.Env = frontendCommandEnv(frontend.env, root)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile VM frontend: %v\n%s", err, output)
	}
	image, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	source := `package main
import "unsafe"
type Large[T any] [1<<32]T
func Check[T any]() {
 var array Large[struct{}]
 array[1<<32-1]=struct{}{}
 if len(array)!=1<<32 || unsafe.Sizeof(array)!=0 {panic("wide zero array")}
 if _,ok:=any(array).(Large[struct{}]);!ok {panic("wide array identity")}
 values:=make([]struct{},1<<32)
 values=append(([]struct{})(nil),values...)
 if len(values)!=1<<32 || cap(values)<len(values) {panic("wide slice")}
 var fn func(T)
 const n=unsafe.Sizeof(fn)
 var bytes[n]byte
 if _,ok:=any(bytes).([unsafe.Sizeof(fn)]byte);!ok {panic("callable bound")}
}
func main(){Check[int]();print("PASS\n")}
`
	files := vmFrontendSourceFiles(t, root)
	files = append(files,
		vm.File{Name: "/probe/go.mod", Data: []byte("module example.com/widevm\ngo 1.25\n"), Mode: 0644},
		vm.File{Name: "/probe/main.go", Data: []byte(source), Mode: 0644})
	result := vm.RunConfig(image, vm.Config{
		Limits: vm.Limits{Steps: 500000000, Memory: 256 * 1024 * 1024},
		Args:   []string{"renvo", "-t", "linux/amd64", "-s", "-o", "/probe/program", "."},
		Env:    []string{"PWD=/probe", "RENVO_STDROOT=/workspace/std"},
		Files:  files,
	})
	if result.Trap != vm.TrapNone || result.ExitCode != 0 {
		t.Fatalf("VM frontend: trap=%d exit=%d steps=%d stderr=%q output=%q", result.Trap, result.ExitCode, result.Steps, result.Stderr, result.Output)
	}
	var binary []byte
	for _, file := range result.Files {
		if file.Name == "/probe/program" {
			binary = file.Data
		}
	}
	outputPath := filepath.Join(t.TempDir(), "program")
	if err := os.WriteFile(outputPath, binary, 0755); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(outputPath).CombinedOutput(); err != nil || string(output) != "PASS\n" {
		t.Fatalf("execute 64-bit program from 32-bit frontend: %v output=%q", err, output)
	}
	t.Logf("32-bit frontend compiled wide arrays, slices and callable bounds in %d steps", result.Steps)
}
