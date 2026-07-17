package check

import (
	"testing"

	"j5.nz/rtg/rtg/internal/load"
)

func TestCheckGraphUsesSharedCore(t *testing.T) {
	graph := checkTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/lib"

const answer = 42
func appMain() int { return answer + lib.Value() }
`)},
		{Path: "/repo/case/lib/lib.go", Src: []byte("package lib\nfunc Value() int { return 1 }\n")},
	})
	viaCompatibilityName := CheckGraph(graph)
	viaCoreName := CheckGraphCore(graph)
	if !viaCompatibilityName.Ok || !viaCoreName.Ok {
		t.Fatalf("check failed: compatibility=%#v core=%#v", viaCompatibilityName, viaCoreName)
	}
	if len(viaCompatibilityName.Packages) != len(viaCoreName.Packages) {
		t.Fatalf("package counts differ: %d/%d", len(viaCompatibilityName.Packages), len(viaCoreName.Packages))
	}
	root := checkRootPackage(t, graph)
	got := viaCompatibilityName.Packages[root]
	want := viaCoreName.Packages[root]
	if len(got.Symbols) != len(want.Symbols) || len(got.Decls) != len(want.Decls) || len(got.CoreBodies) != len(want.CoreBodies) {
		t.Fatalf("shared checker models differ: compatibility=%#v core=%#v", got, want)
	}
	if len(got.Imports) != 1 || !got.Imports[0].Used {
		t.Fatalf("import usage = %#v", got.Imports)
	}
}

func TestCheckGraphCoreDiagnostics(t *testing.T) {
	cases := []struct {
		name  string
		files []load.SourceFile
		err   int
	}{
		{
			name: "duplicate symbol",
			files: []load.SourceFile{
				{Path: "/repo/case/cmd/app/a.go", Src: []byte("package main\nvar value int\n")},
				{Path: "/repo/case/cmd/app/b.go", Src: []byte("package main\nfunc value() {}\n")},
			},
			err: CheckErrDuplicate,
		},
		{
			name:  "duplicate parameter",
			files: []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc value(x int, x int) {}\n")}},
			err:   CheckErrScope,
		},
		{
			name:  "return count",
			files: []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc value() int { return }\n")}},
			err:   CheckErrReturnCount,
		},
		{
			name:  "assignment type",
			files: []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc main() { var value bool; value = 1; _ = value }\n")}},
			err:   CheckErrType,
		},
		{
			name:  "excluded goroutine",
			files: []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc work() {}\nfunc main() { go work() }\n")}},
			err:   CheckErrExcluded,
		},
		{
			name: "unused import",
			files: []load.SourceFile{
				{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nimport \"example.com/case/lib\"\nfunc main() {}\n")},
				{Path: "/repo/case/lib/lib.go", Src: []byte("package lib\n")},
			},
			err: CheckErrUnusedImport,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := CheckGraphCore(checkTestGraph(t, tc.files))
			if program.Ok || program.Error != tc.err {
				t.Fatalf("check result = %#v, want error %d", program, tc.err)
			}
			if program.ErrorPackage < 0 || program.ErrorFile < 0 || program.ErrorToken < 0 {
				t.Fatalf("diagnostic has no source location: %#v", program)
			}
		})
	}
}

func TestCheckGraphCoreAllowsMultipleInitFunctions(t *testing.T) {
	graph := checkTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/a.go", Src: []byte("package main\nfunc init() {}\n")},
		{Path: "/repo/case/cmd/app/b.go", Src: []byte("package main\nfunc init() {}\nfunc main() {}\n")},
	})
	program := CheckGraphCore(graph)
	if !program.Ok {
		t.Fatalf("multiple init functions were rejected: %#v", program)
	}
	if len(program.Packages[0].CoreBodies) != 3 {
		t.Fatalf("bodies = %#v", program.Packages[0].CoreBodies)
	}
}

func checkTestGraph(t *testing.T, files []load.SourceFile) load.Graph {
	t.Helper()
	module := load.Module{Root: "/repo/case", Path: "example.com/case", Ok: true}
	graph := load.LoadGraph(module, "/std", "/repo/case", "./cmd/app", files)
	if !graph.Ok {
		t.Fatalf("LoadGraph failed: err=%d pkg=%d", graph.Error, graph.ErrorPackage)
	}
	return graph
}

func checkRootPackage(t *testing.T, graph load.Graph) int {
	t.Helper()
	for i := 0; i < len(graph.Packages); i++ {
		if graph.Packages[i].Ref.ImportPath == graph.Root {
			return i
		}
	}
	t.Fatal("root package not found")
	return -1
}
