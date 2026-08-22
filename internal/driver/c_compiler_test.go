package driver

import "testing"

type cCompilerResponseTestFS struct {
	files map[string][]byte
}

func (fs cCompilerResponseTestFS) ReadFile(path string) ([]byte, bool) {
	data, ok := fs.files[path]
	return data, ok
}

func (cCompilerResponseTestFS) ReadDir(string) ([]DirEntry, bool) { return nil, false }
func (cCompilerResponseTestFS) PathExists(string) bool            { return false }

func TestCCompilerIdentificationRequest(t *testing.T) {
	args := []string{"renvo", "cc", "-E", "-P", "-x", "c", "-"}
	request := InspectCCompilerRequest(args)
	if request.Kind != CCompilerRequestPreprocessStdin {
		t.Fatalf("request = %#v", request)
	}
	status, output := ExecuteCCompilerRequest(request, []byte("#if defined(__GNUC__)\nGCC __GNUC__ __GNUC_MINOR__ __GNUC_PATCHLEVEL__\n#endif\n"))
	if status != 0 || output != "GCC 5 1 0\n" {
		t.Fatalf("status/output = %d/%q", status, output)
	}
}

func TestUnknownCCompilerRequestFallsThrough(t *testing.T) {
	request := InspectCCompilerRequest([]string{"renvo", "cc", "-fmade-up", "-c", "/dev/null", "-o", "probe.o"})
	if request.Kind != CCompilerRequestNone {
		t.Fatalf("unknown compiler operation was intercepted: %#v", request)
	}
}

func TestUnknownPreprocessorProbeFallsThrough(t *testing.T) {
	request := InspectCCompilerRequest([]string{"renvo", "cc", "-E", "-fmade-up", "-P", "-x", "c", "-"})
	if request.Kind != CCompilerRequestNone {
		t.Fatalf("unknown preprocessor option was intercepted: %#v", request)
	}
}

func TestCCompilerAssemblerVersionRequest(t *testing.T) {
	request := InspectCCompilerRequest([]string{"renvo", "cc", "-Wa,--version", "-c", "-x", "assembler-with-cpp", "/dev/null", "-o", "/dev/null"})
	status, output := ExecuteCCompilerRequest(request, nil)
	if request.Kind != CCompilerRequestAssemblerVersion || status != 0 || output != "GNU assembler (GNU Binutils) 2.25\n" {
		t.Fatalf("assembler request/status/output = %#v/%d/%q", request, status, output)
	}
}

func TestCCompilerResponseFiles(t *testing.T) {
	files := map[string][]byte{
		"compile.rsp": []byte("-I 'kernel include' @defines.rsp -c leaf.c -o leaf.o"),
		"defines.rsp": []byte("-DNAME='Renvo Kernel'"),
	}
	result := ExpandCCompilerResponseFiles([]string{"renvo", "cc", "@compile.rsp"}, cCompilerResponseTestFS{files: files})
	if !result.Ok {
		t.Fatalf("response expansion failed: %#v", result)
	}
	want := []string{"renvo", "cc", "-I", "kernel include", "-DNAME=Renvo Kernel", "-c", "leaf.c", "-o", "leaf.o"}
	if len(result.Args) != len(want) {
		t.Fatalf("response args = %#v, want %#v", result.Args, want)
	}
	for i := range want {
		if result.Args[i] != want[i] {
			t.Fatalf("response args = %#v, want %#v", result.Args, want)
		}
	}
}

func TestCCompilerResponseRejectsTrailingEscape(t *testing.T) {
	result := ExpandCCompilerResponseFiles([]string{"renvo", "cc", "@bad.rsp"}, cCompilerResponseTestFS{files: map[string][]byte{"bad.rsp": []byte("-c source.c\\")}})
	if result.Ok {
		t.Fatalf("malformed response file expanded as %#v", result.Args)
	}
}
