//go:build renvo_bundle

package driver

import (
	"testing"
	"testing/fstest"
)

func TestLibcLinkAutoconfPrototype(t *testing.T) {
	fs := memorySourceFS{files: fstest.MapFS{"probe.c": {Data: []byte("char fcntl(); int main(void){return fcntl();}")}}}
	r, e := CompileCommand(&CommandRequest{Filesystem: fs, Args: []string{"cc", "probe.c"}, Target: "linux/amd64"})
	if e != nil || !r.Ok {
		t.Fatalf("%+v %v", r, e)
	}
	t.Logf("image %d", len(r.Binary))
}

func TestLibcObjectResolutionBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		args         []string
		ok           bool
	}{
		{"libc", "extern int fcntl(int,int,...); int main(void){return fcntl(-1,1);}", nil, true},
		{"nostdlib", "extern int fcntl(int,int,...); int main(void){return fcntl(-1,1);}", []string{"-nostdlib"}, false},
		{"unknown", "extern int unknown_library_call(void); int main(void){return unknown_library_call();}", nil, false},
		{"user_definition", "int fcntl(int fd,int action,...){return 0;} int main(void){return fcntl(1,1);}", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := memorySourceFS{files: fstest.MapFS{"main.c": {Data: []byte(tc.source)}}}
			obj, e := CompileCommand(&CommandRequest{Filesystem: fs, Args: []string{"cc", "-c", "main.c"}, Target: "linux/amd64"})
			if e != nil || !obj.Ok {
				t.Fatalf("object: %+v %v", obj, e)
			}
			fs.files["main.o"] = &fstest.MapFile{Data: obj.Binary}
			args := append([]string{"cc", "main.o"}, tc.args...)
			r, e := CompileCommand(&CommandRequest{Filesystem: fs, Args: args, Target: "linux/amd64"})
			if e != nil {
				t.Fatal(e)
			}
			if r.Ok != tc.ok {
				t.Fatalf("link: %+v", r.Diagnostic)
			}
		})
	}
}
