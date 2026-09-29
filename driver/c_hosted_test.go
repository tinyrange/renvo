package driver

import (
	"fmt"
	"testing"
	"testing/fstest"
)

func TestCCompilerHostedModes(t *testing.T) {
	for _, tc := range []struct {
		flags  []string
		hosted int
	}{{nil, 1}, {[]string{"-ffreestanding"}, 0}, {[]string{"-ffreestanding", "-fhosted"}, 1}, {[]string{"-fhosted", "-ffreestanding"}, 0}} {
		source := fmt.Sprintf(`_Static_assert(__STDC_HOSTED__ == %d, "hosted"); _Static_assert(__GNUC_STDC_INLINE__ == 1, "inline"); int value;`, tc.hosted)
		fs := memorySourceFS{files: fstest.MapFS{"main.c": {Data: []byte(source)}}}
		args := append([]string{"cc", "-c", "main.c"}, tc.flags...)
		result, err := CompileCommand(&CommandRequest{Filesystem: fs, Args: args, Target: "linux/amd64"})
		if err != nil || !result.Ok {
			t.Fatalf("%v: %v %+v", tc.flags, err, result)
		}
	}
}
