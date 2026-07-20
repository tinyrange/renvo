package unit

import (
	"reflect"
	"testing"
)

func TestFrontendCacheRoundTripPreservesLinkMetadata(t *testing.T) {
	program := Program{
		Package: "sample", ImportPath: "example.com/sample", Text: []byte("package sample\n"),
		Tokens:    []Token{MakeToken(TokenPackage, 0, 7, 1), MakeToken(TokenEOF, 15, 0, 2)},
		Imports:   []Import{{NameTok: -1, PathTok: 2}},
		Symbols:   []Symbol{{Name: "Value", Package: 3, Token: 4}},
		Decls:     []Decl{{Kind: TokenType, NameStart: 8, NameEnd: 14, StartTok: 1, EndTok: 5}},
		Funcs:     []Func{{NameStart: 8, NameEnd: 14, StartTok: 1, NameTok: 2, ReceiverStart: -1, ReceiverEnd: -1, BodyStart: 4, BodyEnd: 5, EndTok: 6}},
		TypeRefs:  []TypeRef{{Kind: TypeRefPackage, Token: 2, BaseTok: -1, DotTok: -1, Package: 3, Symbol: 4}},
		Calls:     []Call{{Kind: CallPackage, CalleeTok: 2, BaseTok: -1, DotTok: -1}},
		Refs:      []NameRef{{Kind: RefPackage, Token: 2, Index: 4, Package: 3}},
		Selectors: []Selector{{BaseTok: 1, DotTok: 2, NameTok: 3, BaseKind: RefImport, BaseIndex: 0, BasePackage: 2, Package: 3, Symbol: 4}},
	}
	data, ok := MarshalFrontendCache(program)
	if !ok {
		t.Fatal("MarshalFrontendCache failed")
	}
	decoded, ok := UnmarshalFrontendCache(data)
	if !ok || !reflect.DeepEqual(decoded, program) {
		t.Fatalf("round trip = %#v, %v; want %#v", decoded, ok, program)
	}
}

func TestFrontendCacheRejectsTruncatedData(t *testing.T) {
	data, _ := MarshalFrontendCache(Program{Package: "p", Text: []byte("package p")})
	for end := 0; end < len(data); end++ {
		if _, ok := UnmarshalFrontendCache(data[:end]); ok {
			t.Fatalf("accepted prefix of length %d", end)
		}
	}
}
