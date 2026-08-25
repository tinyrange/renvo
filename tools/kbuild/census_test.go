package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCensusShellWordsAndFlagClassification(t *testing.T) {
	words := shellWords(`cc -DNAME='kernel value' -Wp,-MMD,obj.d -c source.c -o source.o`)
	if len(words) != 7 || words[1] != "-DNAME=kernel value" || words[3] != "-c" || classifyFlag(words[2]) != "-Wp," {
		t.Fatalf("words/classification = %#v/%q", words, classifyFlag(words[2]))
	}
	keys := sortedKeys(map[string]int{"z": 1, "a": 2})
	if len(keys) != 2 || keys[0] != "a" || keys[1] != "z" {
		t.Fatalf("sorted keys = %#v", keys)
	}
}

func TestAcceptedTranslationUnitsAreMonotonic(t *testing.T) {
	result, err := collect(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.AcceptedUnits) != 1 || result.AcceptedUnits[0] != "lib/union_find.c" {
		t.Fatalf("accepted translation units = %#v", result.AcceptedUnits)
	}
}

func TestRenvoCCUsesHostOnlyForAssemblyAndM16(t *testing.T) {
	dir := t.TempDir()
	frontend := filepath.Join(dir, "frontend")
	external := filepath.Join(dir, "external")
	for path, source := range map[string]string{
		frontend: "#!/bin/sh\nprintf 'frontend:%s\\n' \"$1\"\n",
		external: "#!/bin/sh\nprintf 'external\\n'\n",
	} {
		if err := os.WriteFile(path, []byte(source), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	wrapper, err := filepath.Abs("renvo-cc")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{name: "ordinary C", args: []string{"-c", "main.c"}, want: "frontend:cc"},
		{name: "m16 C", args: []string{"-m16", "-c", "main.c"}, want: "frontend:cc"},
		{name: "assembly suffix", args: []string{"-c", "entry.S"}, want: "external"},
		{name: "assembly language", args: []string{"-x", "assembler-with-cpp", "-c", "entry"}, want: "external"},
		{name: "m16 macro text", args: []string{"-DVALUE=-m16", "-c", "main.c"}, want: "frontend:cc"},
	} {
		t.Run(test.name, func(t *testing.T) {
			command := exec.Command(wrapper, test.args...)
			command.Env = append(os.Environ(), "RENVO_KBUILD_COMPILER="+frontend, "RENVO_KBUILD_EXTERNAL_CC="+external)
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("renvo-cc: %v\n%s", err, output)
			}
			if got := strings.TrimSpace(string(output)); got != test.want {
				t.Fatalf("route = %q, want %q", got, test.want)
			}
		})
	}
}
