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

func TestIntegerBoundsAndErrors(t *testing.T) {
	if FormatInt(-9223372036854775807-1, 10) != "-9223372036854775808" {
		t.Fatal("MinInt64 formatting")
	}
	if value, err := ParseInt("-128", 10, 8); err != nil || value != -128 {
		t.Fatalf("ParseInt min = %d, %v", value, err)
	}
	if value, err := ParseInt("128", 10, 8); value != 127 || err == nil {
		t.Fatalf("ParseInt overflow = %d, %v", value, err)
	}
	if value, err := ParseUint("256", 10, 8); value != 255 || err == nil {
		t.Fatalf("ParseUint overflow = %d, %v", value, err)
	}
	if value, err := ParseUint("0b1010_0101", 0, 8); value != 165 || err != nil {
		t.Fatalf("ParseUint prefix = %d, %v", value, err)
	}
	_, err := ParseUint("300", 10, 8)
	num, ok := err.(*NumError)
	if !ok || num.Func != "ParseUint" || num.Num != "300" || num.Err != ErrRange {
		t.Fatalf("NumError = %#v", err)
	}
}

func TestFloatAndAppend(t *testing.T) {
	for _, tc := range []struct {
		text  string
		value float64
	}{{"1.5", 1.5}, {"-2.5e2", -250}, {".125", 0.125}} {
		value, err := ParseFloat(tc.text, 64)
		if err != nil || value != tc.value {
			t.Fatalf("ParseFloat(%q)=%v,%v", tc.text, value, err)
		}
	}
	if FormatFloat(12.5, 'f', 2, 64) != "12.50" || FormatFloat(12.5, 'g', -1, 64) != "12.5" || FormatFloat(1250, 'e', 2, 64) != "1.25e+03" {
		t.Fatal("FormatFloat")
	}
	if string(AppendInt([]byte("x"), -12, 10)) != "x-12" || string(AppendQuote(nil, "a\n")) != "\"a\\n\"" {
		t.Fatal("Append")
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
