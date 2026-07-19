package strconv

import "testing"

func TestIntsAndBools(t *testing.T) {
	if Itoa(-42) != "-42" || FormatInt(255, 16) != "ff" || FormatUint(35, 36) != "z" {
		t.Fatalf("format integer failed")
	}
	if v, err := Atoi("-17"); err != nil || v != -17 {
		t.Fatalf("Atoi = %d, %v", v, err)
	}
	if v, err := ParseUint("ff", 16, 0); err != nil || v != 255 {
		t.Fatalf("ParseUint = %d, %v", v, err)
	}
	if b, err := ParseBool("True"); err != nil || !b || FormatBool(false) != "false" {
		t.Fatalf("bool conversion failed")
	}
}

func TestQuote(t *testing.T) {
	q := Quote("a\n\"b\"")
	if q != "\"a\\n\\\"b\\\"\"" {
		t.Fatalf("Quote = %q", q)
	}
	u, err := Unquote(q)
	if err != nil || u != "a\n\"b\"" {
		t.Fatalf("Unquote = %q, %v", u, err)
	}
	if _, err := Unquote("bad"); err == nil {
		t.Fatalf("Unquote accepted invalid input")
	}
}
