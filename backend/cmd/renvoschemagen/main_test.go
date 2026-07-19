package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGeneratedSchemaIsCurrent(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate generator source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	cmd := exec.Command("go", "run", "./cmd/renvoschemagen", "-schema", "unit/schema.json", "-root", ".", "-check")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated schema drift: %v\n%s", err, output)
	}
}
