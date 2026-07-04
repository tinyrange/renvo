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
