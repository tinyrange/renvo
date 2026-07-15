package main

import (
	"debug/pe"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"j5.nz/rtg/rtgunit"
)

var windowsE2ETargets = []struct {
	name    string
	machine uint16
}{
	{name: "windows/amd64", machine: pe.IMAGE_FILE_MACHINE_AMD64},
	{name: "windows/386", machine: pe.IMAGE_FILE_MACHINE_I386},
}

func TestWindowsPEImages(t *testing.T) {
	outDir := t.TempDir()
	stage0 := buildWindowsHostCompiler(t, outDir)
	files := windowsCompilerSourceFiles(t)
	for _, target := range windowsE2ETargets {
		target := target
		t.Run(target.name, func(t *testing.T) {
			stage1 := buildWindowsStage1(t, stage0, target.name, files, t.TempDir())
			validateWindowsPE(t, stage1, target.machine)
		})
	}
}

func TestWindowsTargetsEndToEnd(t *testing.T) {
	windowsRunnerRequired(t)
	outDir := t.TempDir()
	stage0 := buildWindowsHostCompiler(t, outDir)
	files := windowsCompilerSourceFiles(t)
	testSources := []string{
		"tests/windows_args_env.go",
		"tests/print_pass_smoke.go",
		"tests/open_close_open_create_read_write_returns_usable_fd.go",
		"tests/read_write_read_after_write_before_close.go",
		"tests/chmod_success_then_read_validates_content.go",
		"tests/globals_multiple_globals_combine_into_checksum.go",
		"tests/functions_call_int_helper.go",
		"tests/control_integration_final_checksum.go",
	}
	amd64TestSources := []string{
		"tests/windows_amd64_linkstatic_even_stack_args.rtg",
	}

	for _, target := range windowsE2ETargets {
		target := target
		t.Run(target.name, func(t *testing.T) {
			targetDir := t.TempDir()
			targetTestSources := append([]string(nil), testSources...)
			if target.name == "windows/amd64" {
				targetTestSources = append(targetTestSources, amd64TestSources...)
			}
			t.Run("stage0_print_smoke", func(t *testing.T) {
				compileAndRunWindowsInput(t, target.name, stage0, []string{"tests/print_pass_smoke.go"})
			})
			t.Run("stage0_open_smoke", func(t *testing.T) {
				compileAndRunWindowsInput(t, target.name, stage0, []string{"tests/windows_open_create_close_smoke.go"})
			})
			stage1 := buildWindowsStage1(t, stage0, target.name, files, targetDir)
			stage2 := filepath.Join(targetDir, "rtg-stage2.exe")
			compileArgs := []string{"-t", target.name, "-s", "-o", stage2}
			compileArgs = append(compileArgs, files...)
			result, err := runWindowsCommand(t, ".", stage1, compileArgs...)
			if err != nil {
				t.Fatalf("stage1 execution failed: %v", err)
			}
			assertWindowsCommandOK(t, "stage1 compiler", result)
			validateWindowsPE(t, stage2, target.machine)

			for _, source := range targetTestSources {
				source := source
				t.Run(filepath.Base(source), func(t *testing.T) {
					compileAndRunWindowsInput(t, target.name, stage2, []string{source})
				})
			}

			t.Run("rtgu_frontend_smoke", func(t *testing.T) {
				unitDir := t.TempDir()
				unitPath := filepath.Join(unitDir, "smoke.rtgu")
				program, err := rtgunit.ConvertFiles([]string{"tests/print_pass_smoke.go"})
				if err != nil {
					t.Fatalf("frontend unit conversion failed: %v", err)
				}
				if err := rtgunit.WriteFile(unitPath, program); err != nil {
					t.Fatalf("frontend unit write failed: %v", err)
				}
				compileAndRunWindowsInput(t, target.name, stage2, []string{unitPath})
			})
		})
	}
}

func windowsCompilerSourceFiles(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile("compiler_sources.txt")
	if err != nil {
		t.Fatalf("read compiler source manifest: %v", err)
	}
	files := strings.Fields(string(data))
	if len(files) == 0 {
		t.Fatal("compiler source manifest is empty")
	}
	for _, path := range files {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("compiler source %s: %v", path, err)
		}
	}
	return files
}

func buildWindowsHostCompiler(t *testing.T, outDir string) string {
	t.Helper()
	name := "rtg-stage0"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	stage0 := filepath.Join(outDir, name)
	result, err := runCommandInDir(t, ".", "go", "build", "-o", stage0, ".")
	if err != nil {
		t.Fatalf("host compiler build failed: %v", err)
	}
	assertWindowsCommandOK(t, "host compiler build", result)
	return stage0
}

func buildWindowsStage1(t *testing.T, stage0 string, target string, files []string, outDir string) string {
	t.Helper()
	stage1 := filepath.Join(outDir, "rtg-stage1.exe")
	args := []string{"-t", target, "-s", "-o", stage1}
	args = append(args, files...)
	result, err := runCommandInDir(t, ".", stage0, args...)
	if err != nil {
		t.Fatalf("stage0 execution failed: %v", err)
	}
	assertWindowsCommandOK(t, "stage0 compiler", result)
	return stage1
}

func compileAndRunWindowsInput(t *testing.T, target string, compiler string, inputs []string) {
	t.Helper()
	output := filepath.Join(t.TempDir(), "program.exe")
	args := []string{"-t", target, "-s", "-o", output}
	args = append(args, inputs...)
	result, err := runWindowsCommand(t, ".", compiler, args...)
	if err != nil {
		t.Fatalf("Windows compiler execution failed: %v", err)
	}
	assertWindowsCommandOK(t, "Windows compiler", result)

	result, err = runWindowsCommand(t, t.TempDir(), output)
	if err != nil {
		t.Fatalf("Windows output execution failed: %v", err)
	}
	if result.exitCode != 0 || result.stdout != "PASS\n" || result.stderr != "" {
		t.Fatalf("Windows output mismatch: exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
	}
}

func runWindowsCommand(t *testing.T, dir string, path string, args ...string) (commandResult, error) {
	t.Helper()
	result, err := runCommandInDir(t, dir, path, args...)
	if runtime.GOOS != "windows" || result.exitCode == 0 || os.Getenv("RTG_WINDOWS_GDB") == "" {
		return result, err
	}
	gdbArgs := []string{
		"--batch",
		"-ex=run",
	}
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-t" && args[i+1] == "windows/386" {
			gdbArgs = append(gdbArgs, "-ex=continue")
			break
		}
	}
	gdbArgs = append(gdbArgs,
		"-ex=p/x$rax",
		"-ex=p/x$rsp",
		"-ex=p/x$rcx",
		"-ex=p/x$rdx",
		"-ex=p/x$r8",
		"-ex=p/x$r9",
		"-ex=p/x$rip",
		"-ex=p/x$eax",
		"-ex=p/x$esp",
		"-ex=p/x$esi",
		"-ex=p/x$eip",
		"-ex=x/8gx$rsp",
		"-ex=x/s$rcx",
		"-ex=x/s$esi",
		"-ex=bt",
		"-ex=x/16i$pc-32",
		"--args", path,
	)
	gdbArgs = append(gdbArgs, args...)
	diagnostic, diagnosticErr := runCommandInDir(t, dir, "gdb", gdbArgs...)
	t.Logf("native Windows crash diagnostics: err=%v exit=%d\nstdout:\n%s\nstderr:\n%s", diagnosticErr, diagnostic.exitCode, diagnostic.stdout, diagnostic.stderr)
	return result, err
}

func windowsRunnerRequired(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("Windows end-to-end execution requires a native Windows host")
	}
}

func validateWindowsPE(t *testing.T, path string, machine uint16) {
	t.Helper()
	image, err := pe.Open(path)
	if err != nil {
		t.Fatalf("open PE image: %v", err)
	}
	defer image.Close()
	if image.FileHeader.Machine != machine {
		t.Fatalf("PE machine = %#x, want %#x", image.FileHeader.Machine, machine)
	}
	symbols, err := image.ImportedSymbols()
	if err != nil {
		t.Fatalf("read PE imports: %v", err)
	}
	foundKernel32 := false
	for _, symbol := range symbols {
		if strings.HasSuffix(strings.ToLower(symbol), ":kernel32.dll") {
			foundKernel32 = true
		}
	}
	if !foundKernel32 {
		t.Fatalf("PE imports = %v, want kernel32.dll", symbols)
	}
}

func assertWindowsCommandOK(t *testing.T, operation string, result commandResult) {
	t.Helper()
	if result.exitCode != 0 {
		t.Fatalf("%s failed: exit=%d stdout=%q stderr=%q", operation, result.exitCode, result.stdout, result.stderr)
	}
}
