package driver

import (
	"testing"
	"testing/fstest"
)

func TestCompileCommandRelativeIncludeRoot(t *testing.T) {
	fs := memorySourceFS{files: fstest.MapFS{
		"src/probe.c": {Data: []byte("#include \"config.h\"\nint main(void) {return VALUE;}\n")},
		"config.h":    {Data: []byte("#define VALUE 42\n")},
	}}
	r, e := CompileCommand(&CommandRequest{Filesystem: fs, Args: []string{"cc", "-I.", "-c", "src/probe.c", "-o", "probe.o"}, Target: "linux/amd64"})
	if e != nil || !r.Ok || len(r.Outputs["probe.o"]) == 0 {
		t.Fatalf("%+v %v", r, e)
	}
}
func TestPreprocessIncludeNextSubdirectory(t *testing.T) {
	fs := memorySourceFS{files: fstest.MapFS{
		"probe.c":             {Data: []byte("#include <sys/types.h>\nVALUE\n")},
		"wrapper/sys/types.h": {Data: []byte("#include_next <sys/types.h>\n")},
		"system/sys/types.h":  {Data: []byte("#define VALUE 42\n")},
	}}
	r, e := CompileCommand(&CommandRequest{Filesystem: fs, Args: []string{"cc", "-E", "-P", "-nostdinc", "-Iwrapper", "-Isystem", "probe.c"}, Target: "linux/amd64"})
	if e != nil || !r.Ok || string(r.Outputs["-"]) != "42\n" {
		t.Fatalf("%+v %v output=%q", r, e, r.Outputs["-"])
	}
}
