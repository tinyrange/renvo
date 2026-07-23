package frontend_tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"renvo.dev/internal/linkedimage"
)

const frontendBundleTestsEnv = "RENVO_FRONTEND_BUNDLE_TESTS"

func TestBundledFrontendStandaloneAllTargets(t *testing.T) {
	if os.Getenv(frontendBundleTestsEnv) != "1" {
		t.Skipf("set %s=1 to run bundled frontend tests", frontendBundleTestsEnv)
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("bundled frontend execution requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	root := repoRoot(t)
	toolDir := t.TempDir()
	backend := filepath.Join(toolDir, "renvo-backend")
	buildGoTool(t, root, backend, nil, "./backend")
	stage0 := filepath.Join(toolDir, "renvo-stage0")
	buildGoTool(t, root, stage0, []string{"renvo_bundle"}, "./cmd/renvo")
	stage1 := filepath.Join(toolDir, "renvo-stage1")
	cmd := exec.Command(stage0, "-tags", "renvo_bundle", "-t", "linux/amd64", "-s", "-o", stage1, "./cmd/renvo")
	cmd.Dir = root
	cmd.Env = []string{"PWD=" + root, "RENVO_BACKEND=" + backend}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bundled frontend self-host build failed: %v\n%s", err, string(out))
	}
	targets := []struct {
		name     string
		artifact string
		prefix   []byte
	}{
		{name: "linux/amd64", artifact: "renvo-linux-amd64", prefix: []byte{0x7f, 'E', 'L', 'F'}},
		{name: "linux/386", artifact: "renvo-linux-386", prefix: []byte{0x7f, 'E', 'L', 'F'}},
		{name: "linux/aarch64", artifact: "renvo-linux-arm64", prefix: []byte{0x7f, 'E', 'L', 'F'}},
		{name: "linux/arm", artifact: "renvo-linux-arm", prefix: []byte{0x7f, 'E', 'L', 'F'}},
		{name: "windows/amd64", artifact: "renvo-windows-amd64.exe", prefix: []byte{'M', 'Z'}},
		{name: "windows/386", artifact: "renvo-windows-386.exe", prefix: []byte{'M', 'Z'}},
		{name: "windows/arm64", artifact: "renvo-windows-arm64.exe", prefix: []byte{'M', 'Z'}},
		{name: "darwin/arm64", artifact: "renvo-darwin-arm64", prefix: []byte{0xcf, 0xfa, 0xed, 0xfe}},
		{name: "wasi/wasm32", artifact: "renvo-wasi-wasm32.wasm", prefix: []byte{0, 'a', 's', 'm'}},
	}
	var releaseFrontend string
	for _, target := range targets {
		artifact := filepath.Join(toolDir, target.artifact)
		cmd = exec.Command(stage1, "-tags", "renvo_bundle", "-t", target.name, "-s", "-o", artifact, "./cmd/renvo")
		cmd.Dir = root
		cmd.Env = []string{"PWD=" + root}
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("release frontend build for %s failed: %v\n%s", target.name, err, string(out))
		}
		data, err := os.ReadFile(artifact)
		if err != nil {
			t.Fatalf("read %s frontend failed: %v", target.name, err)
		}
		if !bytes.HasPrefix(data, target.prefix) {
			t.Fatalf("%s frontend prefix = % x, want % x", target.name, data[:minBundleLength(len(data), len(target.prefix))], target.prefix)
		}
		if target.name == "linux/amd64" {
			releaseFrontend = artifact
		}
	}

	standaloneDir := t.TempDir()
	standalone := filepath.Join(standaloneDir, "renvo")
	if err := os.Rename(releaseFrontend, standalone); err != nil {
		t.Fatalf("isolate bundled frontend failed: %v", err)
	}
	entries, err := os.ReadDir(standaloneDir)
	if err != nil {
		t.Fatalf("read standalone frontend directory failed: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "renvo" {
		t.Fatalf("standalone frontend directory entries = %#v", entries)
	}
	if _, err := os.Stat(filepath.Join(standaloneDir, "std")); !os.IsNotExist(err) {
		t.Fatalf("test unexpectedly has an adjacent standard library: %v", err)
	}

	helpDir := t.TempDir()
	cmd = exec.Command(standalone)
	cmd.Dir = helpDir
	cmd.Env = []string{"PWD=" + helpDir}
	help, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("standalone frontend help failed: %v\n%s", err, string(help))
	}
	for _, want := range []string{"Usage: renvo", "file.go...", "Exactly the named files", "Targets:", "windows/amd64", "darwin/arm64", "wasi/wasm32"} {
		if !strings.Contains(string(help), want) {
			t.Fatalf("standalone frontend help missing %q:\n%s", want, string(help))
		}
	}
	cmd = exec.Command(standalone, "run", "--help")
	cmd.Dir = helpDir
	cmd.Env = []string{"PWD=" + helpDir}
	runHelp, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(runHelp), "Usage: renvo run") || !strings.Contains(string(runHelp), "Top-level statements") {
		t.Fatalf("standalone run help failed: err=%v output=%q", err, runHelp)
	}

	project := writeBundleProject(t)
	cmd = exec.Command(standalone, "run", "script.go", "--", "argument")
	cmd.Dir = project
	cmd.Env = []string{"PWD=" + project}
	if out, err := cmd.CombinedOutput(); err != nil || string(out) != "PASS argument\n" {
		t.Fatalf("standalone script execution failed: err=%v output=%q", err, out)
	}

	replBinary := filepath.Join(toolDir, "renvorepl")
	cmd = exec.Command(stage1, "-tags", "renvo_bundle", "-t", "linux/amd64", "-s", "-o", replBinary, "./cmd/renvorepl")
	cmd.Dir = root
	cmd.Env = []string{"PWD=" + root}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("pure Renvo REPL build failed: %v\n%s", err, string(out))
	}
	cmd = exec.Command(replBinary)
	cmd.Dir = helpDir
	cmd.Env = []string{"PWD=" + helpDir}
	cmd.Stdin = strings.NewReader(
		"func initialize() int { print(\"INIT\\n\"); return 40 }\n" +
			"answer := initialize()\n" +
			"answer + 2\n" +
			"answer++\n" +
			"answer\n" +
			"var bonus int = 1\n" +
			"bonus += 2\n" +
			"bonus\n" +
			"items := []int{1}\n" +
			"items = append(items, 7)\n" +
			"items[1]\n" +
			"next := func() int { answer++; return answer }\n" +
			"next()\n" +
			"next()\n" +
			"answer\n" +
			"func twice(v int) int {\nreturn v * 2\n}\n" +
			"twice(answer)\n" +
			"func shadow(answer int) int { return answer }\n" +
			"shadow(9)\n" +
			"func stableValue() int { return 1234 }\n" +
			"saved := stableValue\n" +
			"saved()\n" +
			"import \"strings\"\n" +
			"separatorCount := strings.Count(\"a-a\", \"-\")\n" +
			"separatorCount\n" +
			"saved()\n" +
			":source\n:quit\n",
	)
	replOutput, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pure Renvo REPL session failed: %v\n%s", err, string(replOutput))
	}
	if strings.Count(string(replOutput), "INIT\n") != 1 {
		t.Fatalf("REPL replayed an earlier initializer:\n%s", replOutput)
	}
	if strings.Count(string(replOutput), "renvo> 1234\n") != 2 {
		t.Fatalf("REPL did not preserve a named function value across import relinking:\n%s", replOutput)
	}
	for _, want := range []string{
		"42\n", "41\n", "3\n", "7\n", "42\n", "43\n", "86\n", "9\n", "1234\n", "1\n",
		"var renvo_repl_storage_0 = initialize()",
		"var renvo_repl_value_0 = &renvo_repl_storage_0",
		"func twice(v int) int",
	} {
		if !strings.Contains(string(replOutput), want) {
			t.Fatalf("pure Renvo REPL output missing %q:\n%s", want, string(replOutput))
		}
	}
	if scriptTool, err := exec.LookPath("script"); err == nil {
		cmd = exec.Command(scriptTool, "-qfec", replBinary, "/dev/null")
		cmd.Dir = helpDir
		cmd.Env = []string{"PWD=" + helpDir}
		// Change 2+3 to 2+2 in the middle of the line, recall it, then
		// exercise live binding, package-member, and command completion.
		cmd.Stdin = strings.NewReader(
			"2+3\x1b[D\x1b[3~2\r\x1b[A\r" +
				"answer := 40\rans\t\r" +
				"import \"str\t\t\rstrings.Co\t(\"a\", \t\"a\")\r" +
				":h\t\r:quit\r",
		)
		terminalOutput, terminalErr := cmd.CombinedOutput()
		if terminalErr != nil {
			t.Fatalf("pure Renvo readline session failed: %v\n%s", terminalErr, terminalOutput)
		}
		if bytes.Count(terminalOutput, []byte("\r\n4\r\n")) != 2 ||
			!bytes.Contains(terminalOutput, []byte("\x1b[1D")) ||
			!bytes.Contains(terminalOutput, []byte("renvo> answer\x1b[K")) ||
			!bytes.Contains(terminalOutput, []byte("renvo> import \"strings\"\x1b[K")) ||
			!bytes.Contains(terminalOutput, []byte("renvo> strings.Count\x1b[K")) ||
			!bytes.Contains(terminalOutput, []byte("argument 2: substr string")) ||
			!bytes.Contains(terminalOutput, []byte("renvo> :help\x1b[K")) {
			t.Fatalf("pure Renvo readline did not edit, recall, and complete lines:\n%q", terminalOutput)
		}
	}
	for _, target := range []struct {
		name   string
		suffix string
		prefix []byte
	}{
		{name: "windows/amd64", suffix: "windows-amd64.exe", prefix: []byte{'M', 'Z'}},
		{name: "windows/386", suffix: "windows-386.exe", prefix: []byte{'M', 'Z'}},
		{name: "windows/arm64", suffix: "windows-arm64.exe", prefix: []byte{'M', 'Z'}},
		{name: "darwin/arm64", suffix: "darwin-arm64", prefix: []byte{0xcf, 0xfa, 0xed, 0xfe}},
	} {
		artifact := filepath.Join(toolDir, "renvorepl-"+target.suffix)
		cmd = exec.Command(stage1, "-tags", "renvo_bundle", "-t", target.name, "-s", "-o", artifact, "./cmd/renvorepl")
		cmd.Dir = root
		cmd.Env = []string{"PWD=" + root}
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("pure Renvo REPL build for %s failed: %v\n%s", target.name, err, string(out))
		}
		data, err := os.ReadFile(artifact)
		if err != nil || !bytes.HasPrefix(data, target.prefix) {
			t.Fatalf("pure Renvo REPL artifact for %s: err=%v prefix=% x", target.name, err, data[:minBundleLength(len(data), len(target.prefix))])
		}
		if strings.HasPrefix(target.name, "windows/") {
			_, _, _, _, _, imports, ok := linkedimage.WindowsLayout(data)
			if !ok {
				t.Fatalf("pure Renvo REPL PE layout for %s is invalid", target.name)
			}
			for _, want := range []string{
				"VirtualAlloc", "VirtualProtect", "VirtualFree", "LoadLibraryA",
				"GetProcAddress", "GetCurrentProcess", "FlushInstructionCache",
				"GetStdHandle", "GetConsoleMode", "SetConsoleMode",
			} {
				if !bundleHasNativeImport(imports, want) {
					t.Fatalf("pure Renvo REPL PE for %s is missing import %q: %#v", target.name, want, imports)
				}
			}
		} else {
			_, _, _, imports, libraries, ok := linkedimage.DarwinLayout(data)
			if !ok || len(libraries) == 0 {
				t.Fatalf("pure Renvo REPL Mach-O layout is invalid: libraries=%#v", libraries)
			}
			for _, want := range []string{
				"_mmap", "_mprotect", "_munmap", "_dlopen", "_dlsym",
				"_sys_icache_invalidate", "_tcgetattr", "_tcsetattr", "_cfmakeraw",
			} {
				if !bundleHasNativeImport(imports, want) {
					t.Fatalf("pure Renvo REPL Mach-O is missing import %q: %#v", want, imports)
				}
			}
		}
	}
	var runnable string
	for _, target := range targets {
		output := filepath.Join(project, "app-"+bundleTargetName(target.name))
		cmd = exec.Command(standalone, "-t", target.name, "-s", "-o", output, "./cmd/app")
		cmd.Dir = project
		cmd.Env = []string{"PWD=" + project}
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("standalone bundled compile for %s failed: %v\n%s", target.name, err, string(out))
		}
		data, err := os.ReadFile(output)
		if err != nil {
			t.Fatalf("read %s output failed: %v", target.name, err)
		}
		if !bytes.HasPrefix(data, target.prefix) {
			t.Fatalf("%s output prefix = % x, want % x", target.name, data[:minBundleLength(len(data), len(target.prefix))], target.prefix)
		}
		if target.name == "linux/amd64" {
			runnable = output
		}
	}

	cmd = exec.Command(runnable)
	cmd.Dir = project
	cmd.Env = []string{"PWD=" + project}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("standalone bundled output failed: %v\n%s", err, string(out))
	}
	if string(out) != "PASS\n" {
		t.Fatalf("standalone bundled output = %q", string(out))
	}

	fontOutput := filepath.Join(project, "font-embed-app")
	cmd = exec.Command(standalone, "-s", "-o", fontOutput, "./fontprobe")
	cmd.Dir = project
	cmd.Env = []string{"PWD=" + project}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("standalone bundled font compile failed: %v\n%s", err, string(out))
	}
	cmd = exec.Command(fontOutput)
	cmd.Dir = project
	cmd.Env = []string{"PWD=" + project}
	if out, err := cmd.CombinedOutput(); err != nil || string(out) != "PASS\n" {
		t.Fatalf("standalone bundled font output failed: err=%v output=%q", err, string(out))
	}

	fileOutput := filepath.Join(project, "file-mode-app")
	cmd = exec.Command(standalone, "-s", "-o", fileOutput, "./probe/main_linux_arm64.go")
	cmd.Dir = project
	cmd.Env = []string{"PWD=" + project}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("standalone explicit-file compile failed: %v\n%s", err, string(out))
	}
	cmd = exec.Command(fileOutput)
	cmd.Dir = project
	cmd.Env = []string{"PWD=" + project}
	if out, err := cmd.CombinedOutput(); err != nil || string(out) != "PASS\n" {
		t.Fatalf("standalone explicit-file output failed: err=%v output=%q", err, string(out))
	}
	cmd = exec.Command(standalone, "-o", fileOutput, "./probe/main_linux_arm64.go", "./probe/other.go")
	cmd.Dir = project
	cmd.Env = []string{"PWD=" + project}
	if out, err := cmd.CombinedOutput(); err == nil || !strings.Contains(string(out), "RENVO-LOAD-012") || !strings.Contains(string(out), "other.go") {
		t.Fatalf("standalone mixed-package diagnostic: err=%v output=%q", err, string(out))
	}
	cmd = exec.Command(standalone, "-o", fileOutput, "./probe/main_linux_arm64.go", "./cmd/app/main.go")
	cmd.Dir = project
	cmd.Env = []string{"PWD=" + project}
	if out, err := cmd.CombinedOutput(); err == nil || !strings.Contains(string(out), "RENVO-LOAD-021") {
		t.Fatalf("standalone mixed-directory diagnostic: err=%v output=%q", err, string(out))
	}
}

func TestBundledFrontendNativeREPLSession(t *testing.T) {
	if os.Getenv(frontendBundleTestsEnv) != "1" {
		t.Skipf("set %s=1 to run bundled frontend tests", frontendBundleTestsEnv)
	}
	target := ""
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "windows/amd64":
		target = "windows/amd64"
	case "windows/386":
		target = "windows/386"
	case "windows/arm64":
		target = "windows/arm64"
	case "darwin/arm64":
		target = "darwin/arm64"
	default:
		t.Skipf("native linked-image REPL is not supported on %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	root := repoRoot(t)
	toolDir := t.TempDir()
	executableSuffix := ""
	if runtime.GOOS == "windows" {
		executableSuffix = ".exe"
	}
	backend := filepath.Join(toolDir, "renvo-backend"+executableSuffix)
	buildGoTool(t, root, backend, nil, "./backend")
	stage0 := filepath.Join(toolDir, "renvo-stage0"+executableSuffix)
	buildGoTool(t, root, stage0, []string{"renvo_bundle"}, "./cmd/renvo")
	stage1 := filepath.Join(toolDir, "renvo-stage1"+executableSuffix)
	cmd := exec.Command(stage0, "-tags", "renvo_bundle", "-t", target, "-s", "-o", stage1, "./cmd/renvo")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PWD="+root, "RENVO_BACKEND="+backend)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native bundled frontend self-host build failed: %v\n%s", err, out)
	}
	replBinary := filepath.Join(toolDir, "renvorepl"+executableSuffix)
	cmd = exec.Command(stage1, "-tags", "renvo_bundle", "-t", target, "-s", "-o", replBinary, "./cmd/renvorepl")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PWD="+root, "RENVO_FRONTEND_TRACE=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native pure Renvo REPL build failed: %v\n%s", err, out)
	}
	cmd = exec.Command(replBinary)
	cmd.Dir = toolDir
	cmd.Env = append(os.Environ(), "PWD="+toolDir)
	cmd.Stdin = strings.NewReader(
		"func initialize() int { print(\"INIT\\n\"); return 40 }\n" +
			"answer := initialize()\n" +
			"answer + 2\n" +
			"answer++\n" +
			"answer\n" +
			"next := func() int { answer++; return answer }\n" +
			"next()\n" +
			"next()\n" +
			"func stableValue() int { return 1234 }\n" +
			"saved := stableValue\n" +
			"saved()\n" +
			"import \"strings\"\n" +
			"strings.Count(\"a-a\", \"-\")\n" +
			"saved()\n" +
			":quit\n",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native pure Renvo REPL session failed: %v\n%s", err, output)
	}
	text := strings.ReplaceAll(string(output), "\r\n", "\n")
	if strings.Count(text, "INIT\n") != 1 ||
		strings.Count(text, "renvo> 1234\n") != 2 {
		t.Fatalf("native REPL did not preserve execution state and code pointers:\n%s", text)
	}
	for _, want := range []string{"renvo> 42\n", "renvo> 41\n", "renvo> 42\n", "renvo> 43\n", "renvo> 1\n"} {
		if !strings.Contains(text, want) {
			t.Fatalf("native REPL output missing %q:\n%s", want, text)
		}
	}
}

func bundleHasNativeImport(imports []linkedimage.NativeImport, name string) bool {
	for i := 0; i < len(imports); i++ {
		if imports[i].Name == name {
			return true
		}
	}
	return false
}

func buildGoTool(t *testing.T, root string, output string, tags []string, pkg string) {
	t.Helper()
	args := []string{"build", "-o", output}
	if len(tags) > 0 {
		args = append(args, "-tags", tags[0])
	}
	args = append(args, pkg)
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build %s failed: %v\n%s", pkg, err, string(out))
	}
}

func writeBundleProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/renvobundle\n\ngo 1.25\n\nrequire renvo.dev v0.0.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod failed: %v", err)
	}
	appDir := filepath.Join(dir, "cmd", "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("create app directory failed: %v", err)
	}
	source := `package main

import "strings"

func main() {
	parts := strings.Split("renvo-bundle", "-")
	if len(parts) == 2 && parts[0] == "renvo" && parts[1] == "bundle" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`
	if err := os.WriteFile(filepath.Join(appDir, "main.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write app source failed: %v", err)
	}
	script := `import "os"
print("PASS ")
if len(os.Args) == 2 { print(os.Args[1]) }
print("\n")
`
	if err := os.WriteFile(filepath.Join(dir, "script.go"), []byte(script), 0o644); err != nil {
		t.Fatalf("write script source failed: %v", err)
	}
	fontDir := filepath.Join(dir, "fontprobe")
	if err := os.MkdirAll(fontDir, 0o755); err != nil {
		t.Fatalf("create font probe directory failed: %v", err)
	}
	fontSource := `package main

import "renvo.dev/std/graphics/gofont"

func main() {
	if gofont.New(15) != nil && gofont.NewMono(15) != nil {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`
	if err := os.WriteFile(filepath.Join(fontDir, "main.go"), []byte(fontSource), 0o644); err != nil {
		t.Fatalf("write font probe source failed: %v", err)
	}
	probeDir := filepath.Join(dir, "probe")
	if err := os.MkdirAll(probeDir, 0o755); err != nil {
		t.Fatalf("create file-mode probe directory failed: %v", err)
	}
	probe := `//go:build windows

package main

import "strings"

func main() {
	if strings.Join([]string{"PA", "SS", "\n"}, "") == "PASS\n" {
		print("PASS\n")
	}
}
`
	if err := os.WriteFile(filepath.Join(probeDir, "main_linux_arm64.go"), []byte(probe), 0o644); err != nil {
		t.Fatalf("write file-mode probe failed: %v", err)
	}
	sibling := "package main\nfunc main() { print(\"FAIL sibling included\\n\") }\n"
	if err := os.WriteFile(filepath.Join(probeDir, "sibling.go"), []byte(sibling), 0o644); err != nil {
		t.Fatalf("write file-mode sibling failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(probeDir, "other.go"), []byte("package other\n"), 0o644); err != nil {
		t.Fatalf("write file-mode mixed-package source failed: %v", err)
	}
	return dir
}

func bundleTargetName(target string) string {
	out := make([]byte, len(target))
	for i := 0; i < len(target); i++ {
		if target[i] == '/' {
			out[i] = '-'
		} else {
			out[i] = target[i]
		}
	}
	return string(out)
}

func minBundleLength(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
