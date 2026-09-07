package load

import "testing"

func TestGoReleaseTags(t *testing.T) {
	for _, tc := range []struct {
		tag  string
		want bool
	}{
		{"go1", false}, {"go1.1", true}, {"go1.9", true}, {"go1.25", true},
		{"go1.26", false}, {"go1.100", false}, {"go1.9999999999999999999999999999", false},
		{"go1.0", false}, {"go1.025", false}, {"go1.25.0", false}, {"go1.25rc1", false},
		{"go2", false}, {"go2.1", false}, {"go1.", false}, {"go", false}, {"linux", false},
	} {
		if got := HasGoReleaseTag(tc.tag); got != tc.want {
			t.Errorf("%q: %v, want %v", tc.tag, got, tc.want)
		}
	}
}
