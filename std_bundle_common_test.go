//go:build renvo_bundle

package renvo

import "testing"

func TestBundledStdHostOnlyLineEndings(t *testing.T) {
	for _, source := range []string{
		"//go:build !renvo\n\npackage bytes\n",
		"//go:build !renvo\r\n\r\npackage bytes\r\n",
	} {
		if !bundledStdHostOnly([]byte(source)) {
			t.Fatalf("host-only constraint not recognized in %q", source)
		}
	}
	if bundledStdHostOnly([]byte("//go:build renvo\n\npackage bytes\n")) {
		t.Fatal("Renvo source was classified as host-only")
	}
}
