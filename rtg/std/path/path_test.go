package path

import "testing"

func TestPathHelpers(t *testing.T) {
	if Clean("/a//b/../c") != "/a/c" || Clean("") != "." || Join("a", "b", "..", "c") != "a/c" {
		t.Fatalf("clean/join failed")
	}
	if Base("/a/b.txt") != "b.txt" || Dir("/a/b.txt") != "/a" || Ext("/a/b.txt") != ".txt" || !IsAbs("/a") {
		t.Fatalf("path component failed")
	}
	dir, file := Split("/a/b.txt")
	if dir != "/a/" || file != "b.txt" {
		t.Fatalf("Split = %q %q", dir, file)
	}
}
