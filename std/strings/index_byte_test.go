package strings

import "testing"

func TestIndexByte(t *testing.T) {
	for _, tc := range []struct {
		s    string
		b    byte
		want int
	}{
		{"", 0, -1}, {"abc", 'b', 1}, {"abc", 'x', -1}, {"\x00\xff\x00", 0, 0}, {"\x00\xff\x00", 255, 1},
	} {
		if got := IndexByte(tc.s, tc.b); got != tc.want {
			t.Fatalf("IndexByte(%q, %d)=%d, want %d", tc.s, tc.b, got, tc.want)
		}
	}
}
