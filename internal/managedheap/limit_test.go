package managedheap

import "testing"

func TestLimitSyntaxAndOverflow(t *testing.T) {
	valid := map[string]uint64{
		"1": 1, "00064": 64, "64KiB": 65536, "128MiB": 128 << 20,
		"2GiB": 2 << 30, "3TiB": 3 << 40,
		"18446744073709551615": ^uint64(0),
		"18014398509481983KiB": (^uint64(0) >> 10) << 10,
	}
	for text, want := range valid {
		if got, ok := ParseLimit(text); !ok || got != want {
			t.Errorf("ParseLimit(%q) = %d, %v; want %d", text, got, ok, want)
		}
	}
	for _, text := range []string{"", "0", "0MiB", "-1", "+1", " 1", "1 ", "1.5MiB", "1MB", "1mib", "MiB", "18446744073709551616", "18014398509481984KiB", "16777216TiB", "99999999999999999999999999999999999999"} {
		if _, ok := ParseLimit(text); ok {
			t.Errorf("ParseLimit(%q) unexpectedly succeeded", text)
		}
	}
}

func TestLimitPrecedence(t *testing.T) {
	for _, tc := range []struct {
		target, compiled uint64
		environment      string
		present          bool
		want             uint64
		ok               bool
	}{
		{0, 0, "", false, 0, true},
		{1024, 0, "", false, 1024, true},
		{1024, 2048, "", false, 2048, true},
		{1024, 2048, "512", true, 512, true},
		{1024, 2048, "1GiB", true, 1 << 30, true},
		{1024, 2048, "", true, 0, false},
		{1024, 2048, "bad", true, 0, false},
	} {
		got, ok := ResolveLimit(tc.target, tc.compiled, tc.environment, tc.present)
		if got != tc.want || ok != tc.ok {
			t.Errorf("ResolveLimit(%+v) = %d, %v", tc, got, ok)
		}
	}
}
