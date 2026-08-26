package rtg

import "testing"

func TestParseAssemblyPreservesEntryBodies(t *testing.T) {
	source := []byte("rtgasm 1\nassembly {\n byteSwap64(out:emitter) {\n  out.Bytes3(0x48, 0x0f, 0xc8)\n }\n wait(out:emitter) { let loop = out.NewLabel(); out.Mark(loop) }\n}\n")
	document := ParseAssembly(source, "bits_amd64.rtgasm")
	if !document.Ok || len(document.Diagnostics) != 0 {
		t.Fatalf("ParseAssembly failed: %#v", document.Diagnostics)
	}
	if document.Version != 1 || len(document.Entries) != 2 {
		t.Fatalf("assembly document = %#v", document)
	}
	first := document.Entries[0]
	if first.Name != "byteSwap64" || string(source[first.BodyStart:first.BodyEnd]) != "\n  out.Bytes3(0x48, 0x0f, 0xc8)\n " {
		t.Fatalf("first entry = %#v body %q", first, source[first.BodyStart:first.BodyEnd])
	}
}

func TestParseAssemblyRejectsDuplicateEntries(t *testing.T) {
	document := ParseAssembly([]byte("rtgasm 1 assembly { f(out:emitter) {} f(out:emitter) {} }"), "duplicate.rtgasm")
	if document.Ok || len(document.Diagnostics) != 1 || document.Diagnostics[0].Code != "RTGASM-PARSE-009" {
		t.Fatalf("duplicate diagnostic = %#v", document.Diagnostics)
	}
}

func TestGenerateAssemblyEvaluatorRejectsNonSequenceStatements(t *testing.T) {
	resolved := Resolve(Parse([]byte(completeDirectEmitterDefinition()), "tiny.rtg"))
	assembly := ParseAssembly([]byte("rtgasm 1\nassembly { f(out:emitter) { mystery = 1 } }"), "bad.rtgasm")
	generated := GenerateAssemblyEvaluator(resolved, "test/tiny64", assembly, 0)
	if generated.Ok || len(generated.Diagnostics) != 1 || generated.Diagnostics[0].Code != "RTGASM-GENERATE-003" {
		t.Fatalf("unsupported statement diagnostics = %#v", generated.Diagnostics)
	}
}
