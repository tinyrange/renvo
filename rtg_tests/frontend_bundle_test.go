package rtg_tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const frontendBundleTestsEnv = "RTG_FRONTEND_BUNDLE_TESTS"

func TestBundledFrontendStandaloneAllTargets(t *testing.T) {
	if os.Getenv(frontendBundleTestsEnv) != "1" {
		t.Skipf("set %s=1 to run bundled frontend tests", frontendBundleTestsEnv)
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("bundled frontend execution requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	root := repoRoot(t)
	toolDir := t.TempDir()
	backend := filepath.Join(toolDir, "rtgx-backend")
	buildGoTool(t, root, backend, nil, ".")
	stage0 := filepath.Join(toolDir, "rtg-stage0")
	buildGoTool(t, root, stage0, []string{"rtg_bundle"}, "./rtg/cmd/rtg")
	stage1 := filepath.Join(toolDir, "rtg-stage1")
	cmd := exec.Command(stage0, "-tags", "rtg_bundle", "-t", "linux/amd64", "-s", "-o", stage1, "./rtg/cmd/rtg")
	cmd.Dir = root
	cmd.Env = []string{"PWD=" + root, "RTG_BACKEND=" + backend}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bundled frontend self-host build failed: %v\n%s", err, string(out))
	}
	targets := []struct {
		name     string
		artifact string
		prefix   []byte
	}{
		{name: "linux/amd64", artifact: "rtg-linux-amd64", prefix: []byte{0x7f, 'E', 'L', 'F'}},
		{name: "linux/386", artifact: "rtg-linux-386", prefix: []byte{0x7f, 'E', 'L', 'F'}},
		{name: "linux/aarch64", artifact: "rtg-linux-arm64", prefix: []byte{0x7f, 'E', 'L', 'F'}},
		{name: "linux/arm", artifact: "rtg-linux-arm", prefix: []byte{0x7f, 'E', 'L', 'F'}},
		{name: "windows/amd64", artifact: "rtg-windows-amd64.exe", prefix: []byte{'M', 'Z'}},
		{name: "windows/386", artifact: "rtg-windows-386.exe", prefix: []byte{'M', 'Z'}},
		{name: "darwin/arm64", artifact: "rtg-darwin-arm64", prefix: []byte{0xcf, 0xfa, 0xed, 0xfe}},
		{name: "wasi/wasm32", artifact: "rtg-wasi-wasm32.wasm", prefix: []byte{0, 'a', 's', 'm'}},
	}
	var releaseFrontend string
	for _, target := range targets {
		artifact := filepath.Join(toolDir, target.artifact)
		cmd = exec.Command(stage1, "-tags", "rtg_bundle", "-t", target.name, "-s", "-o", artifact, "./rtg/cmd/rtg")
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
	standalone := filepath.Join(standaloneDir, "rtg")
	if err := os.Rename(releaseFrontend, standalone); err != nil {
		t.Fatalf("isolate bundled frontend failed: %v", err)
	}
	entries, err := os.ReadDir(standaloneDir)
	if err != nil {
		t.Fatalf("read standalone frontend directory failed: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "rtg" {
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
	for _, want := range []string{"Usage: rtg", "Targets:", "windows/amd64", "darwin/arm64", "wasi/wasm32"} {
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
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/rtgbundle\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("write go.mod failed: %v", err)
	}
	appDir := filepath.Join(dir, "cmd", "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("create app directory failed: %v", err)
	}
	source := `package main

import "strings"

func main() {
	parts := strings.Split("rtg-bundle", "-")
	if len(parts) == 2 && parts[0] == "rtg" && parts[1] == "bundle" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`
	if err := os.WriteFile(filepath.Join(appDir, "main.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write app source failed: %v", err)
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
