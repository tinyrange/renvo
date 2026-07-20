package pipeline

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestSessionYieldsAndMatchesSynchronousPipeline(t *testing.T) {
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n\nimport \"example.com/case/pkg/lib\"\n\nfunc appMain() int { return lib.Value() }\n")},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte("package lib\n\nfunc Value() int { return 42 }\n")},
	}
	want := BuildUnit("/repo/case", "/std", "./cmd/app", files)
	if !want.Ok {
		t.Fatalf("synchronous pipeline failed: %#v", want)
	}
	session := BeginSession("/repo/case", "/std", "./cmd/app", files, 0, 0, false, true)
	steps := 0
	for !session.Step() {
		steps++
	}
	steps++
	got := session.Result()
	if !got.Ok {
		t.Fatalf("resumable pipeline failed: %#v", got)
	}
	if !bytes.Equal(got.Link.Data, want.Link.Data) {
		t.Fatal("resumable incremental pipeline changed the linked backend unit")
	}
	if steps < 8 {
		t.Fatalf("pipeline completed in %d steps; expected phase and per-package yields", steps)
	}
}
