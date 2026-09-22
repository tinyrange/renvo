package driver

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestCollectSourcesSharedStandardAndModuleDirectory(t *testing.T) {
	for _, imports := range []string{
		"import (\"unicode/utf8\"; other \"example.com/case/std/unicode/utf8\")",
		"import (other \"example.com/case/std/unicode/utf8\"; \"unicode/utf8\")",
	} {
		fs := memorySourceFS{files: []load.SourceFile{
			{Path: "/repo/go.mod", Src: []byte("module example.com/case\n")},
			{Path: "/repo/cmd/app/main.go", Src: []byte("package main\n" + imports + "\nfunc main() { _ = utf8.Value; _ = other.Value }\n")},
			{Path: "/repo/std/unicode/utf8/utf8.go", Src: []byte("package utf8\nconst Value = 1\n")},
		}}
		result := CollectSourcesForTarget("/repo/cmd/app", "/repo/std", ".", "linux/amd64", fs)
		if !result.Ok {
			t.Fatalf("collect: error=%d path=%s", result.Error, result.ErrorPath)
		}
		count := 0
		for _, file := range result.Files {
			if file.Path == "/repo/std/unicode/utf8/utf8.go" {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("shared file collected %d times", count)
		}
		workspace := load.LoadWorkspace("/repo/cmd/app", "/repo/std", ".", result.Files)
		if !workspace.Ok {
			t.Fatalf("workspace error=%d graph error=%d", workspace.Error, workspace.Graph.Error)
		}
	}
}
