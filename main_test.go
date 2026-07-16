package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

type commandResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func resetRuntime() {
	for k := range files {
		if k >= 3 {
			files[k].Close()
		}
		delete(files, k)
	}

	files = make(map[int]file)
	files[0] = os.Stdin
	files[1] = os.Stdout
	files[2] = os.Stderr
}

type targetConfig struct {
	os   string
	arch string
}

const crossArchTestsEnv = "RTG_CROSS_ARCH_TESTS"

const frontendPerformanceCompilerEnv = "RTG_FRONTEND"
const frontendPerformanceSourceEnv = "RTG_FRONTEND_SOURCE"
const frontendPerformanceTargetEnv = "RTG_FRONTEND_TARGET"
const frontendPerformanceDefaultSource = "rtg/cmd/rtg"
const frontendPerformanceAttempts = 3
const frontendPerformanceCalibrationScale = 1000
const frontendPerformanceMaxCPUPerCalibration = 2 * frontendPerformanceCalibrationScale
const frontendPerformanceMaxRSSKB = 32 * 1024
const frontendPerformanceMaxBinarySize = 1024 * 1024

const frontendPerformanceCalibrationSource = `package main

func mix(data []byte, value int, round int) int {
	index := round & 4095
	current := int(data[index])
	if value&15 == 0 {
		value = value*33 + current + round
	} else {
		value = value*17 - current + round
	}
	value = value ^ value<<7
	value = value ^ value>>9
	data[index] = byte(value)
	return value
}

func main() {
	data := make([]byte, 4096)
	value := 216613626
	for i := 0; i < 23500000; i++ {
		value = mix(data, value, i)
	}
	print(value)
}
`

func getCompilerFiles(config targetConfig) ([]string, error) {
	switch config.os + "/" + config.arch {
	case "linux/amd64", "linux/386", "linux/aarch64", "linux/arm", "wasi/wasm32", "darwin/arm64":
	default:
		return nil, fmt.Errorf("unsupported OS/architecture combination: %s/%s", config.os, config.arch)
	}

	data, err := os.ReadFile("compiler_sources.txt")
	if err != nil {
		return nil, fmt.Errorf("read compiler source manifest: %w", err)
	}
	var files []string
	for _, line := range strings.Split(string(data), "\n") {
		path := strings.TrimSpace(line)
		if path != "" {
			files = append(files, path)
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("compiler source manifest is empty")
	}
	return files, nil
}

func getPerformanceCompilerFiles(t *testing.T, target compilerTarget, outDir string) []string {
	t.Helper()

	targetConst, tryFunc, backendFiles := performanceTargetEntry(t, target.name)
	wrapper := filepath.Join(outDir, "performance_main.go")
	content := fmt.Sprintf(`package main

var rtgCompilerDefaultTarget int = %[1]s
var rtgCompilerFixedTarget int = %[1]s
var rtgCompilerStripSymbols bool

func rtgOpenArg(path string, env []string) int {
	return open(path, O_RDONLY)
}

func rtgPrintErr(s string) {
	write(2, []byte(s), -1)
}

func rtgPrintIntErr(v int) {
	if v == 0 {
		rtgPrintErr("0")
		return
	}
	if v < 0 {
		rtgPrintErr("-")
		v = -v
	}
	var digits []byte
	for v > 0 {
		digits = append(digits, byte('0'+v%%10))
		v = v / 10
	}
	for i := len(digits) - 1; i >= 0; i-- {
		write(2, digits[i:i+1], -1)
	}
}

func rtgPrintUsage() {
	rtgPrintErr("usage: rtg [-s] -o <output|-> <input.go|->...\n")
}

func rtgPerformanceCompile(input []int, output int) int {
	rtgSetTarget(%[1]s)
	src := make([]byte, 0, 393216)
	for i := 0; i < len(input); i++ {
		src = rtgReadAll(input[i], src)
		src = append(src, '\n')
	}
	var prog rtgProgram
	prog = rtgParseProgram(src)
	if !prog.ok {
		rtgPrintErr("rtg: parse failed\n")
		return 1
	}
	var meta rtgMeta
	rtgBuildMetaInto(&prog, &meta)
	if !meta.ok {
		rtgPrintErr("rtg: meta failed\n")
		return 1
	}
	var result rtgCompileResult
	result = %[3]s(&prog, &meta)
	if result.ok {
		write(output, result.data, -1)
		return 0
	}
	rtgPrintErr("rtg: compilation failed\n")
	return 1
}

func appMain(args []string, env []string) int {
	var input []int
	var outputPath string
	i := 1
	for i < len(args) {
		arg := args[i]
		if arg == "-s" {
			rtgCompilerStripSymbols = true
			i++
			continue
		}
		if arg == "-o" {
			i++
			if i >= len(args) {
				rtgPrintErr("rtg: missing argument for -o\n")
				rtgPrintUsage()
				return 1
			}
			outputArg := args[i]
			outputPath = outputArg
			i++
			continue
		}
		if arg == "-" {
			input = append(input, 0)
			i++
			continue
		}
		if len(arg) > 0 {
			if arg[0] == '-' {
				rtgPrintErr("rtg: unknown option: ")
				rtgPrintErr(arg)
				rtgPrintErr("\n")
				rtgPrintUsage()
				return 1
			}
		}
		fd := rtgOpenArg(arg, env)
		if fd < 0 {
			rtgPrintErr("rtg: failed to open input: ")
			rtgPrintErr(arg)
			rtgPrintErr("\n")
			return 1
		}
		input = append(input, fd)
		i++
	}
	if outputPath == "" {
		rtgPrintErr("rtg: missing output path (-o)\n")
		rtgPrintUsage()
		return 1
	}
	if len(input) == 0 {
		rtgPrintErr("rtg: no input files\n")
		rtgPrintUsage()
		return 1
	}
	output := 1
	if outputPath != "-" {
		output = open(outputPath, O_RDWR|O_CREATE|O_TRUNC)
		if output < 0 {
			rtgPrintErr("rtg: failed to open output: ")
			rtgPrintErr(outputPath)
			rtgPrintErr("\n")
			return 1
		}
	}
	return rtgPerformanceCompile(input, output)
}
`, targetConst, target.name, tryFunc)
	if err := os.WriteFile(wrapper, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write performance compiler wrapper: %v", err)
	}
	linuxHelper := filepath.Join(outDir, "performance_linux_impl.go")
	linuxContent := getPerformanceLinuxImpl(t, target)
	if err := os.WriteFile(linuxHelper, []byte(linuxContent), 0o644); err != nil {
		t.Fatalf("failed to write performance Linux helper: %v", err)
	}
	amd64CommonHelper := filepath.Join(outDir, "performance_amd64_common_impl.go")

	files := []string{
		performanceAbsPath(t, "compiler_common_impl.go"),
		wrapper,
		linuxHelper,
	}
	for _, path := range backendFiles {
		if path == "@amd64-common" {
			if err := os.WriteFile(amd64CommonHelper, []byte(getPerformanceAmd64CommonImpl(t)), 0o644); err != nil {
				t.Fatalf("failed to write performance amd64 common helper: %v", err)
			}
			files = append(files, amd64CommonHelper)
			continue
		}
		files = append(files, performanceAbsPath(t, path))
	}
	return files
}

func performanceAbsPath(t *testing.T, path string) string {
	t.Helper()
	if filepath.IsAbs(path) {
		return path
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("failed to resolve %s: %v", path, err)
	}
	return abs
}

func getPerformanceLinuxImpl(t *testing.T, target compilerTarget) string {
	t.Helper()

	data, err := os.ReadFile("compiler_linux_impl.go")
	if err != nil {
		t.Fatalf("failed to read compiler_linux_impl.go: %v", err)
	}
	src := string(data)
	readAllEnd := strings.Index(src, "\nfunc compileLinuxTarget")
	sysStart := strings.Index(src, "func rtgLinuxSysWriteSeq")
	winStart := strings.Index(src, "func rtgWinAmd64CallImport")
	constStart := strings.Index(src, "func rtgEvalBuiltinConst")
	if readAllEnd < 0 || sysStart < 0 || winStart < 0 || constStart < 0 || sysStart >= winStart {
		t.Fatalf("failed to slice compiler_linux_impl.go for performance helper")
	}
	if target.name == "windows/amd64" || target.name == "windows/386" {
		helper := src[sysStart:constStart]
		if target.name == "windows/amd64" {
			helper = removePerformanceFuncsWithPrefix(helper, "rtgWin386")
		} else {
			helper = removePerformanceFuncsWithPrefix(helper, "rtgWinAmd64")
		}
		return src[:readAllEnd] + "\n\n" + helper + "\n\n" + src[constStart:]
	}
	if target.name == "wasi/wasm32" {
		return src[:readAllEnd] + `

const rtgLinuxAmd64SysReadSeq = 0
const rtgLinuxAmd64SysWriteSeq = 1
const rtgLinuxAmd64SysOpen = 2
const rtgLinuxAmd64SysClose = 3
const rtgLinuxAmd64SysReadAt = 17
const rtgLinuxAmd64SysWriteAt = 18
const rtgLinuxAmd64SysFchmod = 91

` + src[sysStart:winStart] + "\n\n" + src[constStart:]
	}
	return src[:readAllEnd] + "\n\n" + src[sysStart:winStart] + "\n\n" + src[constStart:]
}

func removePerformanceFuncsWithPrefix(src string, prefix string) string {
	for {
		start := strings.Index(src, "func "+prefix)
		if start < 0 {
			return src
		}
		next := strings.Index(src[start+1:], "\nfunc ")
		if next < 0 {
			return src[:start]
		}
		end := start + 1 + next + 1
		src = src[:start] + src[end:]
	}
}

func getPerformanceAmd64CommonImpl(t *testing.T) string {
	t.Helper()

	data, err := os.ReadFile("compiler_amd64_impl.go")
	if err != nil {
		t.Fatalf("failed to read compiler_amd64_impl.go: %v", err)
	}
	src := string(data)
	start := strings.Index(src, "func rtgAmd64EmitSwitchStringCaseTest")
	if start < 0 {
		t.Fatalf("failed to slice compiler_amd64_impl.go for performance helper")
	}
	helper := src[start:]
	removeNames := []string{
		"rtgAmd64EmitStringValueRegs",
		"rtgAmd64EmitCallWithWordCount",
		"rtgAmd64EmitFloatBinaryExpr",
		"rtgAmd64EnsureAppendAddrHelper",
		"rtgAmd64EnsureAppend8Helper",
		"rtgAmd64EnsureAppend64Helper",
		"rtgAmd64EnsureAppendBytesHelper",
		"rtgAmd64EnsureCopyWordsHelper",
		"rtgAmd64EnsureStringEqualHelper",
	}
	for _, name := range removeNames {
		next := removePerformanceFunc(helper, name)
		if next == helper {
			t.Fatalf("failed to remove %s for performance helper", name)
		}
		helper = next
	}
	return "package main\n\n" + helper
}

func removePerformanceFunc(src string, name string) string {
	start := strings.Index(src, "func "+name)
	if start < 0 {
		return src
	}
	next := strings.Index(src[start+1:], "\nfunc ")
	if next < 0 {
		return src[:start]
	}
	end := start + 1 + next + 1
	return src[:start] + src[end:]
}

func performanceTargetEntry(t *testing.T, targetName string) (string, string, []string) {
	t.Helper()

	switch targetName {
	case "linux/amd64":
		return "rtgTargetLinuxAmd64", "rtgTryCompileScalarProgramAmd64", []string{"compiler_amd64_impl.go", "compiler_linux_amd64_impl.go"}
	case "linux/386":
		return "rtgTargetLinux386", "rtgTryCompileScalarProgram386", []string{"compiler_386_impl.go", "compiler_linux_386_impl.go"}
	case "linux/aarch64":
		return "rtgTargetLinuxAarch64", "rtgTryCompileScalarProgramAarch64", []string{"@amd64-common", "compiler_aarch64_impl.go", "compiler_linux_aarch64_impl.go"}
	case "linux/arm":
		return "rtgTargetLinuxArm", "rtgTryCompileScalarProgramArm", []string{"@amd64-common", "compiler_arm_impl.go", "compiler_linux_arm_impl.go"}
	case "windows/amd64":
		return "rtgTargetWindowsAmd64", "rtgTryCompileScalarProgramAmd64", []string{"compiler_amd64_impl.go", "compiler_linux_amd64_impl.go"}
	case "windows/386":
		return "rtgTargetWindows386", "rtgTryCompileScalarProgram386", []string{"compiler_386_impl.go", "compiler_linux_386_impl.go"}
	case "wasi/wasm32":
		return "rtgTargetWasiWasm32", "rtgTryCompileScalarProgramWasm32", []string{"@amd64-common", "compiler_wasm32_impl.go", "compiler_linux_amd64_impl.go", "compiler_wasi_wasm32_impl.go"}
	default:
		t.Fatalf("unsupported performance target %s", targetName)
		return "", "", nil
	}
}

func performanceCompilerTargets(t *testing.T) []compilerTarget {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skipf("compiler performance gate requires Linux host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return []compilerTarget{
		{name: "linux/amd64"},
		{name: "linux/386"},
		{name: "linux/aarch64"},
		{name: "linux/arm"},
		{name: "windows/amd64"},
		{name: "windows/386"},
		// WASI has a separate performance follow-up: https://github.com/tinyrange/rtgx/issues/2
	}
}

type compilerTarget struct {
	name     string
	files    []string
	emulated bool
	runner   []string
}

func supportedCompilerTargets(t *testing.T) []compilerTarget {
	t.Helper()

	var targets []compilerTarget
	configs := []targetConfig{}

	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		configs = []targetConfig{
			{os: "linux", arch: "amd64"},
		}
		if crossArchTestsEnabled() {
			configs = append(configs,
				targetConfig{os: "linux", arch: "386"},
				targetConfig{os: "linux", arch: "aarch64"},
				targetConfig{os: "linux", arch: "arm"},
				targetConfig{os: "wasi", arch: "wasm32"},
			)
		}
	case "linux/arm64":
		configs = []targetConfig{
			{os: "linux", arch: "aarch64"},
		}
	case "darwin/arm64":
		configs = []targetConfig{
			{os: "darwin", arch: "arm64"},
		}
	default:
		t.Skipf("no RTG compiler targets supported on %s/%s", runtime.GOOS, runtime.GOARCH)
		return nil
	}
	for _, config := range configs {
		files, err := getCompilerFiles(config)
		if err != nil {
			t.Fatalf("failed to get compiler files for target %s/%s: %v", config.os, config.arch, err)
		}
		targetName := fmt.Sprintf("%s/%s", config.os, config.arch)
		target := compilerTarget{name: targetName, files: files}
		if runtime.GOARCH == "amd64" && config.arch == "aarch64" {
			target.emulated = true
			target.runner = []string{"qemu-aarch64"}
		}
		if runtime.GOARCH == "amd64" && config.arch == "arm" {
			target.emulated = true
			target.runner = []string{"qemu-arm"}
		}
		if config.os == "wasi" && config.arch == "wasm32" {
			target.emulated = true
			target.runner = []string{"wasmtime", "run", "--dir=.", "--dir=/", "--env", "PWD", "--env", "PATH"}
		}
		targets = append(targets, target)
	}
	return targets
}

func crossArchTestsEnabled() bool {
	return os.Getenv(crossArchTestsEnv) == "1"
}

func TestSupportedCompilerTargetsDefaultNativeOnly(t *testing.T) {
	if runtime.GOOS+"/"+runtime.GOARCH != "linux/amd64" {
		t.Skipf("target selection regression is linux/amd64-specific, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	t.Setenv(crossArchTestsEnv, "")

	targets := supportedCompilerTargets(t)
	if len(targets) != 1 {
		t.Fatalf("default targets = %#v, want one native target", targets)
	}
	if targets[0].name != "linux/amd64" {
		t.Fatalf("default target = %q, want linux/amd64", targets[0].name)
	}
}

func TestSupportedCompilerTargetsCrossArchOptIn(t *testing.T) {
	if runtime.GOOS+"/"+runtime.GOARCH != "linux/amd64" {
		t.Skipf("target selection regression is linux/amd64-specific, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	t.Setenv(crossArchTestsEnv, "1")

	targets := supportedCompilerTargets(t)
	var names []string
	for _, target := range targets {
		names = append(names, target.name)
	}
	want := []string{"linux/amd64", "linux/386", "linux/aarch64", "linux/arm", "wasi/wasm32"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("opt-in targets = %v, want %v", names, want)
	}
}

func (target compilerTarget) safeName() string {
	return strings.ReplaceAll(target.name, "/", "-")
}

func skipIfTargetRunnerMissing(t *testing.T, target compilerTarget) {
	t.Helper()
	if len(target.runner) == 0 {
		return
	}
	if _, err := exec.LookPath(target.runner[0]); err != nil {
		t.Skipf("runner %s is not installed", target.runner[0])
	}
}

func compile(inputFiles []string, outputFile string) error {
	resetRuntime()

	var input []int
	for _, path := range inputFiles {
		fd := open(path, O_RDONLY)
		if fd < 0 {
			return fmt.Errorf("failed to open input file: %s", path)
		}
		input = append(input, fd)
	}

	outputFd := open(outputFile, O_RDWR|O_CREATE|O_TRUNC)
	if outputFd < 0 {
		return fmt.Errorf("failed to open output file: %s", outputFile)
	}

	err := 1
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		err = compileDarwinArm64(input, outputFd)
	} else if runtime.GOOS == "linux" && runtime.GOARCH == "arm64" {
		err = compileLinuxAarch64(input, outputFd)
	} else {
		err = compileLinuxAmd64(input, outputFd)
	}
	if err != 0 {
		return fmt.Errorf("compilation failed")
	}
	if chmod(outputFd, 0755) != 0 {
		return fmt.Errorf("failed to set output file permissions")
	}
	close(outputFd)

	return nil
}

func runCommand(t *testing.T, path string, args ...string) (commandResult, error) {
	t.Helper()
	return runCommandInDir(t, t.TempDir(), path, args...)
}

func runCommandInDir(t *testing.T, dir string, path string, args ...string) (commandResult, error) {
	t.Helper()
	release := acquireTestProcess()
	defer release()

	var result commandResult
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(path, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result.stdout = stdout.String()
	result.stderr = stderr.String()
	if cmd.ProcessState != nil {
		result.exitCode = cmd.ProcessState.ExitCode()
		return result, nil
	}
	return result, err
}

func runCompilerBinary(t *testing.T, path string, outputFile string, inputFiles []string) error {
	t.Helper()
	args := append([]string{"-o", outputFile}, inputFiles...)
	result, err := runCommand(t, path, args...)
	if err != nil {
		return err
	}
	if result.exitCode != 0 {
		return fmt.Errorf("exit code %d\nstdout: %sstderr: %s", result.exitCode, result.stdout, result.stderr)
	}
	return nil
}

func runTargetCommand(t *testing.T, target compilerTarget, path string, args ...string) (commandResult, error) {
	t.Helper()
	if len(target.runner) > 0 {
		runArgs := append([]string{path}, args...)
		return runCommand(t, target.runner[0], append(target.runner[1:], runArgs...)...)
	}
	return runCommand(t, path, args...)
}

func runTargetCompilerBinary(t *testing.T, target compilerTarget, path string, outputFile string, inputFiles []string) error {
	t.Helper()
	args := append([]string{"-t", target.name, "-o", outputFile}, inputFiles...)
	result, err := runTargetCommand(t, target, path, args...)
	if err != nil {
		return err
	}
	if result.exitCode != 0 {
		return fmt.Errorf("exit code %d\nstdout: %sstderr: %s", result.exitCode, result.stdout, result.stderr)
	}
	return nil
}

func runHostCompilerBinaryForTarget(t *testing.T, target compilerTarget, path string, outputFile string, inputFiles []string) error {
	t.Helper()
	args := append([]string{"-t", target.name, "-o", outputFile}, inputFiles...)
	result, err := runCommand(t, path, args...)
	if err != nil {
		return err
	}
	if result.exitCode != 0 {
		return fmt.Errorf("exit code %d\nstdout: %sstderr: %s", result.exitCode, result.stdout, result.stderr)
	}
	return nil
}

func frontendPerformanceTarget(t *testing.T) string {
	t.Helper()
	if target := os.Getenv(frontendPerformanceTargetEnv); target != "" {
		if !strings.HasPrefix(target, "linux/") {
			t.Fatalf("%s=%s is not runnable by the frontend performance gate", frontendPerformanceTargetEnv, target)
		}
		return target
	}
	if runtime.GOOS != "linux" {
		t.Skipf("frontend performance gate requires Linux host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	switch runtime.GOARCH {
	case "amd64", "386", "arm":
		return "linux/" + runtime.GOARCH
	case "arm64":
		return "linux/aarch64"
	default:
		t.Skipf("unsupported frontend performance host architecture %s", runtime.GOARCH)
		return ""
	}
}

func frontendPerformanceSource(t *testing.T) string {
	t.Helper()
	if source := os.Getenv(frontendPerformanceSourceEnv); source != "" {
		abs, err := filepath.Abs(source)
		if err != nil {
			t.Fatalf("failed to resolve %s=%s: %v", frontendPerformanceSourceEnv, source, err)
		}
		return abs
	}
	if _, err := os.Stat(frontendPerformanceDefaultSource); err != nil {
		if os.IsNotExist(err) {
			t.Skipf("no frontend source found at %s; set %s to run the frontend performance gate",
				frontendPerformanceDefaultSource, frontendPerformanceSourceEnv)
		}
		t.Fatalf("failed to stat frontend source %s: %v", frontendPerformanceDefaultSource, err)
	}
	abs, err := filepath.Abs(frontendPerformanceDefaultSource)
	if err != nil {
		t.Fatalf("failed to resolve frontend source %s: %v", frontendPerformanceDefaultSource, err)
	}
	return abs
}

func frontendPerformanceStage0(t *testing.T, source string, outDir string) string {
	t.Helper()
	if compiler := os.Getenv(frontendPerformanceCompilerEnv); compiler != "" {
		return compiler
	}
	stage0 := filepath.Join(outDir, "rtg-stage0")
	result, err := runCommandInDir(t, ".", "go", "build", "-o", stage0, source)
	if err != nil {
		t.Fatalf("frontend stage0 host build failed: %v", err)
	}
	if result.exitCode != 0 {
		t.Fatalf("frontend stage0 host build failed with exit code %d\nstdout: %sstderr: %s",
			result.exitCode, result.stdout, result.stderr)
	}
	return stage0
}

func runFrontendCompile(t *testing.T, compiler string, target string, source string, output string) {
	t.Helper()
	result, err := runCommandInDir(t, ".", compiler, "-t", target, "-s", "-o", output, source)
	if err != nil {
		t.Fatalf("frontend compile failed: %v", err)
	}
	if result.exitCode != 0 {
		t.Fatalf("frontend compile failed with exit code %d\nstdout: %sstderr: %s",
			result.exitCode, result.stdout, result.stderr)
	}
}

func buildFrontendCPUCalibration(t *testing.T, compiler string, target string, outDir string) string {
	t.Helper()
	sourceDir, err := os.MkdirTemp(".", "frontend-cpu-calibration-")
	if err != nil {
		t.Fatalf("failed to create frontend CPU calibration source directory: %v", err)
	}
	defer os.RemoveAll(sourceDir)
	if err := os.WriteFile(filepath.Join(sourceDir, "main.go"), []byte(frontendPerformanceCalibrationSource), 0o644); err != nil {
		t.Fatalf("failed to write frontend CPU calibration source: %v", err)
	}
	output := filepath.Join(outDir, "frontend-cpu-calibration-bin")
	runFrontendCompile(t, compiler, target, sourceDir, output)
	return output
}

func runFrontendCPUCalibration(t *testing.T, executable string) time.Duration {
	t.Helper()
	cmd := exec.Command(executable)
	cmd.Env = []string{}
	err := cmd.Run()
	if err != nil {
		t.Fatalf("frontend CPU calibration failed: %v", err)
	}
	if cmd.ProcessState == nil {
		t.Fatal("frontend CPU calibration did not report process usage")
	}
	cpu := cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()
	if cpu <= 0 {
		t.Fatalf("frontend CPU calibration reported invalid CPU time %s", cpu)
	}
	return cpu
}

func runMeasuredFrontendCompile(t *testing.T, compiler string, target string, source string, output string, rssFile string) (time.Duration, int) {
	t.Helper()
	timeArgs := []string{
		"-f", "%U %S %M",
		"-o", rssFile,
		compiler,
		"-t", target,
		"-s",
		"-o", output,
		source,
	}
	cmd := exec.Command("/usr/bin/time", timeArgs...)
	cmd.Dir = "."
	cmd.Env = os.Environ()
	combined, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("resource-measured frontend self-host failed: %v\nOutput: %s", err, string(combined))
	}
	rssData, err := os.ReadFile(rssFile)
	if err != nil {
		t.Fatalf("failed to read frontend compile resource usage: %v", err)
	}
	rssFields := strings.Fields(string(rssData))
	if len(rssFields) != 3 {
		t.Fatalf("failed to read frontend compile resource usage %q", string(rssData))
	}
	userSeconds, err := strconv.ParseFloat(rssFields[0], 64)
	if err != nil {
		t.Fatalf("failed to parse frontend compile user CPU time %q: %v", string(rssData), err)
	}
	systemSeconds, err := strconv.ParseFloat(rssFields[1], 64)
	if err != nil {
		t.Fatalf("failed to parse frontend compile system CPU time %q: %v", string(rssData), err)
	}
	maxRSS, err := strconv.Atoi(rssFields[2])
	if err != nil {
		t.Fatalf("failed to parse frontend compile max RSS %q: %v", string(rssData), err)
	}
	return time.Duration((userSeconds + systemSeconds) * float64(time.Second)), maxRSS
}

func TestCompilerTargetDiagnostics(t *testing.T) {
	if runtime.GOOS+"/"+runtime.GOARCH != "linux/amd64" {
		t.Skipf("compiler target diagnostics require linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	files, err := getCompilerFiles(targetConfig{os: "linux", arch: "amd64"})
	if err != nil {
		t.Fatalf("failed to get compiler files: %v", err)
	}

	outDir := t.TempDir()
	stage0 := filepath.Join(outDir, "stage0")
	if err := compile(files, stage0); err != nil {
		t.Fatalf("stage0 compilation failed: %v", err)
	}

	checkFailure := func(name string, args []string, wants []string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			result, err := runCommand(t, stage0, args...)
			if err != nil {
				t.Fatalf("compiler execution failed: %v", err)
			}
			if result.exitCode == 0 {
				t.Fatalf("compiler accepted invalid arguments\nstdout: %sstderr: %s", result.stdout, result.stderr)
			}
			for _, want := range wants {
				if !strings.Contains(result.stderr, want) {
					t.Fatalf("diagnostic missing %q\nstdout: %sstderr: %s", want, result.stdout, result.stderr)
				}
			}
		})
	}

	outputFile := filepath.Join(outDir, "out")
	checkFailure(
		"unsupported target",
		[]string{"-t", "linux/arm64", "-o", outputFile, "tests/print_pass_smoke.go"},
		[]string{"rtg: unsupported target: linux/arm64", "linux/amd64", "linux/386", "linux/aarch64", "linux/arm", "windows/amd64", "windows/386", "wasi/wasm32", "darwin/arm64"},
	)
	checkFailure(
		"missing target argument",
		[]string{"-t"},
		[]string{"rtg: missing argument for -t", "usage: rtg"},
	)
	checkFailure(
		"missing arena size",
		[]string{"-arena-size"},
		[]string{"rtg: missing argument for -arena-size", "usage: rtg"},
	)
	checkFailure(
		"invalid arena size",
		[]string{"-arena-size", "128", "-o", outputFile, "tests/print_pass_smoke.go"},
		[]string{"rtg: invalid arena size: 128"},
	)
}

func TestStage1CompilerCanEmitSmokeTargets(t *testing.T) {
	for _, target := range supportedCompilerTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			skipIfTargetRunnerMissing(t, target)
			outDir := t.TempDir()
			if target.name == "linux/aarch64" || target.name == "linux/amd64" {
				var err error
				outDir, err = os.MkdirTemp("/tmp", "rtg-"+target.safeName()+"-stage1-")
				if err != nil {
					t.Fatalf("failed to create debug temp dir: %v", err)
				}
				t.Logf("preserving debug dir: %s", outDir)
			}
			stage0 := filepath.Join(outDir, "stage0")
			if err := compile(target.files, stage0); err != nil {
				t.Fatalf("stage0 compilation failed: %v", err)
			}

			stage1 := filepath.Join(outDir, "stage1")
			if err := runHostCompilerBinaryForTarget(t, target, stage0, stage1, target.files); err != nil {
				t.Fatalf("stage1 compilation failed: %v", err)
			}

			smoke := filepath.Join(outDir, "smoke")
			if err := runTargetCompilerBinary(t, target, stage1, smoke, []string{"tests/appmain_no_args.go"}); err != nil {
				t.Fatalf("stage1 smoke compilation failed: %v", err)
			}

			result, err := runTargetCommand(t, target, smoke)
			if err != nil {
				t.Fatalf("stage1 smoke execution failed: %v", err)
			}
			if result.exitCode != 0 || result.stdout != "PASS\n" || result.stderr != "" {
				t.Fatalf("stage1 smoke output mismatch: exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
			}
		})
	}
}

func buildStage2Compiler(t *testing.T, target compilerTarget, outDir string) string {
	t.Helper()
	key := testArtifactKeyForFiles(t, []string{"stage2", target.name}, target.files)
	return cachedTestArtifact(t, "stage2", key, func(stage2 string) error {
		stage0 := filepath.Join(outDir, "stage0-"+target.safeName())
		if err := compile(target.files, stage0); err != nil {
			return fmt.Errorf("stage0 compilation failed: %w", err)
		}
		stage1 := filepath.Join(outDir, "stage1-"+target.safeName())
		if err := runHostCompilerBinaryForTarget(t, target, stage0, stage1, target.files); err != nil {
			return fmt.Errorf("stage1 compilation failed: %w", err)
		}
		if err := runTargetCompilerBinary(t, target, stage1, stage2, target.files); err != nil {
			return fmt.Errorf("stage2 compilation failed: %w", err)
		}
		return nil
	})
}

func runWithHostGo(t *testing.T, path string) commandResult {
	t.Helper()
	runtimeData, err := os.ReadFile("rtg_main.go")
	if err != nil {
		t.Fatalf("failed to read runtime: %v", err)
	}
	testData, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read test source: %v", err)
	}
	key := testArtifactKey([]byte("host-go-result"), []byte(path), runtimeData, testData)
	return cachedCommandResult(t, "host-go-result", key, func() (commandResult, error) {
		outDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(outDir, "rtg_main.go"), runtimeData, 0644); err != nil {
			return commandResult{}, fmt.Errorf("write runtime copy: %w", err)
		}
		if err := os.WriteFile(filepath.Join(outDir, "test.go"), testData, 0644); err != nil {
			return commandResult{}, fmt.Errorf("write test copy: %w", err)
		}
		hostBinary := filepath.Join(outDir, "host-test")
		buildResult, err := runCommandInDir(t, outDir, "go", "build", "-o", hostBinary, "rtg_main.go", "test.go")
		if err != nil {
			return commandResult{}, fmt.Errorf("host go build failed: %w", err)
		}
		if buildResult.exitCode != 0 {
			return commandResult{}, fmt.Errorf("host go build failed with exit code %d\nstdout: %sstderr: %s", buildResult.exitCode, buildResult.stdout, buildResult.stderr)
		}
		result, err := runCommand(t, hostBinary)
		if err != nil {
			return commandResult{}, fmt.Errorf("host-built execution failed: %w", err)
		}
		return result, nil
	})
}

func compareCommandResult(t *testing.T, expected commandResult, actual commandResult) {
	t.Helper()
	if actual.stdout != expected.stdout || actual.stderr != expected.stderr || actual.exitCode != expected.exitCode {
		t.Fatalf("compiled output did not match host go\nstdout: got %q, want %q\nstderr: got %q, want %q\nexit code: got %d, want %d",
			actual.stdout, expected.stdout,
			actual.stderr, expected.stderr,
			actual.exitCode, expected.exitCode)
	}
}

// test that the compiler can compile and run a simple "hello, world!" program.
func TestCompileTests(t *testing.T) {
	targets := supportedCompilerTargets(t)

	// discover all files under tests/ that end with .go
	var inputFiles []string
	err := filepath.Walk("tests", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			inputFiles = append(inputFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to discover test files: %v", err)
	}

	for _, target := range targets {
		target := target
		t.Run(target.name, func(t *testing.T) {
			skipIfTargetRunnerMissing(t, target)

			outDir := t.TempDir()
			stage2 := buildStage2Compiler(t, target, outDir)

			for _, path := range inputFiles {
				path := path
				t.Run(path, func(t *testing.T) {
					t.Parallel()

					expected := runWithHostGo(t, path)

					outputFile := cachedTargetProgram(t, target, stage2, "source", []string{path})

					actual, err := runTargetCommand(t, target, outputFile)
					if err != nil {
						t.Fatalf("execution failed: %v", err)
					}
					compareCommandResult(t, expected, actual)
				})
			}
		})
	}
}

func TestConfiguredArenaExhaustionCannotEscapeBSS(t *testing.T) {
	targets := supportedCompilerTargets(t)
	if len(targets) == 0 {
		t.Fatal("no native compiler target")
	}
	target := targets[0]
	skipIfTargetRunnerMissing(t, target)
	outDir := t.TempDir()
	stage2 := buildStage2Compiler(t, target, outDir)
	outputFile := filepath.Join(outDir, "arena-exhaustion")
	result, err := runTargetCommand(t, target, stage2,
		"-t", target.name,
		"-arena-size", "256",
		"-o", outputFile,
		"tests/arena_bounded_allocation.go")
	if err != nil {
		t.Fatalf("compiler execution failed: %v", err)
	}
	if result.exitCode != 0 {
		t.Fatalf("compilation failed with exit code %d\nstdout: %sstderr: %s", result.exitCode, result.stdout, result.stderr)
	}
	actual, err := runTargetCommand(t, target, outputFile)
	if err != nil {
		t.Fatalf("bounded output execution failed: %v", err)
	}
	if actual.exitCode == 0 {
		t.Fatalf("arena exhaustion unexpectedly succeeded: stdout=%q stderr=%q", actual.stdout, actual.stderr)
	}
}

func TestRunTests(t *testing.T) {
	// discover all files under tests/ that end with .go
	var inputFiles []string
	err := filepath.Walk("tests", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			inputFiles = append(inputFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to discover test files: %v", err)
	}

	for _, path := range inputFiles {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			expected := runWithHostGo(t, path)

			if expected.exitCode != 0 {
				t.Fatalf("host go execution failed with exit code %d\nstdout: %sstderr: %s", expected.exitCode, expected.stdout, expected.stderr)
			}
		})
	}
}

// Check each single-backend Linux-host compiler cross-compiles its target in
// under 50ms, produces a binary under 256KB, and uses under 16MB max RSS.
func TestCompilerPerformance(t *testing.T) {
	for _, target := range performanceCompilerTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			outDir := t.TempDir()
			files := getPerformanceCompilerFiles(t, target, outDir)

			compilerPath := filepath.Join(outDir, "compiler")
			oldStrip := rtgCompilerStripSymbols
			rtgCompilerStripSymbols = true
			if err := compile(files, compilerPath); err != nil {
				rtgCompilerStripSymbols = oldStrip
				t.Fatalf("compiler build failed: %v", err)
			}
			rtgCompilerStripSymbols = oldStrip

			compilerInfo, err := os.Stat(compilerPath)
			if err != nil {
				t.Fatalf("failed to stat compiler binary: %v", err)
			}
			const maxRSSKB = 16 * 1024
			const maxBinarySize = 256 * 1024
			bestElapsed := 24 * time.Hour
			bestRSS := 1 << 30
			for attempt := 0; attempt < 3; attempt++ {
				outputPath := filepath.Join(outDir, fmt.Sprintf("compiler-output-%d", attempt))
				compileArgs := append([]string{"-s", "-o", outputPath}, files...)

				rssFile := filepath.Join(outDir, fmt.Sprintf("compile-rss-%d", attempt))
				timeArgs := append([]string{"-f", "%e %M", "-o", rssFile, compilerPath}, compileArgs...)
				cmd := exec.Command("/usr/bin/time", timeArgs...)
				cmd.Env = []string{}
				output, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("resource-measured compilation failed: %v\nOutput: %s", err, string(output))
				}

				rssData, err := os.ReadFile(rssFile)
				if err != nil {
					t.Fatalf("failed to read compile resource usage: %v", err)
				}
				rssLines := strings.Fields(string(rssData))
				if len(rssLines) == 0 {
					t.Fatalf("failed to read compile resource usage")
				}
				elapsedSeconds, err := strconv.ParseFloat(rssLines[0], 64)
				if err != nil {
					t.Fatalf("failed to parse compile elapsed time %q: %v", string(rssData), err)
				}
				elapsed := time.Duration(elapsedSeconds * float64(time.Second))
				maxRSS, err := strconv.Atoi(rssLines[len(rssLines)-1])
				if err != nil {
					t.Fatalf("failed to parse compile resource usage %q: %v", string(rssData), err)
				}
				if elapsed < bestElapsed {
					bestElapsed = elapsed
				}
				if maxRSS < bestRSS {
					bestRSS = maxRSS
				}
				if elapsed <= 50*time.Millisecond && maxRSS <= maxRSSKB && compilerInfo.Size() <= maxBinarySize {
					return
				}
			}

			var failures []string
			if bestElapsed > 50*time.Millisecond {
				failures = append(failures, fmt.Sprintf("runtime %s > 50ms", bestElapsed))
			}
			if bestRSS > maxRSSKB {
				failures = append(failures, fmt.Sprintf("compile max RSS %dKB > %dKB", bestRSS, maxRSSKB))
			}
			if compilerInfo.Size() > maxBinarySize {
				failures = append(failures, fmt.Sprintf("compiler binary size %dB > %dB", compilerInfo.Size(), maxBinarySize))
			}
			if len(failures) > 0 {
				t.Fatalf("performance limits failed: best runtime=%s, best compile max RSS=%dKB, compiler binary size=%dB; failures: %s",
					bestElapsed, bestRSS, compilerInfo.Size(), strings.Join(failures, "; "))
			}
		})
	}
}

// The replacement frontend must self-host quickly enough to stay usable as it
// grows. Stage0 builds stage1, stage1 builds stage2, and the measured run is
// stage2 building the stripped stage3 compiler. CPU time is normalized against
// a deterministic RTG-generated workload on the same runner so heterogeneous
// CI hosts do not turn a fixed wall-clock threshold into noise.
func TestFrontendCompilerPerformance(t *testing.T) {
	source := frontendPerformanceSource(t)
	target := frontendPerformanceTarget(t)
	outDir := t.TempDir()

	stage0 := frontendPerformanceStage0(t, source, outDir)
	stage1 := filepath.Join(outDir, "rtg-stage1")
	stage2 := filepath.Join(outDir, "rtg-stage2")
	runFrontendCompile(t, stage0, target, source, stage1)
	runFrontendCompile(t, stage1, target, source, stage2)
	calibration := buildFrontendCPUCalibration(t, stage2, target, outDir)

	bestCPUPerCalibration := int64(1 << 62)
	bestCPU := 24 * time.Hour
	bestCalibrationCPU := time.Duration(0)
	bestRSS := 1 << 30
	var bestSize int64 = 1 << 62
	for attempt := 0; attempt < frontendPerformanceAttempts; attempt++ {
		calibrationCPU := runFrontendCPUCalibration(t, calibration)
		stage3 := filepath.Join(outDir, fmt.Sprintf("rtg-stage3-%d", attempt))
		rssFile := filepath.Join(outDir, fmt.Sprintf("frontend-rss-%d", attempt))
		cpu, maxRSS := runMeasuredFrontendCompile(t, stage2, target, source, stage3, rssFile)
		cpuPerCalibration := int64(cpu) * frontendPerformanceCalibrationScale / int64(calibrationCPU)
		stage3Info, err := os.Stat(stage3)
		if err != nil {
			t.Fatalf("failed to stat frontend stage3 compiler: %v", err)
		}
		stage3Size := stage3Info.Size()
		t.Logf("frontend performance attempt %d: CPU=%s calibration=%s normalized=%d/%d RSS=%dKB size=%dB",
			attempt+1, cpu, calibrationCPU, cpuPerCalibration, frontendPerformanceCalibrationScale, maxRSS, stage3Size)
		if cpuPerCalibration < bestCPUPerCalibration {
			bestCPUPerCalibration = cpuPerCalibration
			bestCPU = cpu
			bestCalibrationCPU = calibrationCPU
		}
		if maxRSS < bestRSS {
			bestRSS = maxRSS
		}
		if stage3Size < bestSize {
			bestSize = stage3Size
		}
		if cpuPerCalibration <= frontendPerformanceMaxCPUPerCalibration &&
			maxRSS <= frontendPerformanceMaxRSSKB &&
			stage3Size <= frontendPerformanceMaxBinarySize {
			return
		}
	}

	var failures []string
	if bestCPUPerCalibration > frontendPerformanceMaxCPUPerCalibration {
		failures = append(failures, fmt.Sprintf("stage3 normalized CPU %d/%d calibration units > %d/%d", bestCPUPerCalibration, frontendPerformanceCalibrationScale, frontendPerformanceMaxCPUPerCalibration, frontendPerformanceCalibrationScale))
	}
	if bestRSS > frontendPerformanceMaxRSSKB {
		failures = append(failures, fmt.Sprintf("stage3 self-host max RSS %dKB > %dKB", bestRSS, frontendPerformanceMaxRSSKB))
	}
	if bestSize > frontendPerformanceMaxBinarySize {
		failures = append(failures, fmt.Sprintf("stage3 compiler binary size %dB > %dB", bestSize, frontendPerformanceMaxBinarySize))
	}
	if len(failures) > 0 {
		t.Fatalf("frontend performance limits failed: best stage3 CPU=%s, calibration CPU=%s, normalized CPU=%d/%d, best max RSS=%dKB, best stage3 compiler binary size=%dB; failures: %s",
			bestCPU, bestCalibrationCPU, bestCPUPerCalibration, frontendPerformanceCalibrationScale, bestRSS, bestSize, strings.Join(failures, "; "))
	}
}
