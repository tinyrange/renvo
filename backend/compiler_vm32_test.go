package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"renvo.dev/std/vm"
)

func TestVM32ExecutesCompiledProgram(t *testing.T) {
	source := []byte("package main\nfunc appMain() int { print(\"PASS\\n\"); return 7 }\n")
	image, ok := RenvoCompileSourceToBytesWithOptions(source, "vm/vm32", RenvoCompileOptions{ArenaSize: 8192, StripSymbols: true})
	if !ok {
		t.Fatal("compile vm/vm32")
	}
	result := vm.Run(image, vm.Limits{Steps: 100000, Memory: 32768})
	if result.Trap != vm.TrapNone {
		t.Fatalf("VM trapped: %d after %d steps with %d peak bytes", result.Trap, result.Steps, result.PeakMemory)
	}
	if result.ExitCode != 7 || string(result.Output) != "PASS\n" {
		t.Fatalf("VM result = exit %d, output %q", result.ExitCode, result.Output)
	}
}

func TestVM32FoldsAdjacentRegisterPushPop(t *testing.T) {
	var emitter renvoAsm
	emitter.lastPrimaryStoreEnd = -1
	renvoWasm32EmitReg(&emitter, renvoWasm32OpPushReg, renvoWasm32RegRax)
	renvoWasm32EmitReg(&emitter, renvoWasm32OpPopReg, renvoWasm32RegRax)
	if len(emitter.code) != 0 {
		t.Fatalf("same-register push/pop = %x, want empty", emitter.code)
	}
	renvoWasm32EmitReg(&emitter, renvoWasm32OpPushReg, renvoWasm32RegRax)
	renvoWasm32EmitReg(&emitter, renvoWasm32OpPopReg, renvoWasm32RegRdx)
	want := []byte{renvoWasm32OpMovRegReg, renvoWasm32RegRdx, renvoWasm32RegRax}
	if string(emitter.code) != string(want) {
		t.Fatalf("cross-register push/pop = %x, want %x", emitter.code, want)
	}
}

func TestVM32FoldsAdjacentImmediateBinary(t *testing.T) {
	var emitter renvoAsm
	renvoWasm32EmitRegImm(&emitter, renvoWasm32OpMovRegImm, renvoWasm32RegRdx, 0x12345678)
	renvoWasm32EmitRegReg(&emitter, renvoWasm32OpXorRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
	want := []byte{renvoWasm32OpBinaryRegImm, renvoWasm32RegRax, 0x78, 0x56, 0x34, 0x12, renvoWasm32OpXorRegReg}
	if string(emitter.code) != string(want) {
		t.Fatalf("immediate binary = %x, want %x", emitter.code, want)
	}
}

func TestVM32FoldsCommonStackPairs(t *testing.T) {
	tests := []struct {
		name string
		emit func(*renvoAsm)
		want []byte
	}{
		{
			name: "push-load-stack",
			emit: func(a *renvoAsm) {
				renvoWasm32EmitReg(a, renvoWasm32OpPushReg, renvoWasm32RegRax)
				renvoWasm32EmitStack(a, renvoWasm32OpLoadStack, renvoWasm32RegRdx, 12)
			},
			want: []byte{renvoWasm32OpPushRegLoadStack, renvoWasm32RegRax, renvoWasm32RegRdx, 12, 0, 0, 0},
		},
		{
			name: "load-stack-push",
			emit: func(a *renvoAsm) {
				renvoWasm32EmitStack(a, renvoWasm32OpLoadStack, renvoWasm32RegRdx, 12)
				renvoWasm32EmitReg(a, renvoWasm32OpPushReg, renvoWasm32RegRdx)
			},
			want: []byte{renvoWasm32OpLoadStackPushReg, renvoWasm32RegRdx, 12, 0, 0, 0, renvoWasm32RegRdx},
		},
		{
			name: "push-immediate",
			emit: func(a *renvoAsm) {
				renvoWasm32EmitReg(a, renvoWasm32OpPushReg, renvoWasm32RegRax)
				renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRdx, 9)
			},
			want: []byte{renvoWasm32OpPushRegMovImm, renvoWasm32RegRax, renvoWasm32RegRdx, 9, 0, 0, 0},
		},
		{
			name: "load-memory-push",
			emit: func(a *renvoAsm) {
				renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRax, renvoWasm32RegRdx, 4, 2)
				renvoWasm32EmitReg(a, renvoWasm32OpPushReg, renvoWasm32RegRax)
			},
			want: []byte{renvoWasm32OpLoadMemPushReg, renvoWasm32RegRax, renvoWasm32RegRdx, 4, 0, 0, 0, 2, renvoWasm32RegRax},
		},
		{
			name: "load-stack-pop",
			emit: func(a *renvoAsm) {
				renvoWasm32EmitStack(a, renvoWasm32OpLoadStack, renvoWasm32RegRdx, 12)
				renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRcx)
			},
			want: []byte{renvoWasm32OpLoadStackPop, renvoWasm32RegRdx, 12, 0, 0, 0, renvoWasm32RegRcx},
		},
		{
			name: "immediate-pop",
			emit: func(a *renvoAsm) {
				renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRdx, 9)
				renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRcx)
			},
			want: []byte{renvoWasm32OpMovRegImmPop, renvoWasm32RegRdx, 9, 0, 0, 0, renvoWasm32RegRcx},
		},
		{
			name: "register-pop",
			emit: func(a *renvoAsm) {
				renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRdx, renvoWasm32RegRax)
				renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRcx)
			},
			want: []byte{renvoWasm32OpMovRegRegPop, renvoWasm32RegRdx, renvoWasm32RegRax, renvoWasm32RegRcx, 0},
		},
		{
			name: "stack-load-tail-resembles-register-move",
			emit: func(a *renvoAsm) {
				renvoWasm32EmitStack(a, renvoWasm32OpLoadStack, renvoWasm32RegRax, 0x400)
				renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRcx)
			},
			want: []byte{renvoWasm32OpLoadStackPop, renvoWasm32RegRax, 0, 4, 0, 0, renvoWasm32RegRcx},
		},
		{
			name: "memory-load-tail-resembles-register-move",
			emit: func(a *renvoAsm) {
				renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRax, renvoWasm32RegRdx, 0x40000, 0)
				renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRcx)
			},
			want: []byte{
				renvoWasm32OpLoadMem, renvoWasm32RegRax, renvoWasm32RegRdx, 0, 0, 4, 0, 0,
				renvoWasm32OpPopReg, renvoWasm32RegRcx,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var emitter renvoAsm
			test.emit(&emitter)
			if string(emitter.code) != string(test.want) {
				t.Fatalf("encoding = %x, want %x", emitter.code, test.want)
			}
		})
	}
}

func TestWasm32DirectEmissionKeepsVMFusionsDisabled(t *testing.T) {
	tests := []struct {
		name string
		emit func(*renvoAsm)
		want []byte
	}{
		{
			name: "immediate-binary",
			emit: func(a *renvoAsm) {
				renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRdx, 9)
				renvoWasm32EmitRegReg(a, renvoWasm32OpXorRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
			},
			want: []byte{renvoWasm32OpMovRegImm, renvoWasm32RegRdx, 9, 0, 0, 0,
				renvoWasm32OpXorRegReg, renvoWasm32RegRax, renvoWasm32RegRdx},
		},
		{
			name: "push-load-stack",
			emit: func(a *renvoAsm) {
				renvoWasm32EmitReg(a, renvoWasm32OpPushReg, renvoWasm32RegRax)
				renvoWasm32EmitStack(a, renvoWasm32OpLoadStack, renvoWasm32RegRdx, 12)
			},
			want: []byte{renvoWasm32OpPushReg, renvoWasm32RegRax,
				renvoWasm32OpLoadStack, renvoWasm32RegRdx, 12, 0, 0, 0},
		},
		{
			name: "load-stack-push",
			emit: func(a *renvoAsm) {
				renvoWasm32EmitStack(a, renvoWasm32OpLoadStack, renvoWasm32RegRdx, 12)
				renvoWasm32EmitReg(a, renvoWasm32OpPushReg, renvoWasm32RegRdx)
			},
			want: []byte{renvoWasm32OpLoadStack, renvoWasm32RegRdx, 12, 0, 0, 0,
				renvoWasm32OpPushReg, renvoWasm32RegRdx},
		},
		{
			name: "push-immediate",
			emit: func(a *renvoAsm) {
				renvoWasm32EmitReg(a, renvoWasm32OpPushReg, renvoWasm32RegRax)
				renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRdx, 9)
			},
			want: []byte{renvoWasm32OpPushReg, renvoWasm32RegRax,
				renvoWasm32OpMovRegImm, renvoWasm32RegRdx, 9, 0, 0, 0},
		},
		{
			name: "load-memory-push",
			emit: func(a *renvoAsm) {
				renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRax, renvoWasm32RegRdx, 4, 2)
				renvoWasm32EmitReg(a, renvoWasm32OpPushReg, renvoWasm32RegRax)
			},
			want: []byte{renvoWasm32OpLoadMem, renvoWasm32RegRax, renvoWasm32RegRdx, 4, 0, 0, 0, 2,
				renvoWasm32OpPushReg, renvoWasm32RegRax},
		},
		{
			name: "load-stack-pop",
			emit: func(a *renvoAsm) {
				renvoWasm32EmitStack(a, renvoWasm32OpLoadStack, renvoWasm32RegRdx, 12)
				renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRcx)
			},
			want: []byte{renvoWasm32OpLoadStack, renvoWasm32RegRdx, 12, 0, 0, 0,
				renvoWasm32OpPopReg, renvoWasm32RegRcx},
		},
		{
			name: "immediate-pop",
			emit: func(a *renvoAsm) {
				renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRdx, 9)
				renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRcx)
			},
			want: []byte{renvoWasm32OpMovRegImm, renvoWasm32RegRdx, 9, 0, 0, 0,
				renvoWasm32OpPopReg, renvoWasm32RegRcx},
		},
		{
			name: "register-pop",
			emit: func(a *renvoAsm) {
				renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRdx, renvoWasm32RegRax)
				renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRcx)
			},
			want: []byte{renvoWasm32OpMovRegReg, renvoWasm32RegRdx, renvoWasm32RegRax,
				renvoWasm32OpPopReg, renvoWasm32RegRcx},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			context := renvoNewCompileContext(renvoTargetWasiWasm32, false, false, false)
			emitter := renvoAsm{c: context}
			test.emit(&emitter)
			if string(emitter.code) != string(test.want) {
				t.Fatalf("encoding = %x, want unfused %x", emitter.code, test.want)
			}
		})
	}
}

func TestVM32SupportsArgumentsAndFileIO(t *testing.T) {
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
		image, ok := RenvoCompileSourceToBytesWithOptions(source, "vm/vm32", RenvoCompileOptions{ArenaSize: 262144, StripSymbols: true})
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

func TestVM32EnforcesLimitsDeterministically(t *testing.T) {
	source := []byte("package main\nfunc appMain() int { for {} }\n")
	image, ok := RenvoCompileSourceToBytesWithOptions(source, "vm/vm32", RenvoCompileOptions{ArenaSize: 8192, StripSymbols: true})
	if !ok {
		t.Fatal("compile vm/vm32")
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

func TestVM32FullBackendSuite(t *testing.T) {
	resetRuntime()
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
		image, ok := RenvoCompileSourceToBytesWithOptions(source, "vm/vm32", RenvoCompileOptions{
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

func TestVM32SelfHostedBackend(t *testing.T) {
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
	compilerImage, ok := RenvoCompileSourceToBytesWithOptions(compilerSource, "vm/vm32", RenvoCompileOptions{
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
			"renvo-backend", "-t", "vm/vm32", "-arena-size", "8192",
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
	if len(compilerImage) > 2*1024*1024 ||
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
