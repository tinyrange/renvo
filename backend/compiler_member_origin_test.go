package main

import (
	"os"
	"testing"

	"renvo.dev/backend/unit"
	"renvo.dev/internal/targetinfo"
	internalunit "renvo.dev/internal/unit"
	"renvo.dev/std/vm"
)

func TestPrivateMemberIdentityUsesCompactPackageOwnership(t *testing.T) {
	for _, path := range []string{"tests/private_struct_type_identity.go", "tests/private_method_type_identity.go"} {
		t.Run(path, func(t *testing.T) {
			for _, distinct := range []bool{false, true} {
				name := "same_package_fragments"
				if distinct {
					name = "different_packages"
				}
				t.Run(name, func(t *testing.T) {
					resetRuntime()
					program, err := unit.ConvertFiles([]string{path})
					if err != nil {
						t.Fatal(err)
					}
					if len(program.Decls) != 2 || len(program.Funcs) != 3 {
						t.Fatal("unexpected regression declaration layout")
					}
					tokenCount := len(program.Tokens) / 8
					boundaries := []int{0, program.Decls[1].StartTok, program.Funcs[2].StartTok, tokenCount}
					textBoundary := func(token int) int {
						if token == tokenCount {
							return len(program.Text)
						}
						pos := token * 8
						return int(program.Tokens[pos+1]) | int(program.Tokens[pos+2])<<8 | int(program.Tokens[pos+3])<<16
					}
					// Each fragment has the exact source and table spans a package
					// linker supplies. Equal paths must keep one identity even when
					// generated helpers create multiple fragments for that package.
					for part := 0; part < 3; part++ {
						tokenStart, tokenEnd := boundaries[part], boundaries[part+1]
						start, end := textBoundary(tokenStart), textBoundary(tokenEnd)
						declStart, declEnd := part, part+1
						if part == 2 {
							declEnd = 2
						}
						path := "example.com/same"
						if distinct && part == 1 {
							path = "example.com/other"
						}
						program.Packages = append(program.Packages, unit.PackageInfo{
							Name: "main", ImportPath: path,
							TextStart: start, TextEnd: end, TokenStart: tokenStart, TokenEnd: tokenEnd,
							DeclStart: declStart, DeclEnd: declEnd, FuncStart: part, FuncEnd: part + 1,
						})
					}
					encoded, err := unit.Marshal(program)
					if err != nil {
						t.Fatal(err)
					}
					target, definition, version, found := targetinfo.Binding("vm/vm32")
					if !found {
						t.Fatal("VM32 binding unavailable")
					}
					encoded, bound := internalunit.BindTarget(encoded, internalunit.TargetBinding{Target: target, Definition: definition, DescriptorVersion: version})
					if !bound {
						t.Fatal("bind compact unit")
					}
					image, ok := RenvoCompileUnitToBytesWithOptions(encoded, "vm/vm32", RenvoCompileOptions{ArenaSize: 262144, StripSymbols: true})
					if !ok {
						t.Fatal("compile compact unit")
					}
					args := []string{"program"}
					if distinct {
						args = append(args, "distinct")
					}
					want, err := os.ReadFile(path[:len(path)-3] + ".expected")
					if err != nil {
						t.Fatal(err)
					}
					result := vm.RunConfig(image, vm.Config{Args: args, Limits: vm.Limits{Steps: 1000000, Memory: 2 * 1024 * 1024}})
					if result.Trap != vm.TrapNone || result.ExitCode != 0 || string(result.Output) != string(want) || len(result.Stderr) != 0 {
						t.Fatalf("trap=%d exit=%d output=%q stderr=%q", result.Trap, result.ExitCode, result.Output, result.Stderr)
					}
				})
			}
		})
	}
}
