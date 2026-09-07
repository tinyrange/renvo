package load

import "testing"

func TestEffectiveFileGoVersion(t *testing.T) {
	for _, test := range []struct{ module, constraint, want string }{
		{"", "", "1.16"},
		{"1.26.2", "", "1.26"},
		{"1.26rc1", "", "1.26"},
		{"1.22", "go1.26", "1.26"},
		{"1.26", "go1.22", "1.22"},
		{"1.26", "go1.19", "1.21"},
		{"1.16", "go1.22", "1.22"},
		{"1.26", "linux && go1.22", "1.22"},
		{"1.26", "go1.22 || go1.24", "1.22"},
		{"1.26", "go1.22 && go1.24", "1.24"},
		{"1.26", "linux || go1.22", "1.26"},
		{"1.26", "!go1.22", "1.26"},
		{"1.26", "!(linux || !go1.22)", "1.22"},
		{"1.26", "!(linux && !go1.22)", "1.26"},
		{"1.26", "(linux && go1.22) || (windows && go1.24)", "1.22"},
		{"1.26", "go1.22.0", "1.26"},
		{"1.26", "go1.22 &&", "1.26"},
	} {
		src := "/* license */\n// comment\n"
		if test.constraint != "" {
			src += "//go:build " + test.constraint + "\n"
		}
		src += "\npackage main\n"
		pkg := Package{Ref: PackageRef{Kind: PackageInModule}, GoVersion: test.module}
		if got := effectiveFileGoVersion(pkg, []byte(src)); got != test.want {
			t.Fatalf("%q %q: %q, want %q", test.module, test.constraint, got, test.want)
		}
	}
}

func TestFileVersionDoesNotBorrowUnrelatedDirectives(t *testing.T) {
	pkg := Package{Ref: PackageRef{Kind: PackageInModule}, GoVersion: "1.26"}
	if got := effectiveFileGoVersion(pkg, []byte("\xef\xbb\xbf//go:build go1.22\n\npackage main\n")); got != "1.22" {
		t.Fatalf("BOM header: %q", got)
	}
	for _, src := range []string{
		"package main\n//go:build go1.22\n",
		"/* //go:build go1.22 */\npackage main\n",
		"// +build go1.22\n\npackage main\n",
	} {
		if got := effectiveFileGoVersion(pkg, []byte(src)); got != "1.26" {
			t.Fatalf("%s: %q", src, got)
		}
	}
	if got := effectiveFileGoVersion(Package{Ref: PackageRef{Kind: PackageStandard}}, []byte("package fmt")); got != CompilerGoVersion {
		t.Fatalf("standard library: %q", got)
	}
}

func TestStandardLibraryFileLanguageOwnership(t *testing.T) {
	for _, moduleVersion := range []string{"1.16", "1.26"} {
		workspace := LoadWorkspace("/repo", "/std", ".", []SourceFile{
			{Path: "/repo/go.mod", Src: []byte("module example.com/app\ngo " + moduleVersion + "\n")},
			{Path: "/repo/main.go", Src: []byte("package main\nimport \"versionprobe\"\nfunc main(){versionprobe.Use()}\n")},
			{Path: "/std/versionprobe/base.go", Src: []byte("package versionprobe\nfunc Use(){}\n")},
			{Path: "/std/versionprobe/tagged.go", Src: []byte("//go:build go1.22\n\npackage versionprobe\n")},
		})
		if !workspace.Ok {
			t.Fatalf("load failed: %#v", workspace)
		}
		found := false
		for _, pkg := range workspace.Graph.Packages {
			if pkg.Ref.Kind != PackageStandard {
				continue
			}
			found = true
			if pkg.GoVersion != "" {
				t.Fatalf("standard package borrowed module directive %q", pkg.GoVersion)
			}
			if len(pkg.Files) != 2 || pkg.Files[0].GoVersion != CompilerGoVersion || pkg.Files[1].GoVersion != "1.22" {
				t.Fatalf("standard files: %#v", pkg.Files)
			}
		}
		if !found {
			t.Fatal("standard package missing")
		}
	}
}

func TestLoadedFilesRetainEffectiveVersions(t *testing.T) {
	workspace := LoadWorkspace("/repo", "/std", ".", []SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\ngo 1.26.0\n")},
		{Path: "/repo/main.go", Src: []byte("package main\nfunc main() {}\n")},
		{Path: "/repo/version.go", Src: []byte("//go:build go1.22\n\npackage main\n")},
	})
	if !workspace.Ok {
		t.Fatalf("load failed: %#v", workspace)
	}
	files := workspace.Graph.Packages[0].Files
	if len(files) != 2 || files[0].GoVersion != "1.26" || files[1].GoVersion != "1.22" {
		t.Fatalf("files: %#v", files)
	}
}
