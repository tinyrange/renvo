package regexp

import "testing"

func TestMatchBasics(t *testing.T) {
	cases := []struct {
		pattern, text string
		want          bool
	}{
		{"abc", "xxabcyy", true},
		{"abc", "ab", false},
		{"^abc$", "abc", true},
		{"^abc$", "abcd", false},
		{"a.c", "abc", true},
		{"a.c", "a\\nc", false},
		{"[a-z]+", "123abc456", true},
		{"^[^x]*$", "hello", true},
		{"^[^x]*$", "hexlo", false},
		{"\\d+", "no digits", false},
		{"\\d+", "42", true},
		{"\\w+@\\w+\\.com", "user@example.com", true},
		{"\\s", "a b", true},
		{"\\S+", "   ", false},
		{"a|b|cd", "zzcd", true},
		{"^(cat|dog)s?$", "dogs", true},
		{"colou?r", "color", true},
		{"ab*c", "ac", true},
		{"ab+c", "abbbc", true},
		{"a{3}", "aaaa", true},
		{"^a{2,3}$", "aaaa", false},
		{"^a{2,}$", "aaaa", true},
		{"(?:ab)+", "ababab", true},
	}
	for _, tc := range cases {
		re, err := Compile(tc.pattern)
		if err != nil {
			t.Fatalf("compile %q: %v", tc.pattern, err)
		}
		if got := re.MatchString(tc.text); got != tc.want {
			t.Errorf("%q match %q = %v, want %v", tc.pattern, tc.text, got, tc.want)
		}
	}
}

func TestFindAndSubmatches(t *testing.T) {
	re := MustCompile("(\\w+)@(\\w+)\\.com")
	if got := re.FindString("mail bob@example.com now"); got != "bob@example.com" {
		t.Fatalf("find = %q", got)
	}
	loc := re.FindStringIndex("mail bob@example.com now")
	if loc == nil || loc[0] != 5 || loc[1] != 20 {
		t.Fatalf("index = %v", loc)
	}
	sub := re.FindStringSubmatch("from alice@test.com!")
	want := []string{"alice@test.com", "alice", "test"}
	if len(sub) != 3 || sub[0] != want[0] || sub[1] != want[1] || sub[2] != want[2] {
		t.Fatalf("submatch = %v", sub)
	}
	if re.FindStringSubmatch("no address") != nil {
		t.Fatal("expected nil submatch")
	}
	if re.NumSubexp() != 2 || re.String() != "(\\w+)@(\\w+)\\.com" {
		t.Fatalf("meta num=%d str=%q", re.NumSubexp(), re.String())
	}
}

func TestLeftmostFirstSemantics(t *testing.T) {
	// Go/PCRE leftmost-first: the first alternative that can match wins.
	re := MustCompile("(a|ab)(c|bcd)")
	if got := re.FindString("abcd"); got != "abcd" {
		t.Fatalf("leftmost-first = %q, want abcd", got)
	}
	sub := re.FindStringSubmatch("abcd")
	if sub[1] != "a" || sub[2] != "bcd" {
		t.Fatalf("groups = %v", sub)
	}
	lazy := MustCompile("a+?")
	if got := lazy.FindString("aaab"); got != "a" {
		t.Fatalf("lazy = %q", got)
	}
	range2 := MustCompile("<(.{2,4}?)>")
	if got := range2.FindString("<abcd>"); got != "<abcd>" {
		t.Fatalf("lazy range = %q, want <abcd>", got)
	}
	sub2 := range2.FindStringSubmatch("<abcd>")
	if sub2[1] != "abcd" {
		t.Fatalf("lazy range group = %q", sub2[1])
	}
}

func TestFindAllAndReplace(t *testing.T) {
	re := MustCompile("\\d+")
	all := re.FindAllString("a1 b22 c333", -1)
	if len(all) != 3 || all[0] != "1" || all[1] != "22" || all[2] != "333" {
		t.Fatalf("all = %v", all)
	}
	if got := re.FindAllString("a1 b22", 1); len(got) != 1 {
		t.Fatalf("limited = %v", got)
	}
	if got := re.Count("1-2-3"); got != 3 {
		t.Fatalf("count = %d", got)
	}
	swap := MustCompile("(\\w+) (\\w+)")
	if got := swap.ReplaceAllString("hello world and good bye", "$2 $1"); got != "world hello good and bye" {
		t.Fatalf("replace = %q", got)
	}
	empty := MustCompile("x*")
	if got := empty.ReplaceAllString("abc", "-"); got != "-a-b-c-" {
		t.Fatalf("empty replace = %q", got)
	}
}

func TestAnchorsAndEmptyMatches(t *testing.T) {
	re := MustCompile("^")
	all := re.FindAllString("ab", -1)
	if len(all) != 1 {
		t.Fatalf("^ matches = %v", all)
	}
	dollar := MustCompile("$")
	if got := dollar.FindStringIndex("xy"); got == nil || got[0] != 2 {
		t.Fatalf("$ index = %v", got)
	}
}

func TestErrors(t *testing.T) {
	cases := []struct {
		pattern string
		code    ErrorCode
	}{
		{"a(b", ErrMissingParen},
		{"a)b", ErrUnexpectedParen},
		{"a[bc", ErrMissingBracket},
		{"a\\", ErrTrailingBackslash},
		{"**", ErrInvalidRepeatOp},
		{"a**", ErrInvalidRepeatOp},
		{"a{2,1}", ErrInvalidRepeatSize},
		{"a{1001}", ErrInvalidRepeatSize},
		{"\\y", ErrInvalidEscape},
		{"[z-a]", ErrInvalidCharRange},
		{"[]]", ErrMissingBracket},
	}
	for _, tc := range cases {
		_, err := Compile(tc.pattern)
		if err == nil {
			t.Errorf("Compile(%q) succeeded, want error", tc.pattern)
			continue
		}
		e, ok := err.(*Error)
		if !ok || e.Code != tc.code {
			t.Errorf("Compile(%q) = %v, want code %v", tc.pattern, err, tc.code)
		}
	}
}

func TestBoundedExecution(t *testing.T) {
	// A classic catastrophic-backtracking pattern must stay fast because the
	// engine is a linear-time NFA simulation.
	re := MustCompile("(a+)+b")
	input := make([]byte, 0, 64)
	for i := 0; i < 60; i++ {
		input = append(input, 'a')
	}
	if re.Match(input) {
		t.Fatal("unexpected match")
	}
	prefix := MustCompile("^https?://[^/]+(/.*)?$")
	if !prefix.MatchString("https://example.com/path") {
		t.Fatal("url should match")
	}
}

func TestQuoteMetaAndPackageHelpers(t *testing.T) {
	quoted := QuoteMeta("a.b*c")
	if quoted != "a\\.b\\*c" {
		t.Fatalf("quote = %q", quoted)
	}
	if !MatchString(quoted, "a.b*c") || MatchString(quoted, "axb*c") {
		t.Fatal("quoted meta mismatch")
	}
	if !Match("[0-9]{2}", []byte("n=42")) {
		t.Fatal("Match failed")
	}
}

func TestIndicesSubmatchesAndSplit(t *testing.T) {
	re := MustCompile(`(\w+)@(\w+)\.com`)
	text := "a bob@example.com c alice@test.com"
	locs := re.FindAllStringIndex(text, -1)
	if len(locs) != 2 || locs[0][0] != 2 || locs[0][1] != 17 || locs[1][0] != 20 || locs[1][1] != 34 {
		t.Fatalf("all index = %v", locs)
	}
	if got := re.FindAllStringIndex(text, 1); len(got) != 1 {
		t.Fatalf("limited index = %v", got)
	}
	si := re.FindStringSubmatchIndex("bob@example.com")
	want := []int{0, 15, 0, 3, 4, 11}
	if len(si) != len(want) {
		t.Fatalf("submatch index length = %d", len(si))
	}
	for i := range want {
		if si[i] != want[i] {
			t.Fatalf("submatch index = %v", si)
		}
	}
	if re.FindStringSubmatchIndex("nope") != nil {
		t.Fatal("expected nil submatch index")
	}
	all := re.FindAllStringSubmatch("bob@example.com sue@test.com", -1)
	if len(all) != 2 || all[0][1] != "bob" || all[0][2] != "example" || all[1][1] != "sue" || all[1][2] != "test" {
		t.Fatalf("all submatch = %v", all)
	}
	split := MustCompile(`[, ]+`).Split("a, b,,c d", -1)
	if len(split) != 4 || split[0] != "a" || split[1] != "b" || split[2] != "c" || split[3] != "d" {
		t.Fatalf("split = %v", split)
	}
	limited := MustCompile(`[, ]+`).Split("a, b,,c d", 2)
	if len(limited) != 2 || limited[0] != "a" || limited[1] != "b,,c d" {
		t.Fatalf("split limited = %v", limited)
	}
	edges := MustCompile(",").Split(",a,", -1)
	if len(edges) != 3 || edges[0] != "" || edges[1] != "a" || edges[2] != "" {
		t.Fatalf("split edges = %q", edges)
	}
	empty := MustCompile("x").Split("", -1)
	if len(empty) != 1 || empty[0] != "" {
		t.Fatalf("split empty = %q", empty)
	}
}

func TestReplaceVariants(t *testing.T) {
	digits := MustCompile(`\d+`)
	if got := digits.ReplaceAllLiteralString("a1 b22", "$1"); got != "a$1 b$1" {
		t.Fatalf("literal replace = %q", got)
	}
	shout := func(word string) string { return word + "!" }
	if got := MustCompile(`\w+`).ReplaceAllStringFunc("ab cd", shout); got != "ab! cd!" {
		t.Fatalf("func replace = %q", got)
	}
	if got := MustCompile(`x*`).ReplaceAllLiteralString("abc", "-"); got != "-a-b-c-" {
		t.Fatalf("empty literal replace = %q", got)
	}
}
