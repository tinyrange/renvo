//go:build renvo_bundle

package driver

import (
	"testing"
	"testing/fstest"
)

func TestCHeaderFeatureProbes(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		valid        bool
	}{
		{"type", "#include <sys/types.h>\nint main(void){if(sizeof(size_t))return 0;return 0;}", true},
		{"type-is-not-value", "#include <sys/types.h>\nint main(void){if(sizeof((size_t)))return 0;return 0;}", false},
		{"missing-member", "#include <sys/stat.h>\nint main(void){struct stat s;return sizeof(s.st_birthtim.tv_nsec);}", false},
		{"assert-message", "_Static_assert(1,\"joined\" \"message\"); int main(void){return 0;}", true},
		{"false-assert", "_Static_assert(0,\"joined\" \"message\"); int main(void){return 0;}", false},
		{"invalid-assert-message", "_Static_assert(1,42); int main(void){return 0;}", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := memorySourceFS{files: fstest.MapFS{"probe.c": {Data: []byte(tc.source)}}}
			r, e := CompileCommand(&CommandRequest{Filesystem: files, Args: []string{"cc", "-O2", "-c", "probe.c"}, Target: "linux/amd64"})
			if e != nil || r.Ok != tc.valid {
				t.Fatalf("result=%+v error=%v", r, e)
			}
		})
	}
}
