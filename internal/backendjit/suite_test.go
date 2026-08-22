//go:build !renvo

package backendjit

import (
	"os"
	"testing"
)

var backendJITTestCacheDir string

func TestMain(m *testing.M) {
	cache, err := os.MkdirTemp("", "renvo-backendjit-test-cache-")
	if err != nil {
		panic(err)
	}
	backendJITTestCacheDir = cache
	code := m.Run()
	_ = os.RemoveAll(cache)
	os.Exit(code)
}
