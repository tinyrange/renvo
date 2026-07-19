package frontend_tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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

	project := writeBundleProject(t)
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
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/renvobundle\n\ngo 1.25\n"), 0o644); err != nil {
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
