package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"renvo.dev/std/vm"
)

func TestVM32ReadDir(t *testing.T) {
	root := repoRoot(t)
	frontend := frontendCompiler(t, root)
	if frontend.compiler == "" {
		t.Fatal("frontend compiler unavailable")
	}
	project := t.TempDir()
	source := []byte(`package main

import "os"

func main() {
	entries, err := os.ReadDir("/workspace")
	if err != nil || len(entries) != 3 ||
		entries[0].Name() != "a.go" || entries[0].IsDir() ||
		entries[1].Name() != "cmd" || !entries[1].IsDir() ||
		entries[2].Name() != "z.go" || entries[2].IsDir() {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
`)
	input := filepath.Join(project, "input.go")
	if err := os.WriteFile(input, source, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.com/readdir\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(project, "readdir.rnvb")
	cmd := exec.Command(frontend.compiler, "-t", "vm/vm32", "-arena-size", "262144", "-s", "-o", imagePath, "input.go")
	cmd.Dir = project
	cmd.Env = frontendCommandEnv(frontend.env, project)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile VM ReadDir fixture: %v\n%s", err, output)
	}
	image, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	result := vm.RunConfig(image, vm.Config{
		Limits: vm.Limits{Steps: 10 * 1000 * 1000, Memory: 2 * 1024 * 1024},
		Files: []vm.File{
			{Name: "/workspace/z.go"},
			{Name: "/workspace/cmd/app/main.go"},
			{Name: "/workspace/a.go"},
		},
	})
	if result.Trap != vm.TrapNone || result.ExitCode != 0 || string(result.Output) != "PASS\n" {
		t.Fatalf("VM ReadDir: exit %d, trap %d at pc %d, stdout %q, stderr %q",
			result.ExitCode, result.Trap, result.TrapPC, result.Output, result.Stderr)
	}
}

func TestVM32FrontendPerformanceGate(t *testing.T) {
	root := repoRoot(t)
	frontend := frontendCompiler(t, root)
	if frontend.compiler == "" {
		t.Fatal("frontend compiler unavailable")
	}
	imagePath := filepath.Join(t.TempDir(), "renvo-frontend.rnvb")
	cmd := exec.Command(frontend.compiler,
		"-tags", "renvo_bundle",
		"-t", "vm/vm32",
		"-arena-size", "134217728",
		"-s", "-o", imagePath,
		"./cmd/renvo",
	)
	cmd.Dir = root
	cmd.Env = frontendCommandEnv(frontend.env, root)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile VM frontend: %v\n%s", err, output)
	}
	image, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	files := vmFrontendSourceFiles(t, root)
	compileResult := vm.RunConfig(image, vm.Config{
		Limits: vm.Limits{Steps: 9 * 1000 * 1000 * 1000, Memory: 192 * 1024 * 1024},
		Args: []string{
			"renvo", "-tags", "renvo_bundle",
			"-system", "/workspace/systems/frontend-linux-amd64.rtg",
			"-s", "-o", "/workspace/renvo-linux-amd64", "./cmd/renvo",
		},
		Env:   []string{"PATH=/vm", "PWD=/workspace"},
		Files: files,
	})
	if compileResult.Trap != vm.TrapNone || compileResult.ExitCode != 0 {
		t.Fatalf("VM frontend: exit %d, trap %d at pc %d, stdout %q, stderr %q, steps %d, peak %d, files %#v",
			compileResult.ExitCode, compileResult.Trap, compileResult.TrapPC,
			compileResult.Output, compileResult.Stderr, compileResult.Steps, compileResult.PeakMemory, compileResult.Files)
	}
	var output []byte
	for _, file := range compileResult.Files {
		if file.Name == "/workspace/renvo-linux-amd64" {
			output = file.Data
		}
	}
	if len(output) < 4 || output[0] != 0x7f ||
		output[1] != 'E' || output[2] != 'L' || output[3] != 'F' {
		t.Fatalf("VM frontend Linux output prefix = % x", output[:minBundleLength(len(output), 4)])
	}
	if len(image) > 4*1024*1024 ||
		len(output) > 2*1024*1024 ||
		compileResult.Steps > 8*1000*1000*1000 ||
		compileResult.PeakMemory > 150*1024*1024 {
		t.Fatalf("VM frontend performance budget exceeded: artifact=%dB, output=%dB, execution=%d steps, peak=%dB",
			len(image), len(output), compileResult.Steps, compileResult.PeakMemory)
	}
	t.Logf("frontend artifact=%dB, Linux output=%dB, execution=%d steps, peak=%dB",
		len(image), len(output), compileResult.Steps, compileResult.PeakMemory)
}

func vmFrontendSourceFiles(t *testing.T, root string) []vm.File {
	t.Helper()
	var files []vm.File
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if relative == ".git" || relative == "sandbox" ||
				strings.HasPrefix(relative, ".git"+string(filepath.Separator)) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		extension := filepath.Ext(path)
		if extension != ".go" && extension != ".rtg" && filepath.Base(path) != "go.mod" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, vm.File{
			Name: filepath.ToSlash(filepath.Join("/workspace", relative)),
			Data: data,
			Mode: 0o644,
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
