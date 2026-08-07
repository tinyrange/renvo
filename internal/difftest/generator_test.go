//go:build !renvo

package difftest

import (
	"bytes"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGenerateIsDeterministicAndValidSyntax(t *testing.T) {
	if len(caseFamilies) != len(caseFamilyNames) {
		t.Fatalf("%d generators have %d names", len(caseFamilies), len(caseFamilyNames))
	}
	first, err := Generate(42, 30)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Generate(42, 30)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("same seed generated different source")
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "main.go", first, parser.AllErrors); err != nil {
		t.Fatalf("generated source did not parse: %v\n%s", err, first)
	}
}

func TestGenerateNamedFamily(t *testing.T) {
	source, err := GenerateFamily(7, 3, "expression-tree")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "main.go", source, parser.AllErrors); err != nil {
		t.Fatalf("generated family did not parse: %v\n%s", err, source)
	}
	if _, err := GenerateFamily(7, 1, "missing"); err == nil {
		t.Fatal("unknown family was accepted")
	}
}

func TestGenerateSwarmIsDeterministicAndValid(t *testing.T) {
	first, err := GenerateSwarm(19, 40)
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateSwarm(19, 40)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("same swarm seed generated different source")
	}
	if err := validateHostSource(first); err != nil {
		t.Fatalf("generated swarm did not type check: %v\n%s", err, first)
	}
}

func TestGenerationPoliciesAreDeterministicAndValid(t *testing.T) {
	for _, policy := range Policies() {
		first, err := GeneratePolicy(23, 90, policy)
		if err != nil {
			t.Fatalf("%s: %v", policy, err)
		}
		second, err := GeneratePolicy(23, 90, policy)
		if err != nil {
			t.Fatalf("%s repeat: %v", policy, err)
		}
		if !bytes.Equal(first, second) {
			t.Fatalf("%s generated different source for the same seed", policy)
		}
		if err := validateHostSource(first); err != nil {
			t.Fatalf("%s did not type check: %v\n%s", policy, err, first)
		}
	}
	if _, err := GeneratePolicy(1, 1, "missing"); err == nil {
		t.Fatal("unknown generation policy was accepted")
	}
	if source, err := GenerateFamilyPolicy(29, 4, "policy-materialization", "materialize-interface"); err != nil {
		t.Fatal(err)
	} else if err := validateHostSource(source); err != nil {
		t.Fatalf("family policy did not type check: %v\n%s", err, source)
	}
}

func TestGeneratedFamiliesTypeCheck(t *testing.T) {
	for _, family := range caseFamilyNames {
		for seed := uint64(1); seed <= 8; seed++ {
			source, err := GenerateFamily(seed, 1, family)
			if err != nil {
				t.Fatalf("%s seed %d: %v", family, seed, err)
			}
			if err := validateHostSource(source); err != nil {
				t.Fatalf("%s seed %d did not type check: %v\n%s", family, seed, err, source)
			}
		}
	}
}

func TestMetamorphicVariantsAreDistinctAndValid(t *testing.T) {
	source, err := Generate(31, 12)
	if err != nil {
		t.Fatal(err)
	}
	variants, err := Variants(source, 31)
	if err != nil {
		t.Fatal(err)
	}
	if len(variants) != 4 {
		t.Fatalf("variant count = %d, want 4", len(variants))
	}
	names := make(map[string]bool)
	for _, variant := range variants {
		if names[variant.Name] {
			t.Fatalf("duplicate variant name %q", variant.Name)
		}
		names[variant.Name] = true
		if bytes.Equal(source, variant.Source) {
			t.Fatalf("%s did not change the source", variant.Name)
		}
		if err := validateHostSource(variant.Source); err != nil {
			t.Fatalf("%s did not type check: %v\n%s", variant.Name, err, variant.Source)
		}
	}
}

func TestRunnerMatchesHostGo(t *testing.T) {
	if HostTarget() == "" {
		t.Skipf("no runnable Renvo target for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	source := []byte(`package main

func main() {
	value := int64(7)
	for index := 0; index < 4; index++ {
		value = value*3 + int64(index)
	}
	print(value)
	print("\n")
}
`)
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	comparison, err := (Runner{
		StdRoot: filepath.Join(filepath.Dir(filename), "..", "..", "std"),
		Target:  HostTarget(),
	}).Compare(source)
	if err != nil {
		t.Fatal(err)
	}
	if comparison.Interesting {
		t.Fatalf("runner baseline differs (%s): host=%q renvo=%q diagnostic=%s", comparison.Signature, comparison.Host.Output, comparison.Renvo.Output, comparison.Renvo.Diagnostic)
	}
}

func TestMinimizeRemovesUninterestingFunction(t *testing.T) {
	source := []byte(`package main
func irrelevant() int { return 10 }
func broken() int { return 42 }
func main() { print(broken()) }
`)
	minimized, err := Minimize(source, func(candidate []byte) (bool, error) {
		return bytes.Contains(candidate, []byte("broken")) && bytes.Contains(candidate, []byte("main")), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(minimized, []byte("irrelevant")) {
		t.Fatalf("irrelevant declaration remains:\n%s", minimized)
	}
}

func TestMinimizeReplacesConditionalWithInterestingBranch(t *testing.T) {
	source := []byte(`package main
func main() {
	if 1 == 2 {
		print("unused")
	} else {
		print("broken")
	}
}
`)
	minimized, err := Minimize(source, func(candidate []byte) (bool, error) {
		return bytes.Contains(candidate, []byte(`print("broken")`)), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(minimized, []byte("if ")) || bytes.Contains(minimized, []byte("unused")) {
		t.Fatalf("conditional structure remains:\n%s", minimized)
	}
}
