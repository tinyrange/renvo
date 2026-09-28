package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestIncrementalObjectRootsAndCacheModeIsolation(t *testing.T) {
	packageArtifactCacheUsed = nil
	packageArtifactCacheData = nil
	InitializePackageArtifactCache()
	packageArtifactCacheNext = 0
	packageArtifactCacheHits = 0
	packageArtifactCacheMisses = 0
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.22\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc Add(a,b int)int{return a+b}\nfunc main(){_=Add(1,2)}\n")},
	}
	for _, object := range []bool{true, true, false, true, false} {
		built := buildFromFiles(t, files)
		var session *PackageSession
		if object {
			session = BeginObjectPackageSession(built, false)
		} else {
			session = BeginPackageSession(built, false)
		}
		for !session.Step() {
		}
		got := session.Result()
		if !got.Ok {
			t.Fatalf("object=%v failed: %#v", object, got)
		}
		for _, name := range []string{"Add", "main"} {
			if found := bytes.Contains(got.Program.Text, []byte("//export "+name+"\n")); found != object {
				t.Fatalf("object=%v export %s=%v\n%s", object, name, found, got.Program.Text)
			}
		}
	}
	if packageArtifactCacheHits == 0 {
		t.Fatal("repeated object build did not reuse artifacts")
	}
}

func TestIncrementalObjectPolicySeparatesC11Artifacts(t *testing.T) {
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.22\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc Add(a,b int)int{return a+b}\nfunc appMain()int{return Add(1,2)}\n")},
	}
	for _, c11 := range []bool{true, false, true} {
		built := buildFromFiles(t, files)
		built.Units[built.Root].C11 = c11
		wantLines := make([]int, len(built.Units[built.Root].Program.Tokens))
		for i, tok := range built.Units[built.Root].Program.Tokens {
			wantLines[i] = tok.KindLine
		}
		session := BeginObjectPackageSession(built, false)
		for !session.Step() {
		}
		got := session.Result()
		if !got.Ok {
			t.Fatal("object session failed")
		}
		if exported := bytes.Contains(got.Program.Text, []byte("//export Add\n")); exported == c11 {
			t.Fatalf("C11=%v automatic export=%v", c11, exported)
		}
		for i, tok := range built.Units[built.Root].Program.Tokens {
			if tok.KindLine != wantLines[i] {
				t.Fatalf("source token %d mutated: %d want %d", i, tok.KindLine, wantLines[i])
			}
		}
	}
}
