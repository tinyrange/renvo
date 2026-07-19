package unit

import (
	"bytes"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	wireunit "renvo.dev/backend/unit"
)

func coreGoldenProgram() Program {
	return Program{
		Package:    "p",
		ImportPath: "example/p",
		Text:       []byte("package p\n"),
		Tokens: []Token{
			{Kind: TokenPackage, Start: 0, Size: 7, Line: 1},
			{Kind: TokenIdent, Start: 8, Size: 1, Line: 1},
			{Kind: TokenEOF, Start: 10, Size: 0, Line: 2},
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
