package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestFunctionValueImportedDirectFunctionCallArgument(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/callback/callback.go", Src: []byte(`package callback

type SplitFunc func([]byte, bool) (int, []byte, error)
type Scanner struct { split SplitFunc }
func (s *Scanner) Split(split SplitFunc) { s.split = split }
func ScanWords(data []byte, atEOF bool) (int, []byte, error) { return len(data), data, nil }
`)},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/callback"

func main() {
	scanner := &callback.Scanner{}
	scanner.Split(callback.ScanWords)
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if !bytes.Contains(linked.Program.Text, []byte("scanner.Split(SplitFunc{kind: 1})")) ||
		!bytes.Contains(linked.Program.Text, []byte("ScanWords(arg0, arg1)")) ||
		!bytes.Contains(linked.Program.Text, []byte("return __renvo_zero_0, __renvo_zero_1, __renvo_zero_2")) {
		t.Fatalf("imported direct function argument was not lowered:\n%s", linked.Program.Text)
	}
}

func TestFunctionValueDirectFunctionCompositeField(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/callback/callback.go", Src: []byte(`package callback

type SplitFunc func([]byte, bool) (int, []byte, error)
type Scanner struct { split SplitFunc }
func ScanLines(data []byte, atEOF bool) (int, []byte, error) { return len(data), data, nil }
func NewScanner() *Scanner { return &Scanner{split: ScanLines} }
`)},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/callback"

func main() { callback.NewScanner() }
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("LinkBuildCore failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if !bytes.Contains(linked.Program.Text, []byte("&Scanner{split: SplitFunc{kind: 1}}")) ||
		!bytes.Contains(linked.Program.Text, []byte("ScanLines(arg0, arg1)")) {
		t.Fatalf("direct function composite field was not lowered:\n%s", linked.Program.Text)
	}
}
