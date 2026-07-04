package lower

import (
	"bytes"
	"testing"

	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/unit"
	"j5.nz/rtg/rtgunit"
)

func TestEmitRootPackageUnit(t *testing.T) {
	graph := loadTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

const answer = 42

func appMain() int {
	return answer
}
`)},
	})
	result := EmitRoot(graph)
	if !result.Ok {
		t.Fatalf("EmitRoot failed: err=%d file=%d tok=%d", result.Error, result.ErrorFile, result.ErrorToken)
	}
	data, ok := unit.Marshal(result.Program)
	if !ok {
		t.Fatal("unit Marshal failed")
	}
	decoded, err := rtgunit.Unmarshal(data)
	if err != nil {
		t.Fatalf("host unit decode failed: %v", err)
	}
	if decoded.Package != "main" {
		t.Fatalf("package = %q, want main", decoded.Package)
	}
	if !bytes.Equal(decoded.Text, result.Program.Text) {
		t.Fatalf("decoded text mismatch")
	}
	if len(decoded.Decls) != 1 || decoded.Decls[0].NameStart >= decoded.Decls[0].NameEnd {
		t.Fatalf("decls = %#v", decoded.Decls)
	}
	if string(decoded.Text[decoded.Decls[0].NameStart:decoded.Decls[0].NameEnd]) != "answer" {
		t.Fatalf("decl name = %q", string(decoded.Text[decoded.Decls[0].NameStart:decoded.Decls[0].NameEnd]))
	}
	if len(decoded.Funcs) != 1 {
		t.Fatalf("funcs = %#v", decoded.Funcs)
	}
	fn := decoded.Funcs[0]
	if string(decoded.Text[fn.NameStart:fn.NameEnd]) != "appMain" {
		t.Fatalf("func name = %q", string(decoded.Text[fn.NameStart:fn.NameEnd]))
	}
	if tokenText(decoded, fn.BodyStart) != "{" || tokenText(decoded, fn.BodyEnd) != "}" {
		t.Fatalf("body tokens = %q:%q", tokenText(decoded, fn.BodyStart), tokenText(decoded, fn.BodyEnd))
	}
}

func TestEmitPackagePreservesTextAndFileOrder(t *testing.T) {
	graph := loadTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/z.go", Src: []byte("package main\n\nfunc z() int { return 2 }\n")},
		{Path: "/repo/case/cmd/app/a.go", Src: []byte("package main\n\nfunc a() int { return 1 }\n")},
	})
	result := EmitRoot(graph)
	if !result.Ok {
		t.Fatalf("EmitRoot failed: err=%d file=%d tok=%d", result.Error, result.ErrorFile, result.ErrorToken)
	}
	wantText := []byte("package main\n\nfunc a() int { return 1 }\npackage main\n\nfunc z() int { return 2 }\n")
	if !bytes.Equal(result.Program.Text, wantText) {
		t.Fatalf("text = %q, want %q", string(result.Program.Text), string(wantText))
	}
	if len(result.Program.Funcs) != 2 {
		t.Fatalf("func count = %d, want 2", len(result.Program.Funcs))
	}
	if string(result.Program.Text[result.Program.Funcs[0].NameStart:result.Program.Funcs[0].NameEnd]) != "a" {
		t.Fatalf("first func = %q", string(result.Program.Text[result.Program.Funcs[0].NameStart:result.Program.Funcs[0].NameEnd]))
	}
	if string(result.Program.Text[result.Program.Funcs[1].NameStart:result.Program.Funcs[1].NameEnd]) != "z" {
		t.Fatalf("second func = %q", string(result.Program.Text[result.Program.Funcs[1].NameStart:result.Program.Funcs[1].NameEnd]))
	}
}

func TestEmitPackagePreservesLinkStaticDirective(t *testing.T) {
	graph := loadTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

// rtg:linkstatic libc,puts
func puts(s string) int { return 0 }

func appMain() int { return puts("PASS\n") }
`)},
	})
	result := EmitRoot(graph)
	if !result.Ok {
		t.Fatalf("EmitRoot failed: err=%d file=%d tok=%d", result.Error, result.ErrorFile, result.ErrorToken)
	}
	if !bytes.Contains(result.Program.Text, []byte("// rtg:linkstatic libc,puts\nfunc puts")) {
		t.Fatalf("linkstatic directive was not preserved: %q", string(result.Program.Text))
	}
	data, ok := unit.Marshal(result.Program)
	if !ok {
		t.Fatal("unit Marshal failed")
	}
	decoded, err := rtgunit.Unmarshal(data)
	if err != nil {
		t.Fatalf("host unit decode failed: %v", err)
	}
	if !bytes.Contains(decoded.Text, []byte("// rtg:linkstatic libc,puts\nfunc puts")) {
		t.Fatalf("decoded text lost directive: %q", string(decoded.Text))
	}
}

func TestEmitTokenKinds(t *testing.T) {
	graph := loadTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

const whole = 1
const fractional = 1.5
const imported = import

func appMain() int {
	if whole > 0 {
		return whole
	}
	return 0
}
`)},
	})
	result := EmitRoot(graph)
	if !result.Ok {
		t.Fatalf("EmitRoot failed: err=%d file=%d tok=%d", result.Error, result.ErrorFile, result.ErrorToken)
	}
	foundFloat := false
	foundImportIdent := false
	foundIf := false
	for i := 0; i < len(result.Program.Tokens); i++ {
		tok := result.Program.Tokens[i]
		text := string(result.Program.Text[tok.Start : tok.Start+tok.Size])
		if text == "1.5" && tok.Kind == unit.TokenFloat {
			foundFloat = true
		}
		if text == "import" && tok.Kind == unit.TokenIdent {
			foundImportIdent = true
		}
		if text == "if" && tok.Kind == unit.TokenIf {
			foundIf = true
		}
	}
	if !foundFloat {
		t.Fatal("float literal was not emitted as unit.TokenFloat")
	}
	if !foundImportIdent {
		t.Fatal("unsupported import keyword was not downgraded to unit.TokenIdent")
	}
	if !foundIf {
		t.Fatal("if keyword was not emitted as unit.TokenIf")
	}
}

func TestEmitPackageRejectsInvalidPackage(t *testing.T) {
	result := EmitPackage(load.Package{})
	if result.Ok || result.Error != EmitErrPackage {
		t.Fatalf("empty package result = %#v", result)
	}
}

func loadTestGraph(t *testing.T, files []load.SourceFile) load.Graph {
	t.Helper()
	mod := load.Module{Root: "/repo/case", Path: "example.com/case", Ok: true}
	graph := load.LoadGraph(mod, "/std", "/repo/case", "./cmd/app", files)
	if !graph.Ok {
		t.Fatalf("LoadGraph failed: err=%d pkg=%d graph=%#v", graph.Error, graph.ErrorPackage, graph)
	}
	return graph
}

func tokenText(program rtgunit.Program, index int) string {
	if index < 0 || index*8+8 > len(program.Tokens) {
		return ""
	}
	pos := index * 8
	start := int(program.Tokens[pos+1]) | int(program.Tokens[pos+2])<<8 | int(program.Tokens[pos+3])<<16
	size := int(program.Tokens[pos+4])
	if int(program.Tokens[pos]) != unit.TokenOp {
		size = size | int(program.Tokens[pos+5])<<8
	}
	if start < 0 || size < 0 || start+size > len(program.Text) {
		return ""
	}
	return string(program.Text[start : start+size])
}
