package check

import (
	"testing"

	"renvo.dev/internal/syntax"
)

func TestChannelTypeShape(t *testing.T) {
	cases := []struct {
		source    string
		direction int
		element   string
	}{
		{"chan int", ChanBoth, "int"},
		{"chan<- string", ChanSendOnly, "string"},
		{"<-chan []byte", ChanReceiveOnly, "[]byte"},
	}
	for _, test := range cases {
		source := []byte("package sample\ntype C " + test.source + "\n")
		file := syntax.ParseFile(source)
		if !file.Ok || len(file.Decls) != 1 {
			t.Fatalf("parse %q: ok=%v decls=%d err=%d", test.source, file.Ok, len(file.Decls), file.Error)
		}
		decl := buildDeclInfo(file, 0, PackageInfo{}, nil, file.Decls[0])
		typ := buildTypeInfo(file, decl, 0)
		if typ.Kind != TypeChan || typ.Direction != test.direction {
			t.Fatalf("shape %q: kind=%d direction=%d", test.source, typ.Kind, typ.Direction)
		}
		if got := tokenSpanString(file, typ.ElemStart, typ.ElemEnd); got != test.element {
			t.Fatalf("element %q: got %q, want %q", test.source, got, test.element)
		}
	}
}

func tokenSpanString(file syntax.File, start int, end int) string {
	if start < 0 || end <= start {
		return ""
	}
	first := file.Tokens[start]
	last := file.Tokens[end-1]
	return string(file.Src[first.Start:last.End])
}
