package unit

import (
	"bytes"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"unsafe"

	wireunit "renvo.dev/backend/unit"
)

func TestTokenCompactLayout(t *testing.T) {
	tok := MakeToken(TokenIdent, 12, 6, 345)
	if tok.KindLine&255 != TokenIdent || tok.KindLine>>8 != 345 || tok.Start != 12 || tok.Size != 6 {
		t.Fatalf("token fields were not preserved: %#v", tok)
	}
	if got, want := unsafe.Sizeof(tok), 3*unsafe.Sizeof(int(0)); got != want {
		t.Fatalf("token size = %d, want %d", got, want)
	}
}

func coreGoldenProgram() Program {
	return Program{
		Package:    "p",
		ImportPath: "example/p",
		Text:       []byte("package p\n"),
		Tokens: []Token{
			MakeToken(TokenPackage, 0, 7, 1),
			MakeToken(TokenIdent, 8, 1, 1),
			MakeToken(TokenEOF, 10, 0, 2),
		},
	}
}

func TestCoreGoldenVector(t *testing.T) {
	data, ok := Marshal(coreGoldenProgram())
	if !ok {
		t.Fatal("Marshal failed")
	}
	want := readGoldenHex(t, "../../backend/unit/testdata/v1-core.hex")
	if !bytes.Equal(data, want) {
		t.Fatalf("shared encoder drift\ngot  %x\nwant %x", data, want)
	}
}

func TestCoreUnitDecodesWithPublicReader(t *testing.T) {
	program := coreGoldenProgram()
	data, ok := MarshalCore(CoreProgramFrom(program))
	if !ok {
		t.Fatal("MarshalCore failed")
	}
	decoded, err := wireunit.Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Package != program.Package || decoded.ImportPath != program.ImportPath || !bytes.Equal(decoded.Text, program.Text) {
		t.Fatalf("decoded unit = %#v", decoded)
	}
	if len(decoded.Decls) != len(program.Decls) || len(decoded.Funcs) != len(program.Funcs) {
		t.Fatalf("decoded declarations/functions = %d/%d", len(decoded.Decls), len(decoded.Funcs))
	}
}

func TestTransientCoreMarshalPreservesCanonicalEncoding(t *testing.T) {
	program := coreGoldenProgram()
	program.Text = bytes.Repeat([]byte("package p\n"), transientMarshalChunk)
	program.Tokens = make([]Token, transientMarshalChunk+17)
	for i := 0; i < len(program.Tokens); i++ {
		program.Tokens[i] = MakeToken(TokenIdent, i, 1, i/10+1)
	}
	want, ok := MarshalCore(CoreProgramFrom(program))
	if !ok {
		t.Fatal("MarshalCore failed")
	}
	got, ok := MarshalCoreTransient(CoreProgramFrom(program))
	if !ok {
		t.Fatal("MarshalCoreTransient failed")
	}
	if !bytes.Equal(got, want) {
		t.Fatal("transient marshal changed the canonical unit encoding")
	}
}

func readGoldenHex(t *testing.T, path string) []byte {
	t.Helper()
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data, err := hex.DecodeString(strings.TrimSpace(string(encoded)))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
