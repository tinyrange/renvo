package pipeline

import (
	"bytes"
	"testing"

	"renvo.dev/internal/link"
	"renvo.dev/internal/load"
)

func TestCachedGoObjectSessionMatchesOneShot(t *testing.T) {
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.22\n")},
		{Path: "/repo/case/main.go", Src: []byte("package main\nimport \"example.com/case/lib\"\nfunc Add(a,b int)int{return lib.Sum(a,b)}\nfunc main(){_=Add(1,2)}\n")},
		{Path: "/repo/case/lib/lib.go", Src: []byte("package lib\nfunc Sum(a,b int)int{return a+b}\n")},
	}
	want := BuildObjectUnit("/repo/case", "/std", ".", files)
	if !want.Ok {
		t.Fatal("one-shot build failed")
	}
	for i := 0; i < 2; i++ {
		session := BeginObjectSession("/repo/case", "/std", ".", files, true)
		for !session.Step() {
		}
		got := session.Result()
		if !got.Ok {
			t.Fatalf("session failed: pipeline=%d build=%d", got.Error, got.Build.Error)
		}
		// Cached builds intentionally carry graph/source fingerprints absent from
		// BuildObjectUnit. Compare full serialization using the same built units.
		whole := link.LinkBuildObjectCore(got.Build)
		if !whole.Ok || !bytes.Equal(got.Link.Data, whole.Data) || !bytes.Equal(got.Link.Program.Text, want.Link.Program.Text) {
			t.Fatalf("cached object differs\none-shot:\n%s\nsession:\n%s", want.Link.Program.Text, got.Link.Program.Text)
		}
		if bytes.Contains(got.Link.Program.Text, []byte("//export Sum\n")) {
			t.Fatal("dependency function became an automatic object root")
		}
	}
}
