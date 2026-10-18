package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestPackageBlankDeclarationsHaveNoBinding(t *testing.T) {
	graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
var _ = 1
var _ = "other type"
const _ = true
const _ = 2
func main() {}
`)}})
	checked := CheckGraphCore(graph)
	if !checked.Ok {
		t.Fatalf("blank declarations rejected: %#v", checked)
	}
	for _, symbol := range checked.Packages[checkRootPackage(t, graph)].Symbols {
		if symbol.Name == "_" {
			t.Fatal("blank identifier introduced a package binding")
		}
	}
}
