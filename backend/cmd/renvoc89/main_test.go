package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWithoutArgumentsShowsHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout.String(), "usage: renvoc89") || !strings.Contains(stdout.String(), "-mode") {
		t.Fatalf("help output = %q", stdout.String())
	}
}

func TestRunEmitsExplicitReproducibleProfile(t *testing.T) {
	args := []string{
		"-mode", "explicit",
		"-name", "mcu32",
		"-abi", "ilp32e",
		"-endian", "big",
		"-int", "32",
		"-pointer", "32",
		"-runtime", "result",
		"-runtime", "write",
	}
	var first bytes.Buffer
	var second bytes.Buffer
	var stderr bytes.Buffer
	if code := run(args, &first, &stderr); code != 0 {
		t.Fatalf("first exit = %d, stderr = %s", code, stderr.String())
	}
	stderr.Reset()
	if code := run(args, &second, &stderr); code != 0 {
		t.Fatalf("second exit = %d, stderr = %s", code, stderr.String())
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("identical profile arguments produced different C")
	}
	for _, want := range []string{
		"RENVO_C89_ASSERT(language_int_width",
		"RENVO_C89_ASSERT(pointer_width",
		"#define RENVO_C_ENDIAN_BIG 1",
		"#define RENVO_C_RUNTIME_RESULT 1",
		"#define RENVO_C_RUNTIME_WRITE 1",
		"renvo_uint rgsdv",
	} {
		if !strings.Contains(first.String(), want) {
			t.Errorf("generated C missing %q", want)
		}
	}
}

func TestRunWritesPreambleFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile.h")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"-preamble-only", "-hosted", "-o", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), "#define RENVO_C_HOSTED 1") || strings.Contains(string(source), "renvo_uint rgsdv") {
		t.Fatalf("unexpected preamble output:\n%s", source)
	}
}

func TestRunRejectsAmbiguousOrInvalidProfiles(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "automatic widths", args: []string{"-int", "32"}, want: "only valid in explicit mode"},
		{name: "missing widths", args: []string{"-mode", "explicit"}, want: "requires -int and -pointer"},
		{name: "invalid width", args: []string{"-mode", "explicit", "-int", "8", "-pointer", "32"}, want: "16, 32, or 64"},
		{name: "endian", args: []string{"-endian", "native"}, want: "invalid endianness"},
		{name: "mode", args: []string{"-mode", "c32"}, want: "invalid mode"},
		{name: "runtime", args: []string{"-runtime", "not-valid"}, want: "invalid runtime operation"},
		{name: "argument", args: []string{"input.go"}, want: "unexpected argument"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if code := run(test.args, &stdout, &stderr); code != 2 {
				t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
			}
			if !strings.Contains(stderr.String(), test.want) {
				t.Fatalf("stderr missing %q: %s", test.want, stderr.String())
			}
		})
	}
}
