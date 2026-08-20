package time

import "testing"

func TestParseDurationAndString(t *testing.T) {
	cases := []struct {
		input string
		want  Duration
		text  string
	}{
		{"0s", 0, "0s"},
		{"1h2m3.5s", Hour + 2*Minute + 3*Second + 500*Millisecond, "1h2m3.5s"},
		{"-250ms", -250 * Millisecond, "-250ms"},
		{"12us", 12 * Microsecond, "12us"},
	}
	for _, tc := range cases {
		got, err := ParseDuration(tc.input)
		if err != nil || got != tc.want || got.String() != tc.text {
			t.Fatalf("ParseDuration(%q)=%v,%v text=%q", tc.input, got, err, got.String())
		}
	}
	for _, input := range []string{"", "1", "ms", "1fortnight"} {
		if _, err := ParseDuration(input); err == nil {
			t.Fatalf("ParseDuration(%q) succeeded", input)
		}
	}
}
