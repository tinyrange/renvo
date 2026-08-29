package frontend_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

const (
	frontendBackendsReference = int64(2_000_000)
	frontendBundleReference   = int64(4 * 1024 * 1024)
	frontendPayloadMax        = int64(4*1024*1024 + 4*1024)
)

func TestFrontendWithBackendsSize(t *testing.T) {
	path := buildMeasuredFrontend(t, "")
	recordFrontendPayloadSize(t, path, "frontend with backends", frontendBackendsReference)
}

func TestFrontendWithFullBundleSize(t *testing.T) {
	path := buildMeasuredFrontend(t, "renvo_bundle")
	recordFrontendPayloadSize(t, path, "frontend with std/forms/device bundle", frontendBundleReference)
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

func recordFrontendPayloadSize(t *testing.T, path, measurement string, reference int64) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s payload=%dB previous reference=%dB delta=%+dB", measurement, info.Size(), reference, info.Size()-reference)
	if info.Size() > frontendPayloadMax {
		t.Fatalf("%s payload=%dB > %dB", measurement, info.Size(), frontendPayloadMax)
	}
}
