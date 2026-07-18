package driver

import (
	"testing"

	"j5.nz/rtg/rtg/internal/load"
)

func TestAnalyzeWorkspaceReportsParserAndCheckerDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		code string
	}{
		{"parser", "package main\nfunc main( {\n", "RTG-PARSE-001"},
		{"checker", "package main\nfunc main() { var value bool; value = 1; _ = value }\n", "RTG-CHECK-008"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := AnalyzeWorkspace("/repo", "/std", ".", []load.SourceFile{
				{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
				{Path: "/repo/main.go", Src: []byte(test.src)},
			})
			if result.Ok || result.Diagnostic.Code != test.code || result.Diagnostic.Path != "/repo/main.go" || result.Diagnostic.Line < 1 || result.Diagnostic.Column < 1 {
				t.Fatalf("analysis = %#v", result)
			}
		})
	}
}

func TestAnalyzeWorkspaceAcceptsValidFrontendProgram(t *testing.T) {
	result := AnalyzeWorkspace("/repo", "/std", ".", []load.SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/main.go", Src: []byte("package main\nfunc main() { value := 1; _ = value }\n")},
	})
	if !result.Ok || result.Diagnostic.Valid() {
		t.Fatalf("analysis = %#v", result)
	}
}
