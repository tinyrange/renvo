package strconv

import "testing"

func TestParseIntegerRangeLimits(t *testing.T) {
	for _, test := range []struct {
		text     string
		bits     int
		want     int64
		overflow bool
	}{
		{"127", 8, 127, false}, {"128", 8, 127, true}, {"-128", 8, -128, false}, {"-129", 8, -128, true},
		{"9223372036854775807", 64, 9223372036854775807, false},
		{"9223372036854775808", 64, 9223372036854775807, true},
		{"-9223372036854775808", 64, -9223372036854775808, false},
		{"-9223372036854775809", 64, -9223372036854775808, true},
	} {
		got, err := ParseInt(test.text, 10, test.bits)
		if got != test.want || (err == ErrRange) != test.overflow || !test.overflow && err != nil {
			t.Fatal("signed range", test.text)
		}
	}
	for _, test := range []struct {
		text     string
		bits     int
		want     uint64
		overflow bool
	}{
		{"255", 8, 255, false}, {"256", 8, 255, true}, {"65536", 16, 65535, true},
		{"18446744073709551615", 64, 18446744073709551615, false},
		{"18446744073709551616", 64, 18446744073709551615, true},
		{"184467440737095516150", 64, 18446744073709551615, true},
	} {
		got, err := ParseUint(test.text, 10, test.bits)
		if got != test.want || (err == ErrRange) != test.overflow || !test.overflow && err != nil {
			t.Fatal("unsigned range", test.text)
		}
	}
}

func TestParseIntegerUnderscores(t *testing.T) {
	for _, source := range []string{"0xff", "0x_ff", "0b1111_1111", "0o377", "0377", "2_55"} {
		value, err := ParseUint(source, 0, 16)
		if err != nil || value != 255 {
			t.Fatal("base detection", source)
		}
	}
	for _, source := range []string{"_1", "1_", "1__2", "0x__1", "0x_", "-1", "+1"} {
		if _, err := ParseUint(source, 0, 64); err == nil {
			t.Fatal("accepted invalid integer", source)
		}
	}
	if _, err := ParseUint("1_0", 10, 64); err == nil {
		t.Fatal("explicit base accepted underscore")
	}
}
