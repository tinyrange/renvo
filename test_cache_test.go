package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"
)

const testArtifactCacheVersion = "v2"

var testArtifactLocks sync.Map
var testProcessSlots = make(chan struct{}, testProcessLimit())

// RTG_TEST_JOBS overrides the maximum number of external compiler and program
// processes run concurrently. The default avoids oversubscribing common hosts.
func testProcessLimit() int {
	if value := os.Getenv("RTG_TEST_JOBS"); value != "" {
		if jobs, err := strconv.Atoi(value); err == nil && jobs > 0 {
			return jobs
		}
	}
	jobs := runtime.NumCPU()
	if jobs > 16 {
		jobs = 16
	}
	if jobs < 1 {
		jobs = 1
	}
	return jobs
}

func acquireTestProcess() func() {
	testProcessSlots <- struct{}{}
	return func() { <-testProcessSlots }
}

func testArtifactKey(parts ...[]byte) string {
	hash := sha256.New()
	fmt.Fprintf(hash, "%s\x00%s\x00%s\x00%s\x00", testArtifactCacheVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	for _, name := range []string{"GOFLAGS", "GOEXPERIMENT", "CGO_ENABLED"} {
		fmt.Fprintf(hash, "%s=%s\x00", name, os.Getenv(name))
	}
	for _, part := range parts {
		fmt.Fprintf(hash, "%d\x00", len(part))
		hash.Write(part)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func testArtifactKeyForFiles(t *testing.T, labels []string, paths []string) string {
	t.Helper()
	parts := make([][]byte, 0, len(labels)+len(paths)*2)
	for _, label := range labels {
		parts = append(parts, []byte(label))
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to hash test artifact input %s: %v", path, err)
		}
		parts = append(parts, []byte(path), data)
	}
	return testArtifactKey(parts...)
}

func testArtifactKeyForFileContents(t *testing.T, labels []string, paths []string) string {
	t.Helper()
	parts := make([][]byte, 0, len(labels)+len(paths))
	for _, label := range labels {
		parts = append(parts, []byte(label))
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to hash test artifact input %s: %v", path, err)
		}
		parts = append(parts, data)
	}
	return testArtifactKey(parts...)
}

func cachedTestArtifact(t *testing.T, kind string, key string, build func(string) error) string {
	t.Helper()
	cacheDir := filepath.Join(".rtg", "test-cache", testArtifactCacheVersion, kind)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("failed to create test artifact cache: %v", err)
	}
	path, err := filepath.Abs(filepath.Join(cacheDir, key))
	if err != nil {
		t.Fatalf("failed to resolve test artifact cache path: %v", err)
	}
	lockValue, _ := testArtifactLocks.LoadOrStore(path, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()
	if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() && info.Size() > 0 {
		return path
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove invalid cached %s artifact: %v", kind, err)
	}
	temp, err := os.CreateTemp(cacheDir, ".build-*")
	if err != nil {
		t.Fatalf("failed to create cached artifact temporary file: %v", err)
	}
	tempPath, err := filepath.Abs(temp.Name())
	if err != nil {
		t.Fatalf("failed to resolve cached artifact temporary path: %v", err)
	}
	if err := temp.Close(); err != nil {
		t.Fatalf("failed to close cached artifact temporary file: %v", err)
	}
	os.Remove(tempPath)
	defer os.Remove(tempPath)
	if err := build(tempPath); err != nil {
		t.Fatalf("failed to build cached %s artifact: %v", kind, err)
	}
	if err := os.Chmod(tempPath, 0755); err != nil {
		t.Fatalf("failed to mark cached %s artifact executable: %v", kind, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		if info, statErr := os.Stat(path); statErr == nil && info.Mode().IsRegular() && info.Size() > 0 {
			return path
		}
		t.Fatalf("failed to publish cached %s artifact: %v", kind, err)
	}
	return path
}

type cachedCommandResultData struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// cachedCommandResult stores the observable result instead of the host Go
// executable. A host executable is several megabytes, while its result is
// normally only a few bytes and is all the comparison tests consume.
func cachedCommandResult(t *testing.T, kind string, key string, run func() (commandResult, error)) commandResult {
	t.Helper()
	cacheDir := filepath.Join(".rtg", "test-cache", testArtifactCacheVersion, kind)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("failed to create command result cache: %v", err)
	}
	path, err := filepath.Abs(filepath.Join(cacheDir, key+".json"))
	if err != nil {
		t.Fatalf("failed to resolve command result cache path: %v", err)
	}
	lockValue, _ := testArtifactLocks.LoadOrStore(path, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	decode := func() (commandResult, bool) {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return commandResult{}, false
		}
		var cached cachedCommandResultData
		if json.Unmarshal(data, &cached) != nil {
			return commandResult{}, false
		}
		return commandResult{stdout: cached.Stdout, stderr: cached.Stderr, exitCode: cached.ExitCode}, true
	}
	if result, ok := decode(); ok {
		return result
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove invalid cached %s result: %v", kind, err)
	}

	result, err := run()
	if err != nil {
		t.Fatalf("failed to produce cached %s result: %v", kind, err)
	}
	data, err := json.Marshal(cachedCommandResultData{
		Stdout: result.stdout, Stderr: result.stderr, ExitCode: result.exitCode,
	})
	if err != nil {
		t.Fatalf("failed to encode cached %s result: %v", kind, err)
	}
	temp, err := os.CreateTemp(cacheDir, ".result-*")
	if err != nil {
		t.Fatalf("failed to create cached result temporary file: %v", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		t.Fatalf("failed to write cached %s result: %v", kind, err)
	}
	if err := temp.Close(); err != nil {
		t.Fatalf("failed to close cached %s result: %v", kind, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		if existing, ok := decode(); ok {
			return existing
		}
		t.Fatalf("failed to publish cached %s result: %v", kind, err)
	}
	return result
}

func cachedStage3Compiler(t *testing.T, target compilerTarget, stage2 string, unitPath string) string {
	t.Helper()
	key := testArtifactKeyForFileContents(t, []string{"stage3", target.name}, []string{stage2, unitPath})
	return cachedTestArtifact(t, "stage3", key, func(output string) error {
		return runTargetCompilerBinary(t, target, stage2, output, []string{unitPath})
	})
}

func cachedTargetProgram(t *testing.T, target compilerTarget, compiler string, inputKind string, inputFiles []string) string {
	t.Helper()
	paths := make([]string, 0, len(inputFiles)+1)
	paths = append(paths, compiler)
	paths = append(paths, inputFiles...)
	key := testArtifactKeyForFiles(t, []string{"target-program", target.name, inputKind}, paths)
	if inputKind == "unit" {
		key = testArtifactKeyForFileContents(t, []string{"target-program", target.name, inputKind}, paths)
	}
	return cachedTestArtifact(t, "target-program", key, func(output string) error {
		return runTargetCompilerBinary(t, target, compiler, output, inputFiles)
	})
}
