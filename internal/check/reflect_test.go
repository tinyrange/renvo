package check

import (
	"renvo.dev/internal/syntax"
	"testing"
)

func TestReflectedStructKeepsTagsWithoutOptingInOtherTypes(t *testing.T) {
	file := syntax.ParseFile([]byte("package p\n//renvo:reflect\ntype Record struct { A, B int `json:\"value,omitempty\"`; hidden string `json:\"-\"` }\ntype Plain struct { Record Record }\n"))
	if !file.Ok {
		t.Fatal("parse fixture")
	}
	for i, decl := range file.Decls {
		info := buildTypeInfo(file, DeclInfo{Token: decl.NameTok, TypeStart: decl.NameTok + 1, TypeEnd: decl.EndTok}, i)
		if info.Reflect != (i == 0) {
			t.Fatal("unexpected reflection opt-in")
		}
		if i == 0 && (len(info.Fields) != 3 || info.Fields[0].Tag != "json:\"value,omitempty\"" || info.Fields[1].Tag != info.Fields[0].Tag || info.Fields[2].Tag != "json:\"-\"") {
			t.Fatalf("lost struct tags: %+v", info.Fields)
		}
	}
}
