package pipeline

import (
	"bytes"
	"reflect"
	"testing"

	wireunit "renvo.dev/backend/unit"
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
	if !linkedUnitsEqualIgnoringCacheKeys(got.Link.Data, want.Link.Data) {
		t.Fatal("resumable incremental pipeline changed linked program semantics")
	}
	if steps < 8 {
		t.Fatalf("pipeline completed in %d steps; expected phase and per-package yields", steps)
	}
}

func linkedUnitsEqualIgnoringCacheKeys(left []byte, right []byte) bool {
	if bytes.Equal(left, right) {
		return true
	}
	leftProgram, leftErr := wireunit.Unmarshal(left)
	rightProgram, rightErr := wireunit.Unmarshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	for i := 0; i < len(leftProgram.Packages); i++ {
		leftProgram.Packages[i].GraphKeyA = 0
		leftProgram.Packages[i].GraphKeyB = 0
		leftProgram.Packages[i].SourceKeyA = 0
		leftProgram.Packages[i].SourceKeyB = 0
	}
	for i := 0; i < len(rightProgram.Packages); i++ {
		rightProgram.Packages[i].GraphKeyA = 0
		rightProgram.Packages[i].GraphKeyB = 0
		rightProgram.Packages[i].SourceKeyA = 0
		rightProgram.Packages[i].SourceKeyB = 0
	}
	return reflect.DeepEqual(leftProgram, rightProgram)
}
