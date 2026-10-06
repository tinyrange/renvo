//go:build !renvo

package rtg

import (
	"strings"
	"testing"
)

func TestSemanticIntrinsicOwnsRuntimeAllocation(t *testing.T) {
	resolved := managedTestResolved(t)
	v := FrontendOperations(resolved, "linux/amd64")
	if len(v.Intrinsics) != 2 || v.Intrinsics[0].Parameters[0].Kind != "word" {
		t.Fatalf("intrinsic metadata: %+v", v)
	}
	encoded, diagnostics := EncodeIntrinsicAssembly(v, []IntrinsicBlock{{Name: "sum", Intrinsic: "word_add"}}, "sum.rtgasm")
	if len(diagnostics) != 0 || strings.Contains(string(encoded), "register(") {
		t.Fatalf("frontend controls allocation: %s %+v", encoded, diagnostics)
	}
	lowered := LowerTargetAssembly(resolved, "linux/amd64", ParseAssembly(encoded, "sum.rtgasm"))
	if !lowered.Ok || lowered.Entries[0].ManagedInputs != 2 || lowered.Entries[0].ManagedOutputs != 1 {
		t.Fatalf("lowering: %+v", lowered)
	}
	for _, body := range []string{
		"inputs=1; intrinsic=word_add; yield()",
		"inputs=2; intrinsic=privateEncoder; yield()",
		"inputs=2; intrinsic=word_add; yield(register(rdi))",
		"inputs=2; intrinsic=word_add; move(register(rax),register(rdi)); yield()",
	} {
		bad := LowerTargetAssembly(resolved, "linux/amd64", ParseAssembly([]byte("rtgasm 3 assembly {sum(out:emitter){"+body+"}}"), "bad.rtgasm"))
		if bad.Ok {
			t.Fatal("accepted", body)
		}
	}
	// Validate invalid semantic contracts through the same parsed declaration
	// decoder, without depending on a particular backend helper spelling.
	contracts := []string{
		"bad(x:register)->word",
		"bad(x:pointer)->word",
		"bad(x:word)->pointer",
		"bad(result:word)->word",
		"bad(x:word)->register",
	}
	for _, signature := range contracts {
		doc := Parse([]byte("arch test { frontend_intrinsics {"+signature+" {lower=go f;reads=[x];writes=[result];clobbers=[];memory=none;control=none;ordering=ordered;requires=[]}}}"), "bad.rtg")
		declaration, _ := doc.Declaration(DeclArch, "test")
		child := declaration.Statements[0].Children[0]
		if _, message := decodeFrontendIntrinsic(child); message == "" {
			t.Fatal("accepted", signature)
		}
	}
}
