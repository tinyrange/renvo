//go:build renvo_bundle

package driver

import (
	"testing"
	"testing/fstest"
)

func TestLibcHeaderlessConfigureProbes(t *testing.T) {
	for _, n := range []string{"mbrtowc", "setlocale", "getenv", "malloc", "printf", "getrlimit"} {
		t.Run(n, func(t *testing.T) {
			f := memorySourceFS{files: fstest.MapFS{"probe.c": {Data: []byte("char " + n + "(); int main(void){return " + n + "();}")}}}
			r, e := CompileCommand(&CommandRequest{Filesystem: f, Args: []string{"cc", "probe.c"}, Target: "linux/amd64"})
			if e != nil {
				t.Fatal(e)
			}
			if !r.Ok {
				t.Fatalf("%+v", r.Diagnostic)
			}
		})
	}
}
