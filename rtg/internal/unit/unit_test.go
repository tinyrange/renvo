package unit

import (
	"bytes"
	"testing"

	"j5.nz/rtg/rtgunit"
)

func TestMarshalMatchesHostUnitEncoder(t *testing.T) {
	program := helloProgram()
	data, ok := Marshal(program)
	if !ok {
		t.Fatal("Marshal failed")
	}

	hostProgram := rtgunit.Program{
		Package: program.Package,
		Text:    program.Text,
		Tokens:  hostTokenBytes(program),
		Funcs: []rtgunit.Func{{
			NameStart:     program.Funcs[0].NameStart,
			NameEnd:       program.Funcs[0].NameEnd,
			StartTok:      program.Funcs[0].StartTok,
			NameTok:       program.Funcs[0].NameTok,
			ReceiverStart: program.Funcs[0].ReceiverStart,
			ReceiverEnd:   program.Funcs[0].ReceiverEnd,
			BodyStart:     program.Funcs[0].BodyStart,
			BodyEnd:       program.Funcs[0].BodyEnd,
			EndTok:        program.Funcs[0].EndTok,
		}},
	}
	hostData, err := rtgunit.Marshal(hostProgram)
	if err != nil {
		t.Fatalf("host Marshal failed: %v", err)
	}
	if !bytes.Equal(data, hostData) {
		t.Fatalf("frontend unit bytes differ from host encoder\nfrontend=%v\nhost=%v", data, hostData)
	}
}

func TestMarshalDecodesWithHostUnitDecoder(t *testing.T) {
	program := declProgram()
	data, ok := Marshal(program)
	if !ok {
		t.Fatal("Marshal failed")
	}
	decoded, err := rtgunit.Unmarshal(data)
	if err != nil {
		t.Fatalf("host Unmarshal failed: %v", err)
	}
	if decoded.Package != program.Package {
		t.Fatalf("package = %q, want %q", decoded.Package, program.Package)
	}
	if !bytes.Equal(decoded.Text, program.Text) {
		t.Fatalf("text = %q, want %q", string(decoded.Text), string(program.Text))
	}
	if !bytes.Equal(decoded.Tokens, hostTokenBytes(program)) {
		t.Fatalf("decoded tokens = %v, want %v", decoded.Tokens, hostTokenBytes(program))
	}
	if len(decoded.Decls) != 1 {
		t.Fatalf("decl count = %d, want 1", len(decoded.Decls))
	}
	if decoded.Decls[0].NameStart != program.Decls[0].NameStart || decoded.Decls[0].NameEnd != program.Decls[0].NameEnd {
		t.Fatalf("decoded decl = %#v, want %#v", decoded.Decls[0], program.Decls[0])
	}
	if len(decoded.Funcs) != 1 {
		t.Fatalf("func count = %d, want 1", len(decoded.Funcs))
	}
	if decoded.Funcs[0].BodyStart != program.Funcs[0].BodyStart || decoded.Funcs[0].BodyEnd != program.Funcs[0].BodyEnd {
		t.Fatalf("decoded func = %#v, want %#v", decoded.Funcs[0], program.Funcs[0])
	}
}

func TestMarshalRoundTripInternalDecoder(t *testing.T) {
	program := declProgram()
	data, ok := Marshal(program)
	if !ok {
		t.Fatal("Marshal failed")
	}
	decoded, ok := Unmarshal(data)
	if !ok {
		t.Fatal("Unmarshal failed")
	}
	if !equalPrograms(decoded, program) {
		t.Fatalf("decoded program = %#v, want %#v", decoded, program)
	}
}

func TestMarshalRoundTripExpressionShapes(t *testing.T) {
	program := declProgram()
	program.ImportPath = "example.com/case/cmd/app"
	program.Imports = []Import{{
		Name:       "lib",
		ImportPath: "example.com/case/pkg/lib",
		Package:    1,
		NameTok:    13,
		PathTok:    13,
	}}
	program.Symbols = []Symbol{{
		Name:       "answer",
		Kind:       SymbolConst,
		Package:    0,
		Token:      3,
		OwnerKind:  OwnerDecl,
		OwnerIndex: 0,
	}, {
		Name:       "appMain",
		Kind:       SymbolFunc,
		Package:    0,
		Token:      7,
		OwnerKind:  OwnerFunc,
		OwnerIndex: 0,
	}}
	program.DeclMeta = []DeclMeta{{
		DeclIndex:  0,
		Symbol:     0,
		ValueIndex: 0,
		TypeStart:  -1,
		TypeEnd:    -1,
		ValueStart: 5,
		ValueEnd:   6,
		Values:     []ExprSpan{{StartTok: 5, EndTok: 6}},
	}}
	program.InitOrder = []int{0}
	program.Consts = []ConstValue{{DeclIndex: 0, Kind: ConstInt, Int: 42}}
	program.Signatures = []FuncSignature{{
		FuncIndex: 0,
		Results:   []Field{{NameTok: -1, TypeStart: 10, TypeEnd: 11}},
	}}
	program.Stmts = []Statement{{
		FuncIndex: 0,
		Kind:      StmtReturn,
		StartTok:  12,
		EndTok:    14,
		ExprStart: 13,
		ExprEnd:   14,
		BodyStart: -1,
		BodyEnd:   -1,
		ElseStart: -1,
		ElseEnd:   -1,
	}}
	program.Types = []TypeInfo{{
		NameStart: program.Decls[0].NameStart,
		NameEnd:   program.Decls[0].NameEnd,
		Kind:      TypeOther,
		Decl:      0,
		Symbol:    0,
		TypeStart: 5,
		TypeEnd:   6,
		LenStart:  -1,
		LenEnd:    -1,
		KeyStart:  -1,
		KeyEnd:    -1,
		ElemStart: -1,
		ElemEnd:   -1,
	}}
	program.TypeFields = []TypeFields{{
		TypeIndex: 0,
		Fields:    []Field{{NameTok: 3, TypeStart: 5, TypeEnd: 6}},
	}}
	program.TypeIfaces = []TypeIface{{
		TypeIndex: 0,
		Embeds:    []InterfaceEmbed{{TypeStart: 10, TypeEnd: 11}},
		Methods: []InterfaceMethod{{
			NameTok: 13,
			Params:  []Field{{NameTok: -1, TypeStart: 10, TypeEnd: 11}},
			Results: []Field{{NameTok: -1, TypeStart: 10, TypeEnd: 11}},
		}},
	}}
	program.TypeFuncs = []TypeFuncSig{{
		TypeIndex: 0,
		Params:    []Field{{NameTok: -1, TypeStart: 10, TypeEnd: 11}},
		Results:   []Field{{NameTok: -1, TypeStart: 10, TypeEnd: 11}},
	}}
	program.Methods = []MethodInfo{{
		NameTok:   7,
		TypeIndex: 0,
		Symbol:    1,
		FuncIndex: 0,
		Pointer:   true,
	}}
	program.TypeRefs = []TypeRef{{
		OwnerKind:  OwnerDecl,
		OwnerIndex: 0,
		Kind:       TypeRefBuiltin,
		Token:      5,
		BaseTok:    len(program.Tokens) - 1,
		DotTok:     len(program.Tokens) - 1,
		Package:    -1,
		Symbol:     -1,
	}}
	program.Locals = []LocalDecl{{
		FuncIndex:  0,
		Kind:       TokenVar,
		NameStart:  program.Tokens[13].Start,
		NameEnd:    program.Tokens[13].Start + program.Tokens[13].Size,
		Token:      13,
		Scope:      -1,
		ValueIndex: 0,
		TypeStart:  -1,
		TypeEnd:    -1,
		ValueStart: 13,
		ValueEnd:   14,
		Values:     []ExprSpan{{StartTok: 13, EndTok: 14}},
	}}
	program.Indexes = []IndexExpr{{
		OwnerKind:  OwnerFunc,
		OwnerIndex: 0,
		StartTok:   13,
		EndTok:     14,
		BaseStart:  13,
		BaseEnd:    14,
		OpenTok:    13,
		CloseTok:   13,
		IndexStart: 13,
		IndexEnd:   14,
	}}
	program.Composites = []CompositeExpr{{
		OwnerKind:  OwnerDecl,
		OwnerIndex: 0,
		StartTok:   5,
		EndTok:     6,
		TypeStart:  5,
		TypeEnd:    6,
		OpenTok:    5,
		CloseTok:   5,
		Elems:      []ExprSpan{{StartTok: 5, EndTok: 6}},
	}}
	program.Assigns = []Assignment{{
		FuncIndex:  0,
		Kind:       AssignSet,
		StartTok:   13,
		EndTok:     14,
		OpTok:      13,
		LeftStart:  13,
		LeftEnd:    14,
		RightStart: 13,
		RightEnd:   14,
		Targets:    []ExprSpan{{StartTok: 13, EndTok: 14}},
		Values:     []ExprSpan{{StartTok: 13, EndTok: 14}},
	}}
	program.Returns = []Return{{
		FuncIndex: 0,
		StartTok:  12,
		EndTok:    14,
		Values:    []ExprSpan{{StartTok: 13, EndTok: 14}},
	}}
	program.Calls = []Call{{
		OwnerKind:  OwnerFunc,
		OwnerIndex: 0,
		Kind:       CallPackage,
		CalleeTok:  13,
		BaseTok:    len(program.Tokens) - 1,
		DotTok:     len(program.Tokens) - 1,
		ArgsStart:  13,
		ArgsEnd:    14,
		Args:       []ExprSpan{{StartTok: 13, EndTok: 14}},
	}}
	program.Refs = []NameRef{{
		OwnerKind:  OwnerFunc,
		OwnerIndex: 0,
		Kind:       RefPackage,
		Token:      13,
		Index:      0,
		Package:    0,
	}}
	program.Selectors = []Selector{{
		OwnerKind:   OwnerFunc,
		OwnerIndex:  0,
		Kind:        SelectorImport,
		BaseTok:     13,
		DotTok:      13,
		NameTok:     13,
		BaseKind:    RefImport,
		BaseIndex:   0,
		BasePackage: 0,
		Package:     0,
		Symbol:      0,
	}}
	data, ok := Marshal(program)
	if !ok {
		t.Fatal("Marshal failed")
	}
	decoded, ok := Unmarshal(data)
	if !ok {
		t.Fatal("Unmarshal failed")
	}
	if !equalPrograms(decoded, program) {
		t.Fatalf("decoded program = %#v, want %#v", decoded, program)
	}
}

func TestUnmarshalDecodesHostUnitEncoder(t *testing.T) {
	program := declProgram()
	hostProgram := rtgunit.Program{
		Package: program.Package,
		Text:    program.Text,
		Tokens:  hostTokenBytes(program),
		Decls: []rtgunit.Decl{{
			Kind:      program.Decls[0].Kind,
			NameStart: program.Decls[0].NameStart,
			NameEnd:   program.Decls[0].NameEnd,
			StartTok:  program.Decls[0].StartTok,
			EndTok:    program.Decls[0].EndTok,
		}},
		Funcs: []rtgunit.Func{{
			NameStart:     program.Funcs[0].NameStart,
			NameEnd:       program.Funcs[0].NameEnd,
			StartTok:      program.Funcs[0].StartTok,
			NameTok:       program.Funcs[0].NameTok,
			ReceiverStart: program.Funcs[0].ReceiverStart,
			ReceiverEnd:   program.Funcs[0].ReceiverEnd,
			BodyStart:     program.Funcs[0].BodyStart,
			BodyEnd:       program.Funcs[0].BodyEnd,
			EndTok:        program.Funcs[0].EndTok,
		}},
	}
	hostData, err := rtgunit.Marshal(hostProgram)
	if err != nil {
		t.Fatalf("host Marshal failed: %v", err)
	}
	decoded, ok := Unmarshal(hostData)
	if !ok {
		t.Fatal("Unmarshal host data failed")
	}
	if !equalPrograms(decoded, program) {
		t.Fatalf("decoded host program = %#v, want %#v", decoded, program)
	}
}

func TestUnmarshalRejectsMalformedUnit(t *testing.T) {
	if _, ok := Unmarshal(nil); ok {
		t.Fatal("empty input was accepted")
	}
	data, ok := Marshal(helloProgram())
	if !ok {
		t.Fatal("Marshal failed")
	}
	bad := copyBytes(data)
	bad[0] = 'X'
	if _, ok := Unmarshal(bad); ok {
		t.Fatal("bad magic was accepted")
	}
	bad = copyBytes(data)
	bad[4] = byte(Version + 1)
	if _, ok := Unmarshal(bad); ok {
		t.Fatal("bad version was accepted")
	}
	if _, ok := Unmarshal(data[:len(data)-1]); ok {
		t.Fatal("truncated root was accepted")
	}
	bad = copyBytes(data)
	bad[14] = 99
	if _, ok := Unmarshal(bad); ok {
		t.Fatal("unknown child tag was accepted")
	}
}

func TestMarshalRejectsInvalidProgram(t *testing.T) {
	if _, ok := Marshal(Program{}); ok {
		t.Fatal("empty program was accepted")
	}
	program := helloProgram()
	program.Tokens[2].Start = 1
	if _, ok := Marshal(program); ok {
		t.Fatal("backwards token start was accepted")
	}
	program = helloProgram()
	program.Funcs[0].ReceiverStart = -1
	if _, ok := Marshal(program); ok {
		t.Fatal("negative receiver token was accepted")
	}
}

func helloProgram() Program {
	text := []byte("package main\n\nfunc appMain() int { return 0 }\n")
	eof := 11
	return Program{
		Package: "main",
		Text:    text,
		Tokens: []Token{
			{Kind: TokenPackage, Start: 0, Size: 7, Line: 1},
			{Kind: TokenIdent, Start: 8, Size: 4, Line: 1},
			{Kind: TokenFunc, Start: 14, Size: 4, Line: 3},
			{Kind: TokenIdent, Start: 19, Size: 7, Line: 3},
			{Kind: TokenOp, Start: 26, Size: 1, Line: 3},
			{Kind: TokenOp, Start: 27, Size: 1, Line: 3},
			{Kind: TokenIdent, Start: 29, Size: 3, Line: 3},
			{Kind: TokenOp, Start: 33, Size: 1, Line: 3},
			{Kind: TokenReturn, Start: 35, Size: 6, Line: 3},
			{Kind: TokenNumber, Start: 42, Size: 1, Line: 3},
			{Kind: TokenOp, Start: 44, Size: 1, Line: 3},
			{Kind: TokenEOF, Start: 46, Size: 0, Line: 4},
		},
		Funcs: []Func{{
			NameStart:     19,
			NameEnd:       26,
			StartTok:      2,
			NameTok:       3,
			ReceiverStart: eof,
			ReceiverEnd:   eof,
			BodyStart:     7,
			BodyEnd:       10,
			EndTok:        11,
		}},
	}
}

func declProgram() Program {
	text := []byte("package main\n\nconst answer = 42\n\nfunc appMain() int { return answer }\n")
	eof := 15
	return Program{
		Package: "main",
		Text:    text,
		Tokens: []Token{
			{Kind: TokenPackage, Start: 0, Size: 7, Line: 1},
			{Kind: TokenIdent, Start: 8, Size: 4, Line: 1},
			{Kind: TokenConst, Start: 14, Size: 5, Line: 3},
			{Kind: TokenIdent, Start: 20, Size: 6, Line: 3},
			{Kind: TokenOp, Start: 27, Size: 1, Line: 3},
			{Kind: TokenNumber, Start: 29, Size: 2, Line: 3},
			{Kind: TokenFunc, Start: 33, Size: 4, Line: 5},
			{Kind: TokenIdent, Start: 38, Size: 7, Line: 5},
			{Kind: TokenOp, Start: 45, Size: 1, Line: 5},
			{Kind: TokenOp, Start: 46, Size: 1, Line: 5},
			{Kind: TokenIdent, Start: 48, Size: 3, Line: 5},
			{Kind: TokenOp, Start: 52, Size: 1, Line: 5},
			{Kind: TokenReturn, Start: 54, Size: 6, Line: 5},
			{Kind: TokenIdent, Start: 61, Size: 6, Line: 5},
			{Kind: TokenOp, Start: 68, Size: 1, Line: 5},
			{Kind: TokenEOF, Start: 70, Size: 0, Line: 6},
		},
		Decls: []Decl{{
			Kind:      TokenConst,
			NameStart: 20,
			NameEnd:   26,
			StartTok:  2,
			EndTok:    6,
		}},
		Funcs: []Func{{
			NameStart:     38,
			NameEnd:       45,
			StartTok:      6,
			NameTok:       7,
			ReceiverStart: eof,
			ReceiverEnd:   eof,
			BodyStart:     11,
			BodyEnd:       14,
			EndTok:        15,
		}},
	}
}

func hostTokenBytes(program Program) []byte {
	var out []byte
	for i := 0; i < len(program.Tokens); i++ {
		tok := program.Tokens[i]
		var rec [8]byte
		rec[0] = byte(tok.Kind)
		rec[1] = byte(tok.Start)
		rec[2] = byte(tok.Start >> 8)
		rec[3] = byte(tok.Start >> 16)
		rec[4] = byte(tok.Size)
		if tok.Kind == TokenOp && tok.Size > 0 {
			rec[5] = program.Text[tok.Start]
		} else {
			rec[5] = byte(tok.Size >> 8)
		}
		rec[6] = byte(tok.Line)
		rec[7] = byte(tok.Line >> 8)
		out = append(out, rec[:]...)
	}
	return out
}

func equalPrograms(left Program, right Program) bool {
	if left.Package != right.Package || left.ImportPath != right.ImportPath || !bytes.Equal(left.Text, right.Text) {
		return false
	}
	if len(left.Tokens) != len(right.Tokens) || len(left.Imports) != len(right.Imports) ||
		len(left.Symbols) != len(right.Symbols) ||
		len(left.Decls) != len(right.Decls) || len(left.Funcs) != len(right.Funcs) ||
		len(left.DeclMeta) != len(right.DeclMeta) ||
		len(left.InitOrder) != len(right.InitOrder) ||
		len(left.Consts) != len(right.Consts) ||
		len(left.Signatures) != len(right.Signatures) ||
		len(left.Stmts) != len(right.Stmts) ||
		len(left.Types) != len(right.Types) ||
		len(left.TypeFields) != len(right.TypeFields) ||
		len(left.TypeIfaces) != len(right.TypeIfaces) ||
		len(left.TypeFuncs) != len(right.TypeFuncs) ||
		len(left.Methods) != len(right.Methods) ||
		len(left.TypeRefs) != len(right.TypeRefs) ||
		len(left.Locals) != len(right.Locals) ||
		len(left.Indexes) != len(right.Indexes) || len(left.Composites) != len(right.Composites) ||
		len(left.Assigns) != len(right.Assigns) || len(left.Returns) != len(right.Returns) || len(left.Calls) != len(right.Calls) ||
		len(left.Refs) != len(right.Refs) || len(left.Selectors) != len(right.Selectors) {
		return false
	}
	for i := 0; i < len(left.Tokens); i++ {
		if left.Tokens[i] != right.Tokens[i] {
			return false
		}
	}
	for i := 0; i < len(left.Imports); i++ {
		if left.Imports[i] != right.Imports[i] {
			return false
		}
	}
	for i := 0; i < len(left.Symbols); i++ {
		if left.Symbols[i] != right.Symbols[i] {
			return false
		}
	}
	for i := 0; i < len(left.Decls); i++ {
		if left.Decls[i] != right.Decls[i] {
			return false
		}
	}
	for i := 0; i < len(left.Funcs); i++ {
		if left.Funcs[i] != right.Funcs[i] {
			return false
		}
	}
	for i := 0; i < len(left.DeclMeta); i++ {
		if left.DeclMeta[i].DeclIndex != right.DeclMeta[i].DeclIndex ||
			left.DeclMeta[i].Symbol != right.DeclMeta[i].Symbol ||
			left.DeclMeta[i].ValueIndex != right.DeclMeta[i].ValueIndex ||
			left.DeclMeta[i].TypeStart != right.DeclMeta[i].TypeStart ||
			left.DeclMeta[i].TypeEnd != right.DeclMeta[i].TypeEnd ||
			left.DeclMeta[i].ValueStart != right.DeclMeta[i].ValueStart ||
			left.DeclMeta[i].ValueEnd != right.DeclMeta[i].ValueEnd ||
			left.DeclMeta[i].Alias != right.DeclMeta[i].Alias ||
			len(left.DeclMeta[i].Values) != len(right.DeclMeta[i].Values) {
			return false
		}
		for j := 0; j < len(left.DeclMeta[i].Values); j++ {
			if left.DeclMeta[i].Values[j] != right.DeclMeta[i].Values[j] {
				return false
			}
		}
	}
	for i := 0; i < len(left.InitOrder); i++ {
		if left.InitOrder[i] != right.InitOrder[i] {
			return false
		}
	}
	for i := 0; i < len(left.Consts); i++ {
		if left.Consts[i] != right.Consts[i] {
			return false
		}
	}
	for i := 0; i < len(left.Signatures); i++ {
		if !equalSignature(left.Signatures[i], right.Signatures[i]) {
			return false
		}
	}
	for i := 0; i < len(left.Stmts); i++ {
		if left.Stmts[i] != right.Stmts[i] {
			return false
		}
	}
	for i := 0; i < len(left.Types); i++ {
		if left.Types[i] != right.Types[i] {
			return false
		}
	}
	for i := 0; i < len(left.TypeFields); i++ {
		if left.TypeFields[i].TypeIndex != right.TypeFields[i].TypeIndex ||
			len(left.TypeFields[i].Fields) != len(right.TypeFields[i].Fields) {
			return false
		}
		for j := 0; j < len(left.TypeFields[i].Fields); j++ {
			if left.TypeFields[i].Fields[j] != right.TypeFields[i].Fields[j] {
				return false
			}
		}
	}
	for i := 0; i < len(left.TypeIfaces); i++ {
		if !equalTypeInterface(left.TypeIfaces[i], right.TypeIfaces[i]) {
			return false
		}
	}
	for i := 0; i < len(left.TypeFuncs); i++ {
		if !equalTypeFunc(left.TypeFuncs[i], right.TypeFuncs[i]) {
			return false
		}
	}
	for i := 0; i < len(left.Methods); i++ {
		if left.Methods[i] != right.Methods[i] {
			return false
		}
	}
	for i := 0; i < len(left.TypeRefs); i++ {
		if left.TypeRefs[i] != right.TypeRefs[i] {
			return false
		}
	}
	for i := 0; i < len(left.Locals); i++ {
		if left.Locals[i].FuncIndex != right.Locals[i].FuncIndex ||
			left.Locals[i].Kind != right.Locals[i].Kind ||
			left.Locals[i].NameStart != right.Locals[i].NameStart ||
			left.Locals[i].NameEnd != right.Locals[i].NameEnd ||
			left.Locals[i].Token != right.Locals[i].Token ||
			left.Locals[i].Scope != right.Locals[i].Scope ||
			left.Locals[i].ValueIndex != right.Locals[i].ValueIndex ||
			left.Locals[i].TypeStart != right.Locals[i].TypeStart ||
			left.Locals[i].TypeEnd != right.Locals[i].TypeEnd ||
			left.Locals[i].ValueStart != right.Locals[i].ValueStart ||
			left.Locals[i].ValueEnd != right.Locals[i].ValueEnd ||
			left.Locals[i].Alias != right.Locals[i].Alias ||
			len(left.Locals[i].Values) != len(right.Locals[i].Values) {
			return false
		}
		for j := 0; j < len(left.Locals[i].Values); j++ {
			if left.Locals[i].Values[j] != right.Locals[i].Values[j] {
				return false
			}
		}
	}
	for i := 0; i < len(left.Indexes); i++ {
		if left.Indexes[i] != right.Indexes[i] {
			return false
		}
	}
	for i := 0; i < len(left.Composites); i++ {
		if left.Composites[i].OwnerKind != right.Composites[i].OwnerKind ||
			left.Composites[i].OwnerIndex != right.Composites[i].OwnerIndex ||
			left.Composites[i].StartTok != right.Composites[i].StartTok ||
			left.Composites[i].EndTok != right.Composites[i].EndTok ||
			left.Composites[i].TypeStart != right.Composites[i].TypeStart ||
			left.Composites[i].TypeEnd != right.Composites[i].TypeEnd ||
			left.Composites[i].OpenTok != right.Composites[i].OpenTok ||
			left.Composites[i].CloseTok != right.Composites[i].CloseTok ||
			len(left.Composites[i].Elems) != len(right.Composites[i].Elems) {
			return false
		}
		for j := 0; j < len(left.Composites[i].Elems); j++ {
			if left.Composites[i].Elems[j] != right.Composites[i].Elems[j] {
				return false
			}
		}
	}
	for i := 0; i < len(left.Assigns); i++ {
		if left.Assigns[i].FuncIndex != right.Assigns[i].FuncIndex ||
			left.Assigns[i].Kind != right.Assigns[i].Kind ||
			left.Assigns[i].StartTok != right.Assigns[i].StartTok ||
			left.Assigns[i].EndTok != right.Assigns[i].EndTok ||
			left.Assigns[i].OpTok != right.Assigns[i].OpTok ||
			left.Assigns[i].LeftStart != right.Assigns[i].LeftStart ||
			left.Assigns[i].LeftEnd != right.Assigns[i].LeftEnd ||
			left.Assigns[i].RightStart != right.Assigns[i].RightStart ||
			left.Assigns[i].RightEnd != right.Assigns[i].RightEnd ||
			len(left.Assigns[i].Targets) != len(right.Assigns[i].Targets) ||
			len(left.Assigns[i].Values) != len(right.Assigns[i].Values) {
			return false
		}
		for j := 0; j < len(left.Assigns[i].Targets); j++ {
			if left.Assigns[i].Targets[j] != right.Assigns[i].Targets[j] {
				return false
			}
		}
		for j := 0; j < len(left.Assigns[i].Values); j++ {
			if left.Assigns[i].Values[j] != right.Assigns[i].Values[j] {
				return false
			}
		}
	}
	for i := 0; i < len(left.Returns); i++ {
		if left.Returns[i].FuncIndex != right.Returns[i].FuncIndex ||
			left.Returns[i].StartTok != right.Returns[i].StartTok ||
			left.Returns[i].EndTok != right.Returns[i].EndTok ||
			len(left.Returns[i].Values) != len(right.Returns[i].Values) {
			return false
		}
		for j := 0; j < len(left.Returns[i].Values); j++ {
			if left.Returns[i].Values[j] != right.Returns[i].Values[j] {
				return false
			}
		}
	}
	for i := 0; i < len(left.Calls); i++ {
		if left.Calls[i].OwnerKind != right.Calls[i].OwnerKind ||
			left.Calls[i].OwnerIndex != right.Calls[i].OwnerIndex ||
			left.Calls[i].Kind != right.Calls[i].Kind ||
			left.Calls[i].CalleeTok != right.Calls[i].CalleeTok ||
			left.Calls[i].BaseTok != right.Calls[i].BaseTok ||
			left.Calls[i].DotTok != right.Calls[i].DotTok ||
			left.Calls[i].ArgsStart != right.Calls[i].ArgsStart ||
			left.Calls[i].ArgsEnd != right.Calls[i].ArgsEnd ||
			len(left.Calls[i].Args) != len(right.Calls[i].Args) {
			return false
		}
		for j := 0; j < len(left.Calls[i].Args); j++ {
			if left.Calls[i].Args[j] != right.Calls[i].Args[j] {
				return false
			}
		}
	}
	for i := 0; i < len(left.Refs); i++ {
		if left.Refs[i] != right.Refs[i] {
			return false
		}
	}
	for i := 0; i < len(left.Selectors); i++ {
		if left.Selectors[i] != right.Selectors[i] {
			return false
		}
	}
	return true
}

func equalSignature(left FuncSignature, right FuncSignature) bool {
	if left.FuncIndex != right.FuncIndex ||
		len(left.Receiver) != len(right.Receiver) ||
		len(left.Params) != len(right.Params) ||
		len(left.Results) != len(right.Results) {
		return false
	}
	for i := 0; i < len(left.Receiver); i++ {
		if left.Receiver[i] != right.Receiver[i] {
			return false
		}
	}
	for i := 0; i < len(left.Params); i++ {
		if left.Params[i] != right.Params[i] {
			return false
		}
	}
	for i := 0; i < len(left.Results); i++ {
		if left.Results[i] != right.Results[i] {
			return false
		}
	}
	return true
}

func equalTypeInterface(left TypeIface, right TypeIface) bool {
	if left.TypeIndex != right.TypeIndex ||
		len(left.Embeds) != len(right.Embeds) ||
		len(left.Methods) != len(right.Methods) {
		return false
	}
	for i := 0; i < len(left.Embeds); i++ {
		if left.Embeds[i] != right.Embeds[i] {
			return false
		}
	}
	for i := 0; i < len(left.Methods); i++ {
		if left.Methods[i].NameTok != right.Methods[i].NameTok ||
			len(left.Methods[i].Params) != len(right.Methods[i].Params) ||
			len(left.Methods[i].Results) != len(right.Methods[i].Results) {
			return false
		}
		for j := 0; j < len(left.Methods[i].Params); j++ {
			if left.Methods[i].Params[j] != right.Methods[i].Params[j] {
				return false
			}
		}
		for j := 0; j < len(left.Methods[i].Results); j++ {
			if left.Methods[i].Results[j] != right.Methods[i].Results[j] {
				return false
			}
		}
	}
	return true
}

func equalTypeFunc(left TypeFuncSig, right TypeFuncSig) bool {
	if left.TypeIndex != right.TypeIndex ||
		len(left.Params) != len(right.Params) ||
		len(left.Results) != len(right.Results) {
		return false
	}
	for i := 0; i < len(left.Params); i++ {
		if left.Params[i] != right.Params[i] {
			return false
		}
	}
	for i := 0; i < len(left.Results); i++ {
		if left.Results[i] != right.Results[i] {
			return false
		}
	}
	return true
}

func copyBytes(data []byte) []byte {
	out := make([]byte, len(data))
	copy(out, data)
	return out
}
