package main

import (
	"bytes"
	"os"
	"testing"
)

func TestSpecializePreparationSource(t *testing.T) {
	source := []byte("//go:build !renvo_prepared\n\npackage main\n\nconst renvoPreparedBackendActive = 0\nconst renvoRTGStructuredFunctions = 0\n")
	prepared, err := specializePreparationSource("compiler_target_policy_impl.go", source)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(prepared, []byte("//go:build renvo_prepared\n")) ||
		!bytes.Contains(prepared, []byte("const renvoPreparedBackendActive = 1")) {
		t.Fatalf("prepared source = %q", prepared)
	}
	if bytes.Contains(prepared, []byte("const renvoRTGStructuredFunctions")) {
		t.Fatalf("prepared source retained ordinary structured-function mode: %q", prepared)
	}
	if !bytes.Contains(source, []byte("const renvoPreparedBackendActive = 0")) {
		t.Fatal("specialization mutated its input")
	}
}

func TestSpecializePreparationSourceRejectsInvalidSetting(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"missing tag", "package main\nconst renvoPreparedBackendActive = 0\nconst renvoRTGStructuredFunctions = 0\n"},
		{"missing", "//go:build !renvo_prepared\n\npackage main\n"},
		{"variable", "//go:build !renvo_prepared\n\npackage main\nvar renvoPreparedBackendActive = 0\nconst renvoRTGStructuredFunctions = 0\n"},
		{"multiple names", "//go:build !renvo_prepared\n\npackage main\nconst renvoPreparedBackendActive, other = 0, 0\nconst renvoRTGStructuredFunctions = 0\n"},
		{"duplicate", "//go:build !renvo_prepared\n\npackage main\nconst renvoPreparedBackendActive = 0\nconst renvoPreparedBackendActive = 0\nconst renvoRTGStructuredFunctions = 0\n"},
		{"missing structured mode", "//go:build !renvo_prepared\n\npackage main\nconst renvoPreparedBackendActive = 0\n"},
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

func TestSpecializePreparationSourceFoldsPreparedBranches(t *testing.T) {
	source := []byte(`package main
const renvoPreparedBackendActive = 0
func selected(value bool) int {
	if renvoPreparedBackendActive != 0 { return 1 }
	if value && renvoPreparedBackendActive == 0 { return 2 }
	return 3
}
`)
	prepared, err := specializePreparationSource("compiler_main.go", source)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(prepared, []byte("return 1")) || bytes.Contains(prepared, []byte("return 2")) ||
		bytes.Contains(prepared, []byte("return 3")) {
		t.Fatalf("prepared branches were not folded: %s", prepared)
	}
}

func TestSpecializePreparationSourcePreservesUnknownLeftOperandEvaluation(t *testing.T) {
	source := []byte(`package main
const renvoPreparedBackendActive = 0
func observed() bool { return true }
func selected() int {
	if observed() && renvoPreparedBackendActive == 0 { return 1 }
	if observed() || renvoPreparedBackendActive != 0 { return 2 }
	return 3
}
`)
	prepared, err := specializePreparationSource("compiler_main.go", source)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(prepared, []byte("observed()")) != 3 ||
		!bytes.Contains(prepared, []byte("return 1")) || !bytes.Contains(prepared, []byte("return 2")) {
		t.Fatalf("prepared source erased left-operand evaluation: %s", prepared)
	}
}

func TestSpecializePreparationSourceDoesNotFoldShadowedSettingOrGotoBody(t *testing.T) {
	source := []byte(`package main
const renvoPreparedBackendActive = 0
func shadowed(renvoPreparedBackendActive int) int {
	if renvoPreparedBackendActive != 0 { return 1 }
	return 2
}
func labeled(value bool) int {
	if value { goto chosen }
	if renvoPreparedBackendActive != 0 { return 3 }
	return 4
chosen:
	return 5
}
`)
	prepared, err := specializePreparationSource("compiler_main.go", source)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range [][]byte{
		[]byte("renvoPreparedBackendActive != 0"), []byte("return 1"),
		[]byte("goto chosen"), []byte("return 4"), []byte("chosen:"),
	} {
		if !bytes.Contains(prepared, text) {
			t.Fatalf("prepared source removed %q from a shadowed or goto body: %s", text, prepared)
		}
	}
}

func TestCheckedInPreparedPolicy(t *testing.T) {
	source, err := os.ReadFile("../../../../backend/compiler_target_policy_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	generated, err := specializePreparationSource("compiler_target_policy_impl.go", source)
	if err != nil {
		t.Fatal(err)
	}
	checkedIn, err := os.ReadFile("../../../../backend/compiler_target_policy_prepared_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated, checkedIn) {
		t.Fatal("checked-in prepared target policy is stale; regenerate it with internal/backendcompiled/cmd/gen")
	}
}
