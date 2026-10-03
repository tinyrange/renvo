package check

import "testing"

func TestWideIndexSigned64Limits(t *testing.T) {
	for _, test := range []struct {
		text  string
		value int64
		valid bool
	}{
		{"0", 0, true},
		{"2147483648", 1 << 31, true},
		{"4294967296", 1 << 32, true},
		{"9223372036854775807", 1<<63 - 1, true},
		{"-9223372036854775808", -1 << 63, true},
		{"9223372036854775808", 0, false},
		{"-9223372036854775809", 0, false},
		{"18446744073709551616", 0, false},
	} {
		t.Run(test.text, func(t *testing.T) {
			text := test.text
			negative := text[0] == '-'
			if negative {
				text = text[1:]
			}
			constant := wideIntegerLiteral(text)
			if negative {
				constant = wideNegate(constant)
			}
			value, ok := wideInt64(constant)
			if ok != test.valid || ok && value != test.value {
				t.Fatalf("value=%d valid=%v, want %d/%v", value, ok, test.value, test.valid)
			}
		})
	}
	if _, ok := wideInt64(wideConstant{}); ok {
		t.Fatal("invalid constant became an index")
	}
}
