package frontend_tests

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
)

// Compiler binaries are shared by sync.Once across top-level tests, so their
// directories must outlive any individual t.Cleanup and belong to the suite.
var frontendTemps struct {
	sync.Mutex
	paths  []string
	closed bool
}

func frontendSuiteTempDir(pattern string) (string, error) {
	frontendTemps.Lock()
	defer frontendTemps.Unlock()
	if frontendTemps.closed {
		return "", fmt.Errorf("frontend test suite is shutting down")
	}
	dir, err := os.MkdirTemp("", pattern)
	if err == nil {
		frontendTemps.paths = append(frontendTemps.paths, dir)
	}
	return dir, err
}

func cleanupFrontendSuiteTemps() bool {
	frontendTemps.Lock()
	defer frontendTemps.Unlock()
	frontendTemps.closed = true
	ok := true
	for _, dir := range frontendTemps.paths {
		if err := os.RemoveAll(dir); err != nil {
			fmt.Fprintf(os.Stderr, "frontend test cleanup %s: %v\n", dir, err)
			ok = false
		}
	}
	return ok
}

func TestMain(m *testing.M) {
	signals := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-signals:
			cleanupFrontendSuiteTemps()
			if sig == os.Interrupt {
				os.Exit(130)
			}
			os.Exit(143)
		case <-done:
		}
	}()
	code := m.Run()
	if !cleanupFrontendSuiteTemps() && code == 0 {
		code = 1
	}
	signal.Stop(signals)
	close(done)
	os.Exit(code)
}
