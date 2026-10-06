//go:build !renvo

package rtg

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func frontendTestDefinition(t *testing.T) []byte {
	t.Helper()
	source, err := os.ReadFile("../../backends/msdos.rtg")
	if err != nil {
		t.Fatal(err)
	}
	return source
}
func frontendTestResolved(t *testing.T, source []byte) ResolveResult {
	t.Helper()
	r := Resolve(ParseImports(source, "../../backends/msdos.rtg", testFilesystemImportLoader{}))
	if !r.Ok {
		t.Fatalf("resolve: %+v", r.Diagnostics)
	}
	return r
}

func TestFrontendOperationContract(t *testing.T) {
	base := frontendTestDefinition(t)
	r := frontendTestResolved(t, base)
	v := FrontendOperations(r, "msdos/8086")
	if !v.Ok || len(v.Operations) != 9 || len(v.Registers) != 8 || len(v.Conditions) != 10 {
		t.Fatalf("vocabulary: %+v", v)
	}
	for _, test := range []struct{ old, replacement, diagnostic string }{
		{"lower = go emitImmediate", "lower = go emitReturn", "RTG-FRONTEND-004"},
		{"ordering = ordered", "ordering = unordered", "RTG-FRONTEND-002"},
		{"writes = [destination]", "writes = []", "RTG-FRONTEND-004"},
		{"clobbers = [flags]", "clobbers = [made_up]", "RTG-FRONTEND-004"},
		{"references = [destination]", "references = []", "RTG-FRONTEND-004"},
		{"immediates { value = bits 16 }", "immediates { value = bits 65 }", "RTG-FRONTEND-002"},
		{"requires = []", "requires = []; mystery = []", "RTG-FRONTEND-002"},
		{"memory = none", "memory = none; memory = read", "RTG-FRONTEND-002"},
	} {
		t.Run(test.replacement, func(t *testing.T) {
			source := bytes.Replace(base, []byte(test.old), []byte(test.replacement), 1)
			if bytes.Equal(source, base) {
				t.Fatalf("fixture does not contain %q", test.old)
			}
			bad := Resolve(ParseImports(source, "../../backends/msdos.rtg", testFilesystemImportLoader{}))
			if bad.Ok || !hasDiagnosticCode(bad.Diagnostics, test.diagnostic) {
				t.Fatalf("accepted invalid contract: %+v", bad.Diagnostics)
			}
		})
	}
}

func TestFrontendOperationsCapabilityAndDefinitionIdentity(t *testing.T) {
	base := frontendTestDefinition(t)
	original := frontendTestResolved(t, base)
	restricted := frontendTestResolved(t, bytes.Replace(base, []byte("requires = []"), []byte("requires = [future_extension]"), 1))
	v := FrontendOperations(restricted, "msdos/8086")
	if !v.Ok || len(v.Operations) != 8 || original.Targets[0].Descriptor.Definition == restricted.Targets[0].Descriptor.Definition {
		t.Fatal("capability contract was not applied or fingerprinted")
	}
	assembly := ParseAssembly([]byte("rtgasm 2 assembly { answer(out:emitter) { let done = new_label(); bind_label(done); return() } }"), "restricted.rtgasm")
	lowered := LowerTargetAssembly(restricted, "msdos/8086", assembly)
	if lowered.Ok || !strings.Contains(lowered.Diagnostics[0].Message, "unavailable") {
		t.Fatalf("capability bypass: %+v", lowered.Diagnostics)
	}
}

func TestFrontendTypedAssemblyRejectsEscapesAndBadLabels(t *testing.T) {
	r := frontendTestResolved(t, frontendTestDefinition(t))
	for _, test := range []struct{ body, message string }{
		{"emitReturn(out)", "unknown or unavailable"},
		{"out.Byte(0xc3)", "expected a declared"},
		{"move_immediate(register(ax), int(1+2))", "not an expression"},
		{"move_immediate(register(ax), int(65536))", "declared range"},
		{"move_immediate(register(ax), int(-32769))", "declared range"},
		{"move_immediate(register(ax), int(2147483648))", "invalid int"},
		{"move_immediate(register(ax), int64(42))", "requires int"},
		{"move_immediate(register(r0), int(42))", "invalid register"},
		{"jump(label(0))", "invalid label"},
		{"jump(done)", "unknown machine result"},
		{"let done = new_label(); jump(done)", "unbound machine label"},
		{"let done = new_label(); bind_label(done); bind_label(done)", "bound more than once"},
		{"let done = new_label(); let done = new_label()", "duplicate machine result"},
		{"let unused = new_label(); return()", "unused machine result"},
		{"new_label()", "incompatible result"},
		{"let x = return()", "incompatible result"},
		{"if true { return() }", "nested source"},
	} {
		t.Run(test.body, func(t *testing.T) {
			a := ParseAssembly([]byte("rtgasm 2\nassembly { answer(out:emitter) {\n"+test.body+"\n} }"), "bad.rtgasm")
			if !a.Ok {
				t.Fatalf("wrapper parse: %+v", a.Diagnostics)
			}
			lowered := LowerTargetAssembly(r, "msdos/8086", a)
			if lowered.Ok || len(lowered.Diagnostics) != 1 || !strings.Contains(lowered.Diagnostics[0].Message, test.message) || lowered.Diagnostics[0].Filename != "bad.rtgasm" || lowered.Diagnostics[0].Span.Start.Line < 3 {
				t.Fatalf("diagnostic: %+v", lowered.Diagnostics)
			}
			if generated := GenerateAssemblyEvaluator(r, "msdos/8086", a, 0); generated.Ok {
				t.Fatal("direct evaluator API bypassed typed validation")
			}
		})
	}
}

func TestFrontendProgrammaticAssemblyRoundTrip(t *testing.T) {
	r := frontendTestResolved(t, frontendTestDefinition(t))
	v := FrontendOperations(r, "msdos/8086")
	blocks := []TargetBlock{{Name: "answer", Instructions: []TargetInstruction{
		{Operation: "new_label", Result: "out"}, // safe even when a result shadows emitter/register names
		{Operation: "jump", Operands: []TargetOperand{{Kind: "value", Value: "out"}}},
		{Operation: "move_immediate", Operands: []TargetOperand{{Kind: "register", Value: "ax"}, {Kind: "int", Value: "0xffff"}}},
		{Operation: "bind_label", Operands: []TargetOperand{{Kind: "value", Value: "out"}}},
		{Operation: "move_immediate", Operands: []TargetOperand{{Kind: "register", Value: "ax"}, {Kind: "int", Value: "+42"}}},
		{Operation: "return"},
	}}}
	encoded, diagnostics := EncodeTargetAssembly(v, blocks, "answer.rtgasm")
	if len(diagnostics) != 0 || !bytes.Contains(encoded, []byte("int(65535)")) || !bytes.Contains(encoded, []byte("int(42)")) {
		t.Fatalf("encode: %s %+v", encoded, diagnostics)
	}
	parsed := ParseAssembly(encoded, "answer.rtgasm")
	lowered := LowerTargetAssembly(r, "msdos/8086", parsed)
	if !lowered.Ok || parsed.Version != 2 || lowered.Version != 1 || len(lowered.Entries[0].Steps) != 6 {
		t.Fatalf("round trip: %+v", lowered.Diagnostics)
	}
	if generated := GenerateAssemblyEvaluator(r, "msdos/8086", parsed, 0); !generated.Ok {
		t.Fatalf("generate: %+v", generated.Diagnostics)
	}
	blocks[0].Instructions[4].Operands[1].Value = "1); out.Byte(0xc3); int(2"
	if data, d := EncodeTargetAssembly(v, blocks, "bad.rtgasm"); data != nil || len(d) == 0 {
		t.Fatal("frontend expression injection was serialized")
	}
}

func TestFrontendImmediateBoundaries(t *testing.T) {
	for _, test := range []struct {
		operand TargetOperand
		limit   TargetImmediate
		fits    bool
	}{
		{TargetOperand{"int64", "-9223372036854775808"}, TargetImmediate{"value", "signed", 64}, true},
		{TargetOperand{"int64", "9223372036854775807"}, TargetImmediate{"value", "signed", 64}, true},
		{TargetOperand{"uint64", "18446744073709551615"}, TargetImmediate{"value", "unsigned", 64}, true},
		{TargetOperand{"uint64", "9223372036854775808"}, TargetImmediate{"value", "signed", 64}, false},
		{TargetOperand{"int", "-1"}, TargetImmediate{"value", "unsigned", 8}, false},
		{TargetOperand{"byte", "255"}, TargetImmediate{"value", "bits", 8}, true},
		{TargetOperand{"int", "-128"}, TargetImmediate{"value", "bits", 8}, true},
		{TargetOperand{"int", "-129"}, TargetImmediate{"value", "bits", 8}, false},
	} {
		if got := targetImmediateFits(test.operand, test.limit); got != test.fits {
			t.Fatalf("%+v %+v = %v", test.operand, test.limit, got)
		}
	}
}
