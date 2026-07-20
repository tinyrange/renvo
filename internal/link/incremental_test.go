package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestIncrementalPackageArtifactsMatchWholeProgramLinkAndReuseDependencies(t *testing.T) {
	packageArtifactCacheUsed = nil
	packageArtifactCacheData = nil
	InitializePackageArtifactCache()
	packageArtifactCacheNext = 0
	packageArtifactCacheHits = 0
	packageArtifactCacheMisses = 0
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n\nimport \"example.com/case/pkg/lib\"\n\nfunc appMain() int { return lib.Value() }\n")},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte("package lib\n\nfunc Value() int { return 42 }\n")},
	}
	firstBuild := buildFromFiles(t, files)
	want := LinkBuildCore(firstBuild)
	got := LinkBuildCoreIncremental(firstBuild)
	if !want.Ok || !got.Ok || !bytes.Equal(got.Data, want.Data) {
		t.Fatalf("incremental link differs from whole-program link: whole=%v incremental=%v data=%d/%d text=%d/%d tokens=%#v/%#v funcs=%#v/%#v\nwhole text:\n%s\nincremental text:\n%s", want.Ok, got.Ok, len(want.Data), len(got.Data), len(want.Program.Text), len(got.Program.Text), want.Program.Tokens, got.Program.Tokens, want.Program.Funcs, got.Program.Funcs, want.Program.Text, got.Program.Text)
	}
	if packageArtifactCacheHits != 0 || packageArtifactCacheMisses != 2 {
		t.Fatalf("cold artifact hits/misses = %d/%d, want 0/2", packageArtifactCacheHits, packageArtifactCacheMisses)
	}
	if len(got.Program.Packages) != 2 || got.Program.Packages[0].ImportPath != "example.com/case/pkg/lib" || got.Program.Packages[1].ImportPath != "example.com/case/cmd/app" {
		t.Fatalf("linked package ownership = %#v", got.Program.Packages)
	}

	files[1].Src = []byte("package main\n\nimport \"example.com/case/pkg/lib\"\n\nfunc appMain() int { return lib.Value() + 1 }\n")
	secondBuild := buildFromFiles(t, files)
	second := LinkBuildCoreIncremental(secondBuild)
	if !second.Ok || !bytes.Contains(second.Program.Text, []byte("Value() + 1")) {
		t.Fatalf("incremental root rebuild failed: ok=%v\n%s", second.Ok, second.Program.Text)
	}
	if packageArtifactCacheHits != 1 || packageArtifactCacheMisses != 3 {
		t.Fatalf("warm artifact hits/misses = %d/%d, want 1/3", packageArtifactCacheHits, packageArtifactCacheMisses)
	}
	if packageArtifactCacheNext != 2 {
		t.Fatalf("edited package consumed a new persistent cache slot: next = %d, want 2", packageArtifactCacheNext)
	}
}
