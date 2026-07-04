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
	localNameTok := 20
	localNamePos := localNameTok * tokenStride
	localNameStart := readTokenStart(program.Tokens, localNamePos)
	localNameSize := int(program.Tokens[localNamePos+4]) | int(program.Tokens[localNamePos+5])<<8
	program.ImportPath = "example.com/case/cmd/app"
	program.Imports = []Import{{
		Name:       "lib",
		ImportPath: "example.com/case/pkg/lib",
		Package:    1,
		NameTok:    17,
		PathTok:    17,
	}}
	program.Symbols = []Symbol{{
		Name:       "values",
		Kind:       SymbolVar,
		Package:    0,
		Token:      3,
		OwnerKind:  OwnerDecl,
		OwnerIndex: 0,
	}}
	program.DeclMeta = []DeclMeta{{
		DeclIndex:  0,
		Symbol:     0,
		ValueIndex: 0,
		TypeStart:  -1,
		TypeEnd:    -1,
		ValueStart: 4,
		ValueEnd:   12,
		Values:     []ExprSpan{{StartTok: 4, EndTok: 12}},
	}}
	program.InitOrder = []int{0}
	program.Consts = []ConstValue{{DeclIndex: 0, Kind: ConstInt, Int: -7}}
	program.Signatures = []FuncSignature{{
		FuncIndex: 0,
		Results:   []Field{{NameTok: -1, TypeStart: 17, TypeEnd: 18}},
	}}
	program.Types = []TypeInfo{{
		NameStart: program.Decls[0].NameStart,
		NameEnd:   program.Decls[0].NameEnd,
		Kind:      TypeSlice,
		Decl:      0,
		Symbol:    0,
		Alias:     true,
		TypeStart: 4,
		TypeEnd:   7,
		LenStart:  -1,
		LenEnd:    -1,
		KeyStart:  -1,
		KeyEnd:    -1,
		ElemStart: 6,
		ElemEnd:   7,
	}}
	program.TypeFields = []TypeFields{{
		TypeIndex: 0,
		Fields:    []Field{{NameTok: 3, TypeStart: 4, TypeEnd: 7}},
	}}
	program.TypeIfaces = []TypeIface{{
		TypeIndex: 0,
		Embeds:    []InterfaceEmbed{{TypeStart: 6, TypeEnd: 7}},
		Methods: []InterfaceMethod{{
			NameTok: 17,
			Params:  []Field{{NameTok: -1, TypeStart: 17, TypeEnd: 18}},
			Results: []Field{{NameTok: -1, TypeStart: 17, TypeEnd: 18}},
		}},
	}}
	program.TypeRefs = []TypeRef{{
		OwnerKind:  OwnerDecl,
		OwnerIndex: 0,
		Kind:       TypeRefBuiltin,
		Token:      6,
		BaseTok:    len(program.Tokens)/tokenStride - 1,
		DotTok:     len(program.Tokens)/tokenStride - 1,
		Package:    -1,
		Symbol:     -1,
	}}
	program.Locals = []LocalDecl{{
		FuncIndex:  0,
		Kind:       rtgTokVar,
		NameStart:  localNameStart,
		NameEnd:    localNameStart + localNameSize,
		Token:      localNameTok,
		Scope:      -1,
		ValueIndex: 0,
		TypeStart:  -1,
		TypeEnd:    -1,
		ValueStart: localNameTok,
		ValueEnd:   localNameTok + 1,
		Values:     []ExprSpan{{StartTok: localNameTok, EndTok: localNameTok + 1}},
	}}
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
	program.Calls = []Call{{
		OwnerKind:  OwnerFunc,
		OwnerIndex: 0,
		Kind:       CallPackage,
		CalleeTok:  17,
		BaseTok:    len(program.Tokens)/tokenStride - 1,
		DotTok:     len(program.Tokens)/tokenStride - 1,
		ArgsStart:  19,
		ArgsEnd:    20,
		Args:       []ExprSpan{{StartTok: 19, EndTok: 20}},
	}}
	program.Refs = []NameRef{{
		OwnerKind:  OwnerFunc,
		OwnerIndex: 0,
		Kind:       RefPackage,
		Token:      17,
		Index:      0,
		Package:    0,
	}}
	program.Selectors = []Selector{{
		OwnerKind:   OwnerFunc,
		OwnerIndex:  0,
		Kind:        SelectorImport,
		BaseTok:     17,
		DotTok:      18,
		NameTok:     19,
		BaseKind:    RefImport,
		BaseIndex:   0,
		BasePackage: 0,
		Package:     0,
		Symbol:      0,
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
	if len(decoded.Types) != 1 || decoded.Types[0] != program.Types[0] {
		t.Fatalf("decoded types = %#v, want %#v", decoded.Types, program.Types)
	}
	if len(decoded.TypeFields) != 1 || len(decoded.TypeFields[0].Fields) != 1 || decoded.TypeFields[0].Fields[0] != program.TypeFields[0].Fields[0] {
		t.Fatalf("decoded type fields = %#v, want %#v", decoded.TypeFields, program.TypeFields)
	}
	if len(decoded.TypeIfaces) != 1 || len(decoded.TypeIfaces[0].Embeds) != 1 ||
		len(decoded.TypeIfaces[0].Methods) != 1 ||
		decoded.TypeIfaces[0].Embeds[0] != program.TypeIfaces[0].Embeds[0] ||
		decoded.TypeIfaces[0].Methods[0].NameTok != program.TypeIfaces[0].Methods[0].NameTok ||
		decoded.TypeIfaces[0].Methods[0].Params[0] != program.TypeIfaces[0].Methods[0].Params[0] ||
		decoded.TypeIfaces[0].Methods[0].Results[0] != program.TypeIfaces[0].Methods[0].Results[0] {
		t.Fatalf("decoded type interfaces = %#v, want %#v", decoded.TypeIfaces, program.TypeIfaces)
	}
	if len(decoded.TypeRefs) != 1 || decoded.TypeRefs[0] != program.TypeRefs[0] {
		t.Fatalf("decoded type refs = %#v, want %#v", decoded.TypeRefs, program.TypeRefs)
	}
	if decoded.ImportPath != program.ImportPath || len(decoded.Imports) != 1 || decoded.Imports[0] != program.Imports[0] {
		t.Fatalf("decoded imports = %q %#v, want %q %#v", decoded.ImportPath, decoded.Imports, program.ImportPath, program.Imports)
	}
	if len(decoded.Symbols) != 1 || decoded.Symbols[0] != program.Symbols[0] {
		t.Fatalf("decoded symbols = %#v, want %#v", decoded.Symbols, program.Symbols)
	}
	if len(decoded.InitOrder) != 1 || decoded.InitOrder[0] != program.InitOrder[0] {
		t.Fatalf("decoded init order = %#v, want %#v", decoded.InitOrder, program.InitOrder)
	}
	if len(decoded.Consts) != 1 || decoded.Consts[0] != program.Consts[0] {
		t.Fatalf("decoded consts = %#v, want %#v", decoded.Consts, program.Consts)
	}
	if len(decoded.Signatures) != 1 || len(decoded.Signatures[0].Results) != 1 || decoded.Signatures[0].Results[0] != program.Signatures[0].Results[0] {
		t.Fatalf("decoded signatures = %#v, want %#v", decoded.Signatures, program.Signatures)
	}
	if len(decoded.DeclMeta) != 1 || len(decoded.DeclMeta[0].Values) != 1 || decoded.DeclMeta[0].Values[0] != program.DeclMeta[0].Values[0] {
		t.Fatalf("decoded decl metadata = %#v, want %#v", decoded.DeclMeta, program.DeclMeta)
	}
	if len(decoded.Locals) != 1 || decoded.Locals[0].Kind != rtgTokVar || decoded.Locals[0].Values[0] != program.Locals[0].Values[0] {
		t.Fatalf("decoded locals = %#v, want %#v", decoded.Locals, program.Locals)
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
	if len(decoded.Calls) != 1 || decoded.Calls[0].Kind != CallPackage || decoded.Calls[0].Args[0] != program.Calls[0].Args[0] {
		t.Fatalf("decoded calls = %#v, want %#v", decoded.Calls, program.Calls)
	}
	if len(decoded.Refs) != 1 || decoded.Refs[0] != program.Refs[0] {
		t.Fatalf("decoded refs = %#v, want %#v", decoded.Refs, program.Refs)
	}
	if len(decoded.Selectors) != 1 || decoded.Selectors[0] != program.Selectors[0] {
		t.Fatalf("decoded selectors = %#v, want %#v", decoded.Selectors, program.Selectors)
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
