package strings

import (
	"io"
	"testing"
)

func TestSearchAndTrim(t *testing.T) {
	if !Contains("alpha beta", "ha b") || Index("banana", "na") != 2 || LastIndex("banana", "na") != 4 {
		t.Fatalf("search failed")
	}
	if !HasPrefix("prefix", "pre") || !HasSuffix("suffix", "fix") {
		t.Fatalf("prefix/suffix failed")
	}
	if TrimSpace("\t hi \n") != "hi" || TrimPrefix("prefix", "pre") != "fix" || TrimSuffix("suffix", "fix") != "suf" {
		t.Fatalf("trim failed")
	}
}

func TestSplitJoinReplace(t *testing.T) {
	parts := Split("a,b,c", ",")
	if len(parts) != 3 || parts[1] != "b" || Join(parts, "|") != "a|b|c" {
		t.Fatalf("split/join failed: %#v", parts)
	}
	fields := Fields(" a\tb\n c ")
	if len(fields) != 3 || fields[2] != "c" {
		t.Fatalf("fields failed: %#v", fields)
	}
	if Repeat("ab", 3) != "ababab" || Replace("aaaa", "aa", "b", 1) != "baa" || ReplaceAll("aaaa", "aa", "b") != "bb" {
		t.Fatalf("repeat/replace failed")
	}
}

func TestBuilderReaderAndCase(t *testing.T) {
	var b Builder
	b.Grow(8)
	b.WriteString("Go")
	b.WriteByte(' ')
	b.WriteRune('λ')
	if b.String() != "Go λ" || b.Len() != len("Go λ") {
		t.Fatalf("builder = %q", b.String())
	}
	r := NewReader("body")
	got, err := io.ReadAll(r)
	if err != nil || string(got) != "body" || r.Len() != 0 || r.Size() != 4 {
		t.Fatalf("reader = %q, %v", got, err)
	}
	if ToLower("ΓO") != "γo" || ToUpper("γo") != "ΓO" {
		t.Fatal("Unicode case conversion failed")
	}
}
