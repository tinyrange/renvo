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
		Definition:        string(definition[:]),
		DescriptorVersion: 1,
	})
	if !ok {
		t.Fatal("BindTarget failed")
	}
	got, ok := ReadTargetBinding(bound)
	if !ok || got.Target != "example/new64" || got.Definition != string(definition[:]) || got.DescriptorVersion != 1 {
		t.Fatalf("binding = %#v, ok %v", got, ok)
	}
	if _, ok := ReadTargetBinding(base); ok {
		t.Fatal("unbound unit reported a binding")
	}
}

func TestTargetBindingCanBeReplaced(t *testing.T) {
	base, ok := MarshalCore(CoreProgram{
		Package: "main",
		Text:    []byte("package main"),
		Tokens:  []Token{MakeToken(TokenEOF, 12, 0, 1)},
	})
	if !ok {
		t.Fatal("MarshalCore failed")
	}
	first, ok := BindTarget(base, TargetBinding{Target: "first", Definition: string(make([]byte, 32)), DescriptorVersion: 1})
	if !ok {
		t.Fatal("first BindTarget failed")
	}
	var hash [32]byte
	hash[0] = 9
	second, ok := BindTarget(first, TargetBinding{
		Target: "second", Definition: string(hash[:]), DescriptorVersion: 2,
	})
	if !ok {
		t.Fatal("replacement BindTarget failed")
	}
	got, ok := ReadTargetBinding(second)
	if !ok || got.Target != "second" || got.Definition != string(hash[:]) || got.DescriptorVersion != 2 {
		t.Fatalf("replacement binding = %#v, %v", got, ok)
	}
}

func TestTargetBindingRejectsTruncation(t *testing.T) {
	base, _ := MarshalCore(CoreProgram{
		Package: "main",
		Text:    []byte("package main"),
		Tokens:  []Token{MakeToken(TokenEOF, 12, 0, 1)},
	})
	bound, _ := BindTarget(base, TargetBinding{Target: "test/tiny", Definition: string(make([]byte, 32)), DescriptorVersion: 1})
	for i := 0; i < len(bound); i++ {
		if _, ok := ReadTargetBinding(bound[:i]); ok {
			t.Fatalf("truncation at %d accepted", i)
		}
	}
}
