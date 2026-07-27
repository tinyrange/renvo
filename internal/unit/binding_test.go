package unit

import "testing"

func TestTargetBindingRoundTrip(t *testing.T) {
	base, ok := MarshalCore(CoreProgram{
		Package:    "main",
		ImportPath: "example/main",
		Text:       []byte("package main"),
		Tokens:     []Token{MakeToken(TokenEOF, 12, 0, 1)},
	})
	if !ok {
		t.Fatal("MarshalCore failed")
	}
	var definition [32]byte
	for i := 0; i < len(definition); i++ {
		definition[i] = byte(i)
	}
	bound, ok := BindTarget(base, TargetBinding{
		Target:            "example/new64",
		Definition:        definition,
		DescriptorVersion: 1,
	})
	if !ok {
		t.Fatal("BindTarget failed")
	}
	got, ok := ReadTargetBinding(bound)
	if !ok || got.Target != "example/new64" || got.Definition != definition || got.DescriptorVersion != 1 {
		t.Fatalf("binding = %#v, ok %v", got, ok)
	}
	if _, ok := ReadTargetBinding(base); ok {
		t.Fatal("unbound unit reported a binding")
	}
}

func TestTargetBindingRejectsTruncation(t *testing.T) {
	base, _ := MarshalCore(CoreProgram{
		Package: "main",
		Text:    []byte("package main"),
		Tokens:  []Token{MakeToken(TokenEOF, 12, 0, 1)},
	})
	bound, _ := BindTarget(base, TargetBinding{Target: "test/tiny", DescriptorVersion: 1})
	for i := 0; i < len(bound); i++ {
		if _, ok := ReadTargetBinding(bound[:i]); ok {
			t.Fatalf("truncation at %d accepted", i)
		}
	}
}
