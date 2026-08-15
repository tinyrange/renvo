package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

const (
	frontendBackendsLimit = int64(2_000_000)
	frontendBundleLimit   = int64(4 * 1024 * 1024)
)

func TestFrontendWithBackendsSize(t *testing.T) {
	path := buildMeasuredFrontend(t, "")
	assertFrontendPayloadSize(t, path, "frontend with backends", frontendBackendsLimit)
}

func TestFrontendWithFullBundleSize(t *testing.T) {
	path := buildMeasuredFrontend(t, "renvo_bundle")
	assertFrontendPayloadSize(t, path, "frontend with std/forms/device bundle", frontendBundleLimit)
}

func buildMeasuredFrontend(t *testing.T, tags string) string {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("frontend payload measurement requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root := repoRoot(t)
	dir := t.TempDir()
	buildGoTool(t, root, filepath.Join(dir, "renvo-backend"), nil, "./backend")
	stage0 := filepath.Join(dir, "renvo-bootstrap")
	buildGoTool(t, root, stage0, []string{"renvo_bundle"}, "./cmd/renvobootstrap")
	output := filepath.Join(dir, "renvo")
	args := []string{"-t", "linux/amd64", "-arena-size", "134217728", "-s", "-o", output}
	if tags != "" {
		args = append([]string{"-tags", tags}, args...)
	}
	args = append(args, "./cmd/renvo")
	cmd := exec.Command(stage0, args...)
	cmd.Dir = root
	cmd.Env = []string{"PWD=" + root}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build measured frontend failed: %v\n%s", err, out)
	}
	return output
}

func assertFrontendPayloadSize(t *testing.T, path, measurement string, limit int64) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s payload=%dB limit=%dB", measurement, info.Size(), limit)
	if info.Size() > limit {
		t.Fatalf("%s payload %dB exceeds %dB limit", measurement, info.Size(), limit)
	}
}
