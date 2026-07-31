package main

import (
	"bytes"
	"testing"
)

func TestSpecializePreparationSource(t *testing.T) {
	source := []byte("package main\n\nconst renvoPreparedBackend = 0\n")
	prepared, err := specializePreparationSource("compiler_target_policy_impl.go", source)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(prepared, []byte("const renvoPreparedBackend = 1")) {
		t.Fatalf("prepared source = %q", prepared)
	}
	if !bytes.Contains(source, []byte("const renvoPreparedBackend = 0")) {
		t.Fatal("specialization mutated its input")
	}
}

func TestSpecializePreparationSourceRejectsInvalidSetting(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"missing", "package main\n"},
		{"variable", "package main\nvar renvoPreparedBackend = 0\n"},
		{"multiple names", "package main\nconst renvoPreparedBackend, other = 0, 0\n"},
		{"duplicate", "package main\nconst renvoPreparedBackend = 0\nconst renvoPreparedBackend = 0\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := specializePreparationSource(
				"compiler_target_policy_impl.go", []byte(test.source)); err == nil {
				t.Fatal("accepted invalid preparation setting")
			}
		})
	}
}

func TestSpecializePreparationSourceIgnoresOtherFiles(t *testing.T) {
	source := []byte("not Go source")
	prepared, err := specializePreparationSource("compiler_main.go", source)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(prepared, source) {
		t.Fatalf("other source changed from %q to %q", source, prepared)
	}
}
