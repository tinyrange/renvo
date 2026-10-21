//go:build !renvo

package driver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

var testBackend struct {
	once      sync.Once
	dir, path string
	output    []byte
	err       error
}

func TestMain(m *testing.M) {
	code := m.Run()
	if testBackend.dir != "" {
		if err := os.RemoveAll(testBackend.dir); err != nil {
			fmt.Fprintln(os.Stderr, "remove driver test backend:", err)
			if code == 0 {
				code = 1
			}
		}
	}
	os.Exit(code)
}

// The backend executable is immutable. Each test still starts a fresh compiler
// process and uses its own inputs/outputs; only the identical Go build is shared.
func driverTestBackend(t *testing.T) string {
	t.Helper()
	root := driverRepoRoot(t)
	testBackend.once.Do(func() {
		testBackend.dir, testBackend.err = os.MkdirTemp("", "renvo-driver-test-backend-")
		if testBackend.err != nil {
			return
		}
		testBackend.path = filepath.Join(testBackend.dir, "renvo-backend")
		if runtime.GOOS == "windows" {
			testBackend.path += ".exe"
		}
		cmd := exec.Command("go", "build", "-o", testBackend.path, "./backend")
		cmd.Dir = root
		testBackend.output, testBackend.err = cmd.CombinedOutput()
	})
	if testBackend.err != nil {
		t.Fatalf("backend build failed: %v\n%s", testBackend.err, testBackend.output)
	}
	return testBackend.path
}
