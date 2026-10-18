package driver

import "testing"

func TestHeapLimitParsing(t *testing.T) {
	for _, test := range []struct {
		text string
		want int
	}{
		{"1", 1}, {"65536", 65536}, {"64KiB", 65536}, {"2MiB", 2097152}, {"1GiB", 1073741824},
	} {
		got, ok := parseHeapLimit(test.text)
		if !ok || got != test.want {
			t.Errorf("%q: %d %v", test.text, got, ok)
		}
	}
	for _, text := range []string{"", "0", "-1", "+2", " 8", "4MB", "3kiB", "1KiBx", "18446744073709551616", "9223372036854775808", "99999999999TiB"} {
		if _, ok := parseHeapLimit(text); ok {
			t.Errorf("accepted %q", text)
		}
	}
	if got := allocationBudget(Options{MemoryStrategy: "desktop", Target: "linux/amd64", ArenaSize: 128 << 20}); got != 0 {
		t.Fatalf("implicit arena default leaked into heap: %d", got)
	}
	options := ParseOptions([]string{"-heap-limit", "1MiB", "-o", "app", "main.go"})
	if options.Ok || options.Error != ParseErrHeapLimit {
		t.Fatalf("builtin accepted heap limit: %#v", options)
	}
}
