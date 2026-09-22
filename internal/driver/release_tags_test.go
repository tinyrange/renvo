package driver

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestReleaseBuildConstraints(t *testing.T) {
	for _, tc := range []struct {
		expr string
		want bool
	}{
		{"go1.1 && go1.9 && go1.25", true},
		{"go1.26", false}, {"!go1.26 && go1.22", true},
		{"(go1.25 && linux) || go1.26", true},
		{"go1.25 && windows", false}, {"go1.25.0", false},
	} {
		enabled, valid := sourceConstraintsEnabled([]byte("//go:build "+tc.expr+"\n\npackage p\n"), "linux/amd64", nil)
		if !valid || enabled != tc.want {
			t.Errorf("%q: enabled=%v valid=%v", tc.expr, enabled, valid)
		}
	}
	if enabled, valid := evalPlusBuildLine([]byte("go1.22,!go1.26"), "linux/amd64", nil); !enabled || !valid {
		t.Fatal("legacy release tags")
	}
	if !hasBuildTag("linux/amd64", "go1.26", []string{"go1.26"}) {
		t.Fatal("explicit user tag lost")
	}
}

func TestCollectedReleaseTagsAndFileVersions(t *testing.T) {
	for _, version := range []string{"1.16", "1.26"} {
		fs := memorySourceFS{files: []load.SourceFile{
			{Path: "/repo/app/go.mod", Src: []byte("module example.com/app\ngo " + version + "\n")},
			{Path: "/repo/app/main.go", Src: []byte("//go:build go1.22 && !go1.26\n\npackage main\nfunc main(){}\n")},
			{Path: "/repo/app/future.go", Src: []byte("//go:build go1.26\n\npackage broken\n")},
		}}
		collected := CollectSourcesForTarget("/repo/app", "/std", ".", "linux/amd64", fs)
		if !collected.Ok {
			t.Fatalf("collect: %#v", collected)
		}
		assertSourcePaths(t, collected.Files, []string{"/repo/app/go.mod", "/repo/app/main.go"})
		workspace := load.LoadWorkspace("/repo/app", "/std", ".", collected.Files)
		if !workspace.Ok {
			t.Fatalf("load: %#v", workspace)
		}
		pkg := workspace.Graph.Packages[0]
		if pkg.GoVersion != version || pkg.Files[0].GoVersion != "1.22" {
			t.Fatalf("module=%q file=%q", pkg.GoVersion, pkg.Files[0].GoVersion)
		}
	}
}
