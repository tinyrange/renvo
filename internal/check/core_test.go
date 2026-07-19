//go:build renvo

package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestCheckGraphCoreImportUsage(t *testing.T) {
	files := []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func main() { lib.Value() }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte("package lib\n\nfunc Value() int { return 1 }\n")},
	}
	prog := CheckGraphCore(testGraph(t, files))
	if !prog.Ok {
		t.Fatalf("used import rejected: %#v", prog)
	}
	root := prog.Packages[len(prog.Packages)-1]
	if len(root.CoreBodies) != 1 || len(root.CoreBodies[0].CoreSelectors) != 1 {
		t.Fatalf("import selector metadata = %#v", root.CoreBodies)
	}

	files[0].Src = []byte("package main\n\nimport \"example.com/case/pkg/lib\"\n\nfunc main() {}\n")
	prog = CheckGraphCore(testGraph(t, files))
	if prog.Ok || prog.Error != CheckErrUnusedImport {
		t.Fatalf("unused import accepted: %#v", prog)
	}
}

func TestCheckGraphCoreResolvesNamesAfterLocalDeclaration(t *testing.T) {
	files := []load.SourceFile{{
		Path: "/repo/case/cmd/app/main.go",
		Src: []byte(`package main

var packageValue = 7

func use(value int) int { return value }

func main() {
	var local int
	local = use(packageValue)
	_ = local
}
`),
	}}
	prog := CheckGraphCore(testGraph(t, files))
	if !prog.Ok {
		t.Fatalf("program rejected: %#v", prog)
	}
	root := prog.Packages[len(prog.Packages)-1]
	if len(root.CoreBodies) != 2 {
		t.Fatalf("bodies = %#v", root.CoreBodies)
	}
	body := root.CoreBodies[1]
	if len(body.CoreRefs) != 2 {
		t.Fatalf("package references after local declaration = %#v", body.CoreRefs)
	}
	for i := 0; i < len(body.CoreRefs); i++ {
		name := root.Symbols[body.CoreRefs[i].Index].Name
		if name != "packageValue" && name != "use" {
			t.Fatalf("reference %d resolved to %q", i, name)
		}
	}
}
