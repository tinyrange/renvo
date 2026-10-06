package lower

import "testing"

func TestParseRTGAssemblyBindingsPreservesEntryOrder(t *testing.T) {
	source := []byte("rtgasm 1\nassembly { first(out:emitter) { out.Byte('}') } /* gap */ second(out:emitter) {} }")
	document := parseRTGAssemblyBindings(source)
	if !document.ok || len(document.entries) != 2 || document.entries[0].name != "first" || document.entries[1].name != "second" {
		t.Fatalf("bindings = %#v", document)
	}
}

func TestParseRTGAssemblyBindingsRejectsDuplicates(t *testing.T) {
	document := parseRTGAssemblyBindings([]byte("rtgasm 1 assembly { same(out:emitter) {} same(out:emitter) {} }"))
	if document.ok {
		t.Fatalf("duplicate bindings accepted: %#v", document)
	}
}

func TestRTGAssemblyBindingsAllowSignatureTrivia(t *testing.T) {
	for _, source := range []string{
		"rtgasm 2 assembly { answer ( out : emitter ) { return() } }",
		"rtgasm 2 assembly { answer /* name */ (out /* kind */ : emitter) /* body */ { return() } }",
		"rtgasm 1 assembly { answer\n(\nout\n:\nemitter\n)\n{ out.Byte(0xc3) } }",
	} {
		d := parseRTGAssemblyBindings([]byte(source))
		if !d.ok || len(d.entries) != 1 || d.entries[0].name != "answer" {
			t.Fatalf("wrapper binding: %q %+v", source, d)
		}
	}
}
