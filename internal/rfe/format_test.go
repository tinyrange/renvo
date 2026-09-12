package rfe

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormat(t *testing.T) {
	source := []byte("RFE 1\n-- main.go --\npackage demo\nfunc Main(args []string)int{return 0}\n-- rfe.json --\n{\"import\":\"example/demo\",\"name\":\"demo\",\"format\":1,\"entry\":\"Main\"}\n")
	out, err := Format(source)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Format(out)
	if err != nil || !bytes.Equal(out, again) {
		t.Fatalf("formatter not idempotent: %v", err)
	}
	if !strings.Contains(string(out), "func Main(args []string) int { return 0 }") || !bytes.HasPrefix(out, []byte("RFE 1\n\n-- rfe.json --\n")) {
		t.Fatalf("unexpected formatting:\n%s", out)
	}
	before, _ := Decode(source)
	after, _ := Decode(out)
	if before.Manifest.Name != after.Manifest.Name {
		t.Fatal("manifest changed")
	}
}
func TestFormatRejectsInvalidSections(t *testing.T) {
	header := "RFE 1\n-- rfe.json --\n{\"format\":1,\"name\":\"demo\",\"import\":\"example/demo\"}\n"
	for _, tail := range []string{"-- ../escape.go --\npackage demo\n", "-- main.go --\npackage demo\nfunc {\n", "-- main.go --\npackage demo\n-- main.go --\npackage demo\n"} {
		if _, err := Format([]byte(header + tail)); err == nil {
			t.Fatal("accepted invalid section", tail)
		}
	}
}
func TestCheckedInRFEFormatting(t *testing.T) {
	files, err := filepath.Glob("../../emulators/*.rfe")
	if err != nil || len(files) == 0 {
		t.Fatal("missing emulator sources", err)
	}
	for _, file := range files {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		out, err := Format(source)
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		if !bytes.Equal(source, out) {
			t.Errorf("%s needs renvofmt -w", file)
		}
	}
}
