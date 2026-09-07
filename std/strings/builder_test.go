package strings

import "testing"

func TestBuilderAndRuneSearch(t *testing.T) {
	var b Builder
	b.Grow(32)
	b.WriteString("snow ")
	n, err := b.WriteRune('☃')
	if err != nil || n != 3 {
		t.Fatal("rune write")
	}
	b.WriteByte('!')
	first := b.String()
	b.Reset()
	b.WriteString("new")
	if first != "snow ☃!" || b.String() != "new" {
		t.Fatal("builder contents")
	}
	if IndexRune(first, '☃') != 5 || !ContainsRune("\xff", '\ufffd') ||
		ContainsRune("x", -1) || IndexAny(first, "☃!") != 5 {
		t.Fatal("rune search")
	}
}

func TestBuilderRejectsCopy(t *testing.T) {
	var b Builder
	b.WriteString("x")
	copied := b
	defer func() {
		if recover() == nil {
			t.Fatal("builder accepted copy")
		}
	}()
	copied.WriteString("y")
}

func TestTrimLeftRuneCutset(t *testing.T) {
	if TrimLeft("雪雪 x雪", "雪 ") != "x雪" || TrimLeft("\xffx", "\ufffd") != "x" ||
		TrimLeft("abc", "") != "abc" || TrimLeft("", "x") != "" {
		t.Fatal("TrimLeft cutset")
	}
}
