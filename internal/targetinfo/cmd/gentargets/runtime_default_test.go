package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise both complete projections with real resolved definitions. Renaming
// public identities must not change compatibility code; moving the explicit
// role must change it even when the old familiar name remains in the registry.
func TestRuntimeNumberDefaultRole(t *testing.T) {
	root, err := repositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "internal/targetinfo/targets.json"))
	if err != nil {
		t.Fatal(err)
	}
	var descriptors []sourceDescriptor
	if err := json.Unmarshal(data, &descriptors); err != nil {
		t.Fatal(err)
	}
	if err := mergeMachineDefinitions(root, descriptors); err != nil {
		t.Fatal(err)
	}
	if err := validate(descriptors); err != nil {
		t.Fatal(err)
	}
	for _, projection := range []struct {
		name, marker string
		generate     func(string, []sourceDescriptor) error
	}{
		{"policy", "TARGET REGISTRY", updatePolicyProjection},
		{"runtime", "RUNTIME NUMBERS", updateRuntimeNumbers},
	} {
		t.Run(projection.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "generated.go")
			seed := []byte("package generated\n// BEGIN GENERATED " + projection.marker + "\n// END GENERATED " + projection.marker + "\n")
			generate := func(ds []sourceDescriptor) ([]byte, error) {
				if err := os.WriteFile(path, seed, 0600); err != nil {
					t.Fatal(err)
				}
				err := projection.generate(path, ds)
				got, readErr := os.ReadFile(path)
				if readErr != nil {
					t.Fatal(readErr)
				}
				return got, err
			}
			baseline, err := generate(descriptors)
			if err != nil {
				t.Fatal(err)
			}
			renamed := append([]sourceDescriptor(nil), descriptors...)
			for i := range renamed {
				renamed[i].Name = "private/" + renamed[i].Constant
				renamed[i].OS = "private_os"
				renamed[i].ISA = "private_isa"
			}
			got, err := generate(renamed)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, baseline) {
				t.Fatal("public identities changed compatibility projection")
			}
			moved := append([]sourceDescriptor(nil), descriptors...)
			for i := range moved {
				moved[i].RuntimeNumberDefault = false
			}
			// Use the second real definition, retaining linux/amd64 without the role.
			moved[1].RuntimeNumberDefault = true
			got, err = generate(moved)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Equal(got, baseline) {
				t.Fatal("projection ignored explicit default role")
			}
			if projection.name == "policy" && !strings.Contains(string(got), "const renvoResolvedLinuxAmd64SysExit = 1") {
				t.Fatal("compatibility constants did not use selected definition's exit number")
			}
			for _, invalid := range []string{"missing", "duplicate", "non-backend", "incomplete"} {
				t.Run(invalid, func(t *testing.T) {
					broken := append([]sourceDescriptor(nil), descriptors...)
					switch invalid {
					case "missing":
						broken[0].RuntimeNumberDefault = false
					case "duplicate":
						broken[1].RuntimeNumberDefault = true
					case "non-backend":
						broken[0].Constant = ""
					case "incomplete":
						broken[0].RuntimeNumbers = map[string]int{"exit": 60}
					}
					if err := validate(broken); err == nil {
						t.Fatal("registry validation accepted invalid role")
					}
					got, err := generate(broken)
					if err == nil {
						t.Fatal("projection accepted invalid role")
					}
					if !bytes.Equal(got, seed) {
						t.Fatal("invalid role modified output")
					}
				})
			}
		})
	}
}
