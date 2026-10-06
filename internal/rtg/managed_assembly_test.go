//go:build !renvo

package rtg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func managedTestResolved(t *testing.T) ResolveResult {
	t.Helper()
	path := filepath.Join("..", "..", "backend", "definitions", "linux_amd64.rtg")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	resolved := ResolveDefinitions(ParseImports(source, path, testFilesystemImportLoader{}))
	if !resolved.Ok {
		t.Fatal(resolved.Diagnostics)
	}
	return resolved
}
func TestManagedAssemblyWordBoundary(t *testing.T) {
	resolved := managedTestResolved(t)
	v := FrontendOperations(resolved, "linux/amd64")
	if !v.Managed.Ok || v.Managed.Arguments[0] != "rdi" || v.Managed.Result != "rax" {
		t.Fatalf("policy: %+v", v.Managed)
	}
	doc := ParseAssembly([]byte("rtgasm 3 assembly { sum(out:emitter) { inputs = 2; add(argument(0),argument(1)); yield(register(rdi)) } }"), "sum.rtgasm")
	lowered := LowerTargetAssembly(resolved, "linux/amd64", doc)
	if !lowered.Ok || lowered.Entries[0].ManagedInputs != 2 || lowered.Entries[0].ManagedOutputs != 1 || len(lowered.Entries[0].Steps) != 2 {
		t.Fatalf("lowered: %+v", lowered)
	}
	b := ManagedBlock{Inputs: 2, Result: "rdi", Block: TargetBlock{Name: "sum", Instructions: []TargetInstruction{{Operation: "add", Operands: []TargetOperand{{Kind: "argument", Value: "0"}, {Kind: "argument", Value: "1"}}}}}}
	encoded, ds := EncodeManagedAssembly(v, []ManagedBlock{b}, "sum.rtgasm")
	again := LowerTargetAssembly(resolved, "linux/amd64", ParseAssembly(encoded, "sum.rtgasm"))
	if len(ds) != 0 || !again.Ok || b.Block.Instructions[0].Operands[0].Kind != "argument" {
		t.Fatalf("round trip or mutation: %s %+v %+v", encoded, ds, again.Diagnostics)
	}
	for _, body := range []string{
		"inputs = 7; add(register(rdi),register(rsi)); yield(register(rdi))",
		"inputs = 1; add(argument(1),register(rax)); yield(register(rax))",
		"inputs = 1; add(register(rbp),register(rax)); yield(register(rax))",
		"inputs = 0; return(); yield()",
		"inputs = 0; move_immediate(register(rbx),int64(1)); yield(register(rbx))",
		"inputs = 0; yield(register(rsp))",
		"inputs = 0; yield(); move_immediate(register(rax),int64(1))",
	} {
		bad := LowerTargetAssembly(resolved, "linux/amd64", ParseAssembly([]byte("rtgasm 3 assembly { bad(out:emitter) { "+body+" } }"), "bad.rtgasm"))
		if bad.Ok || len(bad.Diagnostics) == 0 {
			t.Fatal("accepted", body)
		}
	}
	// The managed body never gets a raw return inserted by the frontend.
	if strings.Contains(string(encoded), "return()") {
		t.Fatal("managed block claims a user-owned ABI")
	}
}
