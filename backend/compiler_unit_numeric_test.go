package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"renvo.dev/backend/unit"
)

func TestUnitLeadingDecimalPointKeepsSourceMetadata(t *testing.T) {
	for _, lines := range []int{1, 65537} {
		t.Run(strings.Repeat("extended_", lines/65536)+"lines", func(t *testing.T) {
			resetRuntime()
			source := "package main\n" + strings.Repeat("\n", lines) + `
var leading = .5
type Value float64
func (v Value) Number()float64{return float64(v)+.5}
func first()float64{return .5e2}
func appMain()int{if first()!=50{return 1};return 0}
`
			path := filepath.Join(t.TempDir(), "input.go")
			if err := os.WriteFile(path, []byte(source), 0644); err != nil {
				t.Fatal(err)
			}
			wire, err := unit.ConvertFiles([]string{path})
			if err != nil {
				t.Fatal(err)
			}
			wire.Packages = []unit.PackageInfo{{
				Name: "main", ImportPath: "example.com/case",
				TextEnd: len(wire.Text), TokenEnd: len(wire.Tokens) / 8,
				DeclEnd: len(wire.Decls), FuncEnd: len(wire.Funcs),
			}}
			data, err := unit.Marshal(wire)
			if err != nil {
				t.Fatal(err)
			}
			decoded, isUnit, ok := renvoDecodeUnitProgram(data)
			if !isUnit || !ok {
				t.Fatal("decode")
			}
			parsed := renvoParseProgram(wire.Text)
			if !parsed.ok || decoded.toks.count != parsed.toks.count || len(decoded.decls) != len(parsed.decls) || len(decoded.funcs) != len(parsed.funcs) {
				t.Fatal("unit and source metadata differ")
			}
			// The wire converter prepends a newline to its compact text while
			// retaining the original file's line numbers.
			lineOffset := renvoTokLine(&parsed, 0) - renvoTokLine(&decoded, 0)
			for token := 0; token < parsed.toks.count; token++ {
				if renvoTokAt(&decoded, token) != renvoTokAt(&parsed, token) || int(decoded.toks.data[token*renvoTokenStride])&255 != int(parsed.toks.data[token*renvoTokenStride])&255 || token+1 < parsed.toks.count && renvoTokLine(&decoded, token)+lineOffset != renvoTokLine(&parsed, token) {
					t.Fatalf("token %d differs: unit=%+v line=%d source=%+v line=%d", token, renvoTokAt(&decoded, token), renvoTokLine(&decoded, token), renvoTokAt(&parsed, token), renvoTokLine(&parsed, token))
				}
			}
			last := len(wire.Tokens) - 8
			eofLine := int(wire.Tokens[last+6]) | int(wire.Tokens[last+7])<<8
			if len(wire.TokenLines) != 0 {
				eofLine = wire.TokenLines[len(wire.TokenLines)-1]
			}
			if renvoTokLine(&decoded, decoded.toks.count-1) != eofLine {
				t.Fatal("original EOF line was not retained")
			}
			for index, decl := range parsed.decls {
				if decoded.decls[index] != decl {
					t.Fatalf("declaration %d differs: unit=%+v source=%+v", index, decoded.decls[index], decl)
				}
			}
			for index, fn := range parsed.funcs {
				got := decoded.funcs[index]
				if got.startTok != fn.startTok || got.endTok != fn.endTok || got.nameTok != fn.nameTok || got.bodyStart != fn.bodyStart || got.bodyEnd != fn.bodyEnd || got.receiverStart != fn.receiverStart || got.receiverEnd != fn.receiverEnd {
					t.Fatalf("function %d differs: unit=%+v source=%+v", index, got, fn)
				}
			}
			if len(decoded.packageTable.items) != 1 || decoded.packageTable.items[0].tokenEnd != parsed.toks.count {
				t.Fatal("package token range differs")
			}
		})
	}
}
