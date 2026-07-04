package rtgunit

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestMarshalRoundTrip(t *testing.T) {
	program := testProgram(t, []byte(`package main

const answer = 42

func appMain() int { return answer }
`))
	data, err := Marshal(program)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	decoded, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if decoded.Package != program.Package ||
		!bytes.Equal(decoded.Text, program.Text) ||
		!bytes.Equal(decoded.Tokens, program.Tokens) ||
		len(decoded.Decls) != len(program.Decls) ||
		len(decoded.Funcs) != len(program.Funcs) {
		t.Fatalf("decoded program = %#v, want %#v", decoded, program)
	}
	if len(decoded.Decls) != 1 || decoded.Decls[0].Kind != rtgTokConst {
		t.Fatalf("decls = %#v, want one const decl", decoded.Decls)
	}
	if len(decoded.Funcs) != 1 {
		t.Fatalf("funcs = %#v, want one function", decoded.Funcs)
	}
	if !bytes.Contains(Source(decoded), []byte("func appMain")) {
		t.Fatalf("Source() did not include function text: %q", string(Source(decoded)))
	}
}

func TestConvertFiles(t *testing.T) {
	src := []byte(`package main

const answer = 42

// rtg:linkstatic libc,puts
func puts(s string) int { return 0 }

func appMain() int { return answer }
`)
	program := testProgram(t, src)
	if program.Package != "main" {
		t.Fatalf("package = %q, want main", program.Package)
	}
	if len(program.Decls) != 1 {
		t.Fatalf("decl count = %d, want 1", len(program.Decls))
	}
	if len(program.Funcs) != 2 {
		t.Fatalf("func count = %d, want 2", len(program.Funcs))
	}
	if len(program.Tokens)%tokenStride != 0 {
		t.Fatalf("token table size = %d, want multiple of %d", len(program.Tokens), tokenStride)
	}
	if !bytes.Contains(program.Text, []byte("// rtg:linkstatic libc,puts")) {
		t.Fatalf("linkstatic directive was not preserved: %q", string(program.Text))
	}
	data, err := Marshal(program)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if !bytes.HasPrefix(data, []byte(Magic)) {
		t.Fatalf("unit missing magic: %q", data[:4])
	}
}

func TestMarshalRoundTripExpressionShapes(t *testing.T) {
	program := testProgram(t, []byte(`package main

var values = []int{1, 2}

func appMain() int { return values[0] }
`))
	program.Indexes = []IndexExpr{{
		OwnerKind:  OwnerFunc,
		OwnerIndex: 0,
		StartTok:   17,
		EndTok:     21,
		BaseStart:  17,
		BaseEnd:    18,
		OpenTok:    18,
		CloseTok:   20,
		IndexStart: 19,
		IndexEnd:   20,
	}}
	program.Composites = []CompositeExpr{{
		OwnerKind:  OwnerDecl,
		OwnerIndex: 0,
		StartTok:   4,
		EndTok:     12,
		TypeStart:  4,
		TypeEnd:    7,
		OpenTok:    7,
		CloseTok:   11,
		Elems:      []ExprSpan{{StartTok: 8, EndTok: 9}, {StartTok: 10, EndTok: 11}},
	}}
	program.Assigns = []Assignment{{
		FuncIndex:  0,
		Kind:       AssignSet,
		StartTok:   17,
		EndTok:     21,
		OpTok:      18,
		LeftStart:  17,
		LeftEnd:    18,
		RightStart: 19,
		RightEnd:   20,
		Targets:    []ExprSpan{{StartTok: 17, EndTok: 18}},
		Values:     []ExprSpan{{StartTok: 19, EndTok: 20}},
	}}
	program.Returns = []Return{{
		FuncIndex: 0,
		StartTok:  16,
		EndTok:    21,
		Values:    []ExprSpan{{StartTok: 17, EndTok: 21}},
	}}
	data, err := Marshal(program)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	decoded, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(decoded.Indexes) != 1 || decoded.Indexes[0] != program.Indexes[0] {
		t.Fatalf("decoded indexes = %#v, want %#v", decoded.Indexes, program.Indexes)
	}
	if len(decoded.Composites) != 1 || len(decoded.Composites[0].Elems) != 2 {
		t.Fatalf("decoded composites = %#v, want %#v", decoded.Composites, program.Composites)
	}
	if decoded.Composites[0].OwnerKind != program.Composites[0].OwnerKind ||
		decoded.Composites[0].OwnerIndex != program.Composites[0].OwnerIndex ||
		decoded.Composites[0].TypeStart != program.Composites[0].TypeStart ||
		decoded.Composites[0].TypeEnd != program.Composites[0].TypeEnd ||
		decoded.Composites[0].Elems[0] != program.Composites[0].Elems[0] ||
		decoded.Composites[0].Elems[1] != program.Composites[0].Elems[1] {
		t.Fatalf("decoded composite = %#v, want %#v", decoded.Composites[0], program.Composites[0])
	}
	if len(decoded.Assigns) != 1 || decoded.Assigns[0].Kind != AssignSet ||
		decoded.Assigns[0].Targets[0] != program.Assigns[0].Targets[0] ||
		decoded.Assigns[0].Values[0] != program.Assigns[0].Values[0] {
		t.Fatalf("decoded assigns = %#v, want %#v", decoded.Assigns, program.Assigns)
	}
	if len(decoded.Returns) != 1 || decoded.Returns[0].Values[0] != program.Returns[0].Values[0] {
		t.Fatalf("decoded returns = %#v, want %#v", decoded.Returns, program.Returns)
	}
}

func testProgram(t *testing.T, src []byte) Program {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, src, 0o644); err != nil {
		t.Fatalf("failed to write source: %v", err)
	}
	program, err := ConvertFiles([]string{path})
	if err != nil {
		t.Fatalf("ConvertFiles failed: %v", err)
	}
	return program
}
