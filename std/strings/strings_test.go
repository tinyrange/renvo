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

func TestExtendedSearchAndTrim(t *testing.T) {
	if !EqualFold("GoPHER", "gopher") || IndexByte("abc", 'b') != 1 || LastIndexByte("abca", 'a') != 3 {
		t.Fatal("extended search helpers failed")
	}
	if got := Trim("xyhelloxy", "xy"); got != "hello" {
		t.Fatalf("Trim = %q", got)
	}
	if got := TrimSpace("\u2003 hello \u2003"); got != "hello" {
		t.Fatalf("Unicode TrimSpace = %q", got)
	}
	parts := SplitN("a:b:c", ":", 2)
	if len(parts) != 2 || parts[0] != "a" || parts[1] != "b:c" {
		t.Fatalf("SplitN = %#v", parts)
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
