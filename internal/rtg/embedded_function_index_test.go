package rtg

import (
	"strings"
	"testing"
)

func TestEmbeddedFunctionIndexDeclarationBoundaries(t *testing.T) {
	document := Document{Declarations: []Declaration{
		{Kind: DeclGo, Name: "backend", GoSource: []byte("func hook() { wrongKind() }")},
		{Kind: DeclGo, Name: "compiler", GoSource: []byte("func (v value) hook() { method() }")},
		{Kind: DeclGo, Name: "compiler", GoSource: []byte("func broken(")},
		{Kind: DeclGo, Name: "compiler", GoSource: []byte("func hook(a *renvoAsm) { first(); return }\nfunc labeled(a *renvoAsm) { again: goto again }")},
		{Kind: DeclGo, Name: "compiler", GoSource: []byte("func hook(a *renvoAsm) { second() }")},
	}}
	indexed := indexEmbeddedFunctions(document, "compiler")
	hook, found := indexedEmbeddedFunction(indexed, "hook")
	if !found || !strings.Contains(string(hook.Body), "first()") || !hook.EndsInReturn || hook.HasLabels {
		t.Fatalf("lookup must use the first free function in a valid compiler block: %#v", hook)
	}
	if len(hook.Parameters) != 1 || hook.Parameters[0].Name != "a" || string(hook.Signature) != "(a *renvoAsm)" {
		t.Fatalf("lost hook signature: %#v", hook)
	}
	labeled, _ := indexedEmbeddedFunction(indexed, "labeled")
	if !labeled.HasLabels || len(indexed) != 2 {
		t.Fatalf("lost label safety or admitted an invalid declaration: %#v", indexed)
	}
	// Equal function names in another generation must not reuse this document.
	other := Document{Declarations: []Declaration{{Kind: DeclGo, Name: "compiler", GoSource: []byte("func hook(a *renvoAsm) { other() }")}}}
	if got, _ := indexedEmbeddedFunction(indexEmbeddedFunctions(other, "compiler"), "hook"); !strings.Contains(string(got.Body), "other()") {
		t.Fatalf("function index leaked across documents: %#v", got)
	}
	// A new generation after an edit must not retain a stale parsed body.
	document.Declarations[3].GoSource = []byte("func hook(a *renvoAsm) { edited() }")
	if got, _ := indexedEmbeddedFunction(indexEmbeddedFunctions(document, "compiler"), "hook"); !strings.Contains(string(got.Body), "edited()") {
		t.Fatalf("function index retained a previous generation: %#v", got)
	}
}
