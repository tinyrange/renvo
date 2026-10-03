package frontend_tests

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	wireunit "renvo.dev/backend/unit"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/unit"
)

func TestFrontendGroupedVariablesPreserveAssemblyDeclarations(t *testing.T) {
	root := repoRoot(t)
	dir := t.TempDir()
	assembly := []byte("rtgasm 1\nassembly { answer(out:emitter) { out.Byte(0xc3) } }\n")
	for name, source := range map[string][]byte{
		"go.mod":        []byte("module example.com/assemblygroups\ngo 1.23\n"),
		"main.go":       []byte("package main\nvar(a=17;b=19)\nfunc answer()int\nfunc main(){var(c=23;d=29);print(answer()+a+b+c+d)}\n"),
		"answer.rtgasm": assembly,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), source, 0644); err != nil {
			t.Fatal(err)
		}
	}
	stdRoot := filepath.Join(root, "std")
	built := driver.BuildPackageUnitFromFS(".", "linux/amd64", nil, dir, stdRoot, driver.OSFS{})
	if !built.Ok {
		t.Fatal(driver.FormatDiagnostic(built.Diagnostic))
	}
	program := built.Pipeline.Link.Program
	if len(program.RTGAssemblyFuncs) != 1 {
		t.Fatalf("assembly bindings: %+v", program.RTGAssemblyFuncs)
	}
	binding := program.RTGAssemblyFuncs[0]
	if binding.Func < 0 || binding.Func >= len(program.Funcs) || binding.Source != 0 || binding.Entry != 0 {
		t.Fatalf("invalid assembly binding: %+v", binding)
	}
	fn := program.Funcs[binding.Func]
	if string(program.Text[fn.NameStart:fn.NameEnd]) != "answer" || fn.BodyStart != fn.EndTok || fn.BodyEnd != fn.EndTok {
		t.Fatalf("assembly declaration changed: %+v", fn)
	}
	compact := driver.BuildPackageUnitCompact(".", "linux/amd64", nil, dir, stdRoot, driver.OSFS{})
	if !compact.Ok {
		t.Fatal(driver.FormatDiagnostic(compact.Diagnostic))
	}
	if !unit.HasRTGAssembly(compact.Unit) || !bytes.Contains(compact.Unit, assembly) {
		t.Fatal("compact frontend lost assembly source")
	}
	decoded, err := wireunit.Unmarshal(compact.Unit)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, fn := range decoded.Funcs {
		if string(decoded.Text[fn.NameStart:fn.NameEnd]) == "answer" {
			found = true
			if fn.BodyStart != fn.EndTok || fn.BodyEnd != fn.EndTok {
				t.Fatalf("compact assembly declaration changed: %+v", fn)
			}
		}
	}
	if !found {
		t.Fatal("compact frontend lost assembly declaration")
	}
}
