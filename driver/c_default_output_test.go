package driver

import (
	"bytes"
	"debug/elf"
	"testing"
	"testing/fstest"
)

func TestCDefaultObjectOutput(t *testing.T) {
	for _, name := range []string{"conftest.c", "nested/source.i"} {
		fs := memorySourceFS{files: fstest.MapFS{name: {Data: []byte("int answer(void) { return 42; }")}}}
		result, err := CompileCommand(&CommandRequest{Filesystem: fs, Args: []string{"cc", "-c", name}, Target: "linux/amd64", ArenaSize: 1 << 20})
		if err != nil {
			t.Fatal(err)
		}
		want := "conftest.o"
		if name == "nested/source.i" {
			want = "source.o"
		}
		if !result.Ok || result.Output != want {
			t.Fatalf("result=%+v", result)
		}
		image, err := elf.NewFile(bytes.NewReader(result.Outputs[want]))
		if err != nil {
			t.Fatal(err)
		}
		if image.Type != elf.ET_REL {
			t.Fatalf("type=%v", image.Type)
		}
		image.Close()
	}
}
