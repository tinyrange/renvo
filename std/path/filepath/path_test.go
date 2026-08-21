package filepath

import (
	"strings"
	"testing"
)

func TestLexicalPaths(t *testing.T) {
	cases := map[string]string{"": ".", ".": ".", "a//b/../c": "a/c", "/../../a": "/a", "../a": "../a"}
	for input, want := range cases {
		if got := Clean(input); got != want {
			t.Fatalf("Clean(%q)=%q want %q", input, got, want)
		}
	}
	if Join("/a", "b", "..", "c") != "/a/c" {
		t.Fatal(Join("/a", "b", "..", "c"))
	}
	if Base("/a/b.txt") != "b.txt" || Dir("/a/b.txt") != "/a" || Ext("/a/b.txt") != ".txt" {
		t.Fatal("base/dir/ext")
	}
	dir, file := Split("a/b.txt")
	if dir != "a/" || file != "b.txt" {
		t.Fatal(dir, file)
	}
	if got, err := Rel("/a/b", "/a/c/d"); err != nil || got != "../c/d" {
		t.Fatal(got, err)
	}
	if abs, err := Abs("relative"); err != nil || !IsAbs(abs) || !strings.HasSuffix(abs, "/relative") {
		t.Fatal(abs, err)
	}
}

func TestMatch(t *testing.T) {
	cases := []struct {
		pattern, name string
		want          bool
	}{
		{"*.go", "main.go", true}, {"*.go", "dir/main.go", false}, {"file?.txt", "file1.txt", true},
		{"[a-c].txt", "b.txt", true}, {"[^a-c].txt", "z.txt", true}, {`a\*b`, "a*b", true},
	}
	for _, tc := range cases {
		got, err := Match(tc.pattern, tc.name)
		if err != nil || got != tc.want {
			t.Fatalf("Match(%q,%q)=%v,%v", tc.pattern, tc.name, got, err)
		}
	}
	if _, err := Match("[bad", "b"); err != ErrBadPattern {
		t.Fatalf("bad pattern=%v", err)
	}
}

func TestGlob(t *testing.T) {
	matches, err := Glob("*.go")
	if err != nil || len(matches) < 2 {
		t.Fatalf("Glob=%q,%v", matches, err)
	}
	for i := 1; i < len(matches); i++ {
		if matches[i-1] > matches[i] {
			t.Fatalf("unsorted: %q", matches)
		}
	}
}
