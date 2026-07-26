package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"renvo.dev/std/vm"
)

func TestVMBytecodeExecutesCompiledProgram(t *testing.T) {
	source := []byte("package main\nfunc appMain() int { print(\"PASS\\n\"); return 7 }\n")
	image, ok := RenvoCompileSourceToBytesWithOptions(source, "vm/bytecode", RenvoCompileOptions{ArenaSize: 8192, StripSymbols: true})
	if !ok {
		t.Fatal("compile vm/bytecode")
	}
	result := vm.Run(image, vm.Limits{Steps: 100000, Memory: 32768})
	if result.Trap != vm.TrapNone {
		t.Fatalf("VM trapped: %d after %d steps with %d peak bytes", result.Trap, result.Steps, result.PeakMemory)
	}
	if result.ExitCode != 7 || string(result.Output) != "PASS\n" {
		t.Fatalf("VM result = exit %d, output %q", result.ExitCode, result.Output)
	}
}

func TestVMBytecodeSupportsArgumentsAndFileIO(t *testing.T) {
	tests := []struct {
		path string
		args []string
		env  []string
	}{
		{path: "tests/appmain_args_env.go", args: []string{"program"}, env: []string{"PATH=/vm"}},
		{path: "tests/read_write_read_after_write_before_close.go", args: []string{"program"}},
		{path: "tests/chmod_success_then_read_validates_content.go", args: []string{"program"}},
	}
	for _, test := range tests {
		source, err := os.ReadFile(test.path)
		if err != nil {
			t.Fatal(err)
		}
		image, ok := RenvoCompileSourceToBytesWithOptions(source, "vm/bytecode", RenvoCompileOptions{ArenaSize: 262144, StripSymbols: true})
		if !ok {
			t.Fatalf("compile %s", test.path)
		}
		result := vm.RunConfig(image, vm.Config{
			Limits: vm.Limits{Steps: 10000000, Memory: 2 * 1024 * 1024},
			Args:   test.args,
			Env:    test.env,
		})
		if result.Trap != vm.TrapNone || result.ExitCode != 0 || string(result.Output) != "PASS\n" {
			t.Fatalf("%s: exit %d, trap %d, stdout %q, stderr %q, steps %d",
				test.path, result.ExitCode, result.Trap, result.Output, result.Stderr, result.Steps)
		}
	}
}

func TestVMBytecodeEnforcesLimitsDeterministically(t *testing.T) {
	source := []byte("package main\nfunc appMain() int { for {} }\n")
	image, ok := RenvoCompileSourceToBytesWithOptions(source, "vm/bytecode", RenvoCompileOptions{ArenaSize: 8192, StripSymbols: true})
	if !ok {
		t.Fatal("compile vm/bytecode")
	}
	first := vm.Run(image, vm.Limits{Steps: 1000, Memory: 32768})
	second := vm.Run(image, vm.Limits{Steps: 1000, Memory: 32768})
	if first.Trap != vm.TrapStepLimit || first.Steps != 1000 {
		t.Fatalf("VM limit result = trap %d after %d steps", first.Trap, first.Steps)
	}
	if first.Steps != second.Steps || first.PeakMemory != second.PeakMemory || first.Trap != second.Trap {
		t.Fatalf("VM results differ: first=%+v second=%+v", first, second)
	}
}

func TestVMFullBackendSuite(t *testing.T) {
	paths, err := filepath.Glob("tests/*.go")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		t.Fatal("no backend regression programs")
	}
	for _, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		image, ok := RenvoCompileSourceToBytesWithOptions(source, "vm/bytecode", RenvoCompileOptions{
			ArenaSize:    8 * 1024 * 1024,
			StripSymbols: true,
		})
		if !ok {
			t.Fatalf("%s: compile failed", path)
		}
		result := vm.RunConfig(image, vm.Config{
			Limits: vm.Limits{Steps: 500 * 1000 * 1000, Memory: 16 * 1024 * 1024},
			Args:   []string{"program"},
			Env:    []string{"PATH=/vm"},
		})
		if result.Trap != vm.TrapNone || result.ExitCode != 0 ||
			string(result.Output)+string(result.Stderr) != "PASS\n" {
			t.Fatalf("%s: exit %d, trap %d at pc %d opcode %d, stdout %q, stderr %q, steps %d, peak %d, artifact %d",
				strings.TrimPrefix(path, "tests/"), result.ExitCode, result.Trap,
				result.TrapPC, result.TrapOpcode, result.Output, result.Stderr, result.Steps, result.PeakMemory, len(image))
		}
	}
}

func TestVMSelfHostedBackend(t *testing.T) {
	manifest, err := os.ReadFile("compiler_sources.txt")
	if err != nil {
		t.Fatal(err)
	}
	var compilerSource []byte
	for _, name := range strings.Fields(string(manifest)) {
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		compilerSource = append(compilerSource, source...)
		compilerSource = append(compilerSource, '\n')
	}
	compilerImage, ok := RenvoCompileSourceToBytesWithOptions(compilerSource, "vm/bytecode", RenvoCompileOptions{
		ArenaSize:    64 * 1024 * 1024,
		StripSymbols: true,
	})
	if !ok {
		t.Fatal("compile self-hosted VM backend")
	}
	input := []byte("package main\nfunc appMain() int { print(\"PASS\\n\"); return 0 }\n")
	compileResult := vm.RunConfig(compilerImage, vm.Config{
		Limits: vm.Limits{Steps: 2 * 1000 * 1000 * 1000, Memory: 96 * 1024 * 1024},
		Args: []string{
			"renvo-backend", "-t", "vm/bytecode", "-arena-size", "8192",
			"-s", "-o", "output.rnvb", "input.go",
		},
		Env:   []string{"PATH=/vm", "PWD=/"},
		Files: []vm.File{{Name: "input.go", Data: input}},
	})
	if compileResult.Trap != vm.TrapNone || compileResult.ExitCode != 0 {
		t.Fatalf("self-hosted VM backend: exit %d, trap %d at pc %d, stderr %q, steps %d, peak %d",
			compileResult.ExitCode, compileResult.Trap, compileResult.TrapPC,
			compileResult.Stderr, compileResult.Steps, compileResult.PeakMemory)
	}
	if len(compilerImage) > 1200*1024 ||
		compileResult.Steps > 400000 ||
		compileResult.PeakMemory > 80*1024*1024 {
		t.Fatalf("VM backend performance budget exceeded: artifact=%dB, execution=%d steps, peak=%dB",
			len(compilerImage), compileResult.Steps, compileResult.PeakMemory)
	}
	t.Logf("compiler artifact=%dB, execution=%d steps, peak=%dB", len(compilerImage), compileResult.Steps, compileResult.PeakMemory)
	var output []byte
	for _, file := range compileResult.Files {
		if file.Name == "output.rnvb" {
			output = file.Data
		}
	}
	if len(output) == 0 {
		t.Fatal("self-hosted VM backend did not produce output.rnvb")
	}
	result := vm.Run(output, vm.Limits{Steps: 100000, Memory: 32768})
	if result.Trap != vm.TrapNone || result.ExitCode != 0 || string(result.Output) != "PASS\n" {
		t.Fatalf("self-hosted VM output: exit %d, trap %d, stdout %q, stderr %q",
			result.ExitCode, result.Trap, result.Output, result.Stderr)
	}
}
