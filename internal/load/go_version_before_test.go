package load

import "testing"

func TestGoVersionBefore(t *testing.T) {
	for _, tc := range []struct {
		version string
		want    bool
	}{
		{"", false}, {"bad", false}, {"1.9", true}, {"1.25.99", true},
		{"1.26rc1", false}, {"1.26.0", false}, {"1.100", false},
		{"999999999999999999999999.1", false},
	} {
		if got := GoVersionBefore(tc.version, "1.26"); got != tc.want {
			t.Errorf("%q: %v", tc.version, got)
		}
	}
}
