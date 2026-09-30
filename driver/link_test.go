package driver

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"
)

func TestLinkCommandDiagnostics(t *testing.T) {
	fs := memorySourceFS{files: fstest.MapFS{
		"bad.o":         {Data: []byte("not ELF")},
		"recursive.rsp": {Data: []byte("@recursive.rsp")},
	}}
	for _, tc := range []struct {
		args         []string
		target, want string
	}{
		{nil, "", "no object files"},
		{[]string{"-o"}, "", "requires an output"},
		{[]string{"missing.o"}, "", "could not read object"},
		{[]string{"bad.o"}, "", "not a Linux/amd64 ELF"},
		{[]string{"-shared", "bad.o"}, "", "unsupported linker option"},
		{[]string{"-r", "bad.o"}, "", "unsupported linker option"},
		{[]string{"-o", "-", "bad.o"}, "", "stdout is unsupported"},
		{[]string{"@recursive.rsp"}, "", "could not expand response file"},
		{[]string{"bad.o"}, "windows/amd64", "only supports linux/amd64"},
	} {
		r, err := LinkCommand(&CommandRequest{Filesystem: fs, Args: tc.args, Target: tc.target})
		if err != nil || r.Ok || len(r.Outputs) != 0 || !strings.Contains(r.Diagnostic.Message, tc.want) {
			t.Fatalf("%v: result=%+v err=%v", tc.args, r, err)
		}
	}
	if _, err := LinkCommand(nil); err == nil {
		t.Fatal("nil request accepted")
	}
}

func TestLinkCommandDoesNotCompileSource(t *testing.T) {
	fs := memorySourceFS{files: fstest.MapFS{"input": {Data: []byte("int main(void) { return 0; }")}}}
	r, err := LinkCommand(&CommandRequest{Filesystem: fs, Args: []string{"input"}})
	if err != nil || r.Ok || !strings.Contains(r.Diagnostic.Message, "relocatable object") {
		t.Fatalf("%+v %v", r, err)
	}
	for _, arg := range []string{"-v", "--version"} {
		r, err = LinkCommand(&CommandRequest{Filesystem: fs, Args: []string{arg}})
		if err != nil || !r.Ok || len(r.Outputs) != 1 || !bytes.Equal(r.Outputs["-"], []byte(LinkerVersion)) {
			t.Fatalf("%+v %v", r, err)
		}
	}
}
