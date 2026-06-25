package check

import (
	"strings"
	"testing"

	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/parse"
)

func TestFileRejectsExcludedFeatures(t *testing.T) {
	src := []byte(`package main

type Box[T any] struct { value T }
type Runner interface { Run() }
type fixed [4]int
type holder struct { values [2]int }
func takesArray(values [3]int) int { return 0 }
func useMap(m map[string]int) int { return 0 }
func makeChan(ch chan int) { go makeChan(ch); select {} }
func appMain() int {
	defer print("done")
	fn := func() int { return 1 }
	_ = fn
	for _, v := range []int{1, 2} {
		_ = v
	}
	return 0
}
`)
	file, err := parse.FileSource("bad.go", src)
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	messages := strings.Join(messages(diags), "\n")
	for _, want := range []string{
		"generics are not supported",
		"interfaces are not supported",
		"maps are not supported",
		"arrays are not supported",
		"channels are not supported",
		"goroutines are not supported",
		"select statements are not supported",
		"range is not supported",
		"defer is not supported",
		"function values and function types are not supported",
	} {
		if !strings.Contains(messages, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, messages)
		}
	}
}

func TestFileAcceptsSimpleSubsetProgram(t *testing.T) {
	file, err := parse.FileSource("ok.go", []byte(`package main

type box struct { value int }
func appMain() int {
	var b box
	b.value = 7
	return b.value
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestGraphRejectsDuplicatePackageLevelNames(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app/pkg",
				Name:       "pkg",
				Files: []load.File{
					{
						Path: "a.go",
						Source: []byte(`package pkg

func Value() int { return 1 }
`),
					},
					{
						Path: "b.go",
						Source: []byte(`package pkg

func Value() int { return 2 }
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph succeeded with duplicate declaration")
	}
	msg := err.Error()
	if !strings.Contains(msg, "a.go:3:6: duplicate package-level declaration: Value") {
		t.Fatalf("missing first duplicate diagnostic in:\n%s", msg)
	}
	if !strings.Contains(msg, "b.go:3:6: duplicate package-level declaration: Value") {
		t.Fatalf("missing second duplicate diagnostic in:\n%s", msg)
	}
}

func messages(diags Diagnostics) []string {
	var out []string
	for _, diag := range diags {
		out = append(out, diag.Message)
	}
	return out
}
