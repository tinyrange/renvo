package driver

import (
	"testing"
	"testing/fstest"
)

func TestCObjectBuiltinName(t *testing.T) {
	fs := memorySourceFS{files: fstest.MapFS{"probe.c": {Data: []byte("extern int close(int); static int helper(int fd) {int result=fd; close(result); return 0;} int probe(int fd) {return helper(fd);}")}}}
	r, e := CompileCommand(&CommandRequest{Filesystem: fs, Args: []string{"cc", "-c", "probe.c"}, Target: "linux/amd64"})
	if e != nil {
		t.Fatal(e)
	}
	if !r.Ok {
		t.Fatalf("%+v", r.Diagnostic)
	}
}
