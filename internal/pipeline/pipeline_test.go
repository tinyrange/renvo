package pipeline

import (
	"bytes"
	"testing"

	"renvo.dev/backend/unit"
	"renvo.dev/internal/c11"
	"renvo.dev/internal/load"
)

func TestBuildObjectUnitPreservesC11SemanticsDirective(t *testing.T) {
	result := BuildObjectUnit("/repo/case", "/std", ".", []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{
			Path:       "/repo/case/value.c",
			Src:        []byte(`int read(const int *value) { return *value; }`),
			CObject:    true,
			CDataModel: c11.DataModelLP64,
		},
	})
	if !result.Ok {
		t.Fatalf("BuildObjectUnit failed: err=%d pkg=%d file=%d tok=%d", result.Error, result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}
	decoded, err := unit.Unmarshal(result.Link.Data)
	if err != nil {
		t.Fatalf("object unit did not decode: %v", err)
	}
	if !bytes.Contains(decoded.Text, []byte("// renvo:c11\n")) {
		t.Fatalf("linked object unit lost C11 semantics directive:\n%s", decoded.Text)
	}
}

func TestBuildUnitLinksWorkspace(t *testing.T) {
	result := BuildUnit("/repo/case", "/std", "./cmd/app", []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func appMain() int { return lib.Value() }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

const answer = 42

func Value() int { return answer }
`)},
	})
	if !result.Ok {
		t.Fatalf("BuildUnit failed: err=%d pkg=%d file=%d tok=%d", result.Error, result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}
	if len(result.Link.Data) == 0 {
		t.Fatal("BuildUnit returned empty linked unit data")
	}
	decoded, err := unit.Unmarshal(result.Link.Data)
	if err != nil {
		t.Fatalf("linked unit did not decode: %v", err)
	}
	if decoded.Package != "main" {
		t.Fatalf("linked package = %q, want main", decoded.Package)
	}
	if len(decoded.Decls) != 1 || len(decoded.Funcs) != 2 {
		t.Fatalf("linked decls/funcs = %d/%d, want 1/2", len(decoded.Decls), len(decoded.Funcs))
	}
	if result.Build.Root != 1 || result.Build.Units[result.Build.Root].ImportPath != "example.com/case/cmd/app" {
		t.Fatalf("root unit = %d %#v", result.Build.Root, result.Build.Units)
	}
}

func TestBuildUnitLinksMixedGoAndC11Files(t *testing.T) {
	result := BuildUnit("/repo/case", "/std", "./cmd/app", []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

func goDouble(value int) int { return value * 2 }

func main() {
	if cAdd(20, 22) == 42 && callGo(21) == 42 {
		print("PASS\n")
	}
}
`)},
		{Path: "/repo/case/cmd/app/math.c", Src: []byte(`
int cAdd(int left, int right) { return left + right; }
int callGo(int value) { return goDouble(value); }
`)},
	})
	if !result.Ok {
		t.Fatalf("mixed BuildUnit failed: err=%d pkg=%d file=%d tok=%d graph=%#v", result.Error, result.ErrorPackage, result.ErrorFile, result.ErrorToken, result.Workspace.Graph)
	}
	decoded, err := unit.Unmarshal(result.Link.Data)
	if err != nil {
		t.Fatalf("mixed linked unit did not decode: %v", err)
	}
	for _, name := range []string{"goDouble", "cAdd", "callGo", "main", "appMain"} {
		found := false
		for i := 0; i < len(decoded.Funcs); i++ {
			fn := decoded.Funcs[i]
			if string(decoded.Text[fn.NameStart:fn.NameEnd]) == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("linked unit is missing %s: %s", name, decoded.Text)
		}
	}
}

func TestBuildUnitCompilesC11OnlyRoot(t *testing.T) {
	result := BuildUnit("/repo/case", "/std", "./cmd/app", []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.c", Src: []byte(`int main(void) { print("PASS\n"); return 0; }`)},
	})
	if !result.Ok {
		t.Fatalf("C-only BuildUnit failed: err=%d pkg=%d file=%d tok=%d", result.Error, result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}
	decoded, err := unit.Unmarshal(result.Link.Data)
	if err != nil {
		t.Fatalf("C-only linked unit did not decode: %v", err)
	}
	if decoded.Package != "main" {
		t.Fatalf("C-only package = %q, want main", decoded.Package)
	}
	if !bytes.Contains(decoded.Text, []byte("// renvo:c11\n")) {
		t.Fatalf("linked executable unit lost C11 semantics directive:\n%s", decoded.Text)
	}
}

func TestBuildUnitReportsLoadError(t *testing.T) {
	result := BuildUnit("/repo/case", "/std", "./cmd/app", []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc appMain() int { return 0 }\n")},
	})
	if result.Ok || result.Error != PipelineErrLoad {
		t.Fatalf("missing module result = %#v", result)
	}
	if result.ErrorFile != -1 || result.ErrorPackage != -1 || result.ErrorToken != -1 {
		t.Fatalf("load error location = pkg %d file %d tok %d", result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}

	duplicate := BuildUnit("/repo/case", "/std", "./cmd/app", []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
	})
	if duplicate.Ok || duplicate.Error != PipelineErrLoad {
		t.Fatalf("duplicate file result = %#v", duplicate)
	}
	if duplicate.ErrorPackage != -1 || duplicate.ErrorFile != 1 || duplicate.ErrorToken != -1 {
		t.Fatalf("duplicate load location = pkg %d file %d tok %d", duplicate.ErrorPackage, duplicate.ErrorFile, duplicate.ErrorToken)
	}
}

func TestBuildUnitReportsBuildError(t *testing.T) {
	result := BuildUnit("/repo/case", "/std", "./cmd/app", []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/a.go", Src: []byte("package main\nvar value int\n")},
		{Path: "/repo/case/cmd/app/b.go", Src: []byte("package main\nfunc value() {}\n")},
	})
	if result.Ok || result.Error != PipelineErrBuild {
		t.Fatalf("duplicate symbol result = %#v", result)
	}
	if result.ErrorPackage != 0 || result.ErrorFile != 1 || result.ErrorToken < 0 {
		t.Fatalf("build error location = pkg %d file %d tok %d", result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}
}
