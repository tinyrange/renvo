package time

import "testing"

func TestCalendarAndUnix(t *testing.T) {
	cases := []struct {
		y    int
		m    Month
		d    int
		unix int64
	}{
		{1970, January, 1, 0}, {1969, December, 31, -86400},
		{2000, February, 29, 951782400}, {2024, March, 1, 1709251200},
	}
	for _, c := range cases {
		got := Date(c.y, c.m, c.d, 0, 0, 0, 0, UTC)
		y, m, d := got.Date()
		if got.Unix() != c.unix || y != c.y || m != c.m || d != c.d {
			t.Fatalf("Date(%d,%d,%d)=%d %d-%d-%d", c.y, c.m, c.d, got.Unix(), y, m, d)
		}
	}
	if Unix(0, -1).Unix() != -1 || Unix(0, -1).Nanosecond() != 999999999 {
		t.Fatal("negative Unix normalization")
	}
}

func TestRFC3339(t *testing.T) {
	values := []string{"1970-01-01T00:00:00Z", "2024-02-29T12:34:56.123456789Z", "2023-07-08T09:10:11+05:30"}
	for _, value := range values {
		got, err := Parse(RFC3339Nano, value)
		if err != nil {
			t.Fatalf("Parse(%q): %v", value, err)
		}
		if text := got.Format(RFC3339Nano); text != value {
			t.Fatalf("round trip %q = %q", value, text)
		}
	}
	for _, value := range []string{"", "2023-02-29T00:00:00Z", "2020-01-01T25:00:00Z", "2020-01-01T00:00:00"} {
		if _, err := Parse(RFC3339, value); err == nil {
			t.Fatalf("Parse(%q) succeeded", value)
		}
	}
}

func TestNowAndSince(t *testing.T) {
	start := Now()
	for i := 0; i < 100000; i++ {
	}
	elapsed := Since(start)
	if start.Year() < 2020 || elapsed < 0 || elapsed > Minute {
		t.Fatalf("Now/Since returned start=%v elapsed=%v", start, elapsed)
	}
}

func TestArithmeticAndComparison(t *testing.T) {
	a := Unix(10, 250)
	b := a.Add(2*Second + 750)
	if b.Sub(a) != 2*Second+750 || !b.After(a) || !a.Before(b) || !a.Equal(a.UTC()) {
		t.Fatal("arithmetic/comparison")
	}
}
