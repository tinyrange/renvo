package driver

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestBuildCObjectReadsExplicitSystemHeader(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/case/main.c", Src: []byte("#include <stdio.h>\nint main(void) { puts(\"PASS\"); return 0; }\n")},
		{Path: "/sdk/include/stdio.h", Src: []byte("extern int remove(const char *__name);\nextern int puts(const char *__s);\n")},
	}}
	result := BuildFromFS([]string{"-c", "-isystem", "/sdk/include", "-o", "main.o", "main.c"}, "/repo/case", "/std", fs)
	if !result.Ok {
		t.Fatalf("BuildFromFS hosted C object failed: %#v", result)
	}
	found := false
	for i := 0; i < len(result.Pipeline.Workspace.Files); i++ {
		file := result.Pipeline.Workspace.Files[i]
		if file.Path == "/repo/case/main.c" && file.CObject && bytes.Contains(file.CPrelude, []byte("extern int puts")) && !bytes.Contains(file.CPrelude, []byte("remove")) {
			found = true
		}
	}
	if !found {
		t.Fatalf("workspace did not retain selected C header declaration: %#v", result.Pipeline.Workspace.Files)
	}
}

func TestBuildFreestandingCObjectRetainsForcedAndKernelHeaders(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/case/main.c", Src: []byte("#include <kernel.h>\nforced_word value = KERNEL_VALUE;\n")},
		{Path: "/repo/include/forced.h", Src: []byte("typedef unsigned long forced_word;\n#define FORCED_VALUE 40\n")},
		{Path: "/repo/include/kernel.h", Src: []byte("#define KERNEL_VALUE (FORCED_VALUE + 2)\n")},
	}}
	args := NormalizeCCompilerCommand([]string{
		"renvo", "cc", "-nostdinc", "-I/repo/include", "-include", "/repo/include/forced.h",
		"-c", "main.c", "-o", "main.o",
	})
	result := BuildFromFS(args[1:], "/repo/case", "/std", fs)
	if !result.Ok {
		t.Fatalf("BuildFromFS freestanding C object failed: %#v", result)
	}
	found := false
	for i := 0; i < len(result.Pipeline.Workspace.Files); i++ {
		file := result.Pipeline.Workspace.Files[i]
		if file.Path != "/repo/case/main.c" {
			continue
		}
		found = true
		if len(file.CPrelude) != 0 || !bytes.Contains(file.Src, []byte("typedef unsigned long forced_word")) ||
			!bytes.Contains(file.Src, []byte("forced_word value = ( 40 + 2 )")) {
			t.Fatalf("freestanding source boundary = prelude %q, source %q", file.CPrelude, file.Src)
		}
	}
	if !found {
		t.Fatalf("workspace did not retain freestanding C source: %#v", result.Pipeline.Workspace.Files)
	}
}

func TestBuildCObjectCarriesUnsignedCharSemantics(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{{
		Path: "/repo/case/main.c",
		Src:  []byte("#ifndef __CHAR_UNSIGNED__\n#error unsigned char macro missing\n#endif\nint classify(char value) { return value < 0; }\n"),
	}}}
	args := NormalizeCCompilerCommand([]string{"renvo", "cc", "-nostdinc", "-funsigned-char", "-c", "main.c", "-o", "main.o"})
	result := BuildFromFS(args[1:], "/repo/case", "/std", fs)
	if !result.Ok {
		t.Fatalf("unsigned-char C object failed: %#v", result)
	}
	for i := 0; i < len(result.Pipeline.Workspace.Files); i++ {
		file := result.Pipeline.Workspace.Files[i]
		if file.Path == "/repo/case/main.c" {
			if !file.CUnsignedChar || !bytes.Contains(file.Src, []byte("int classify")) {
				t.Fatalf("unsigned-char source metadata = %#v, source %q", file, file.Src)
			}
			return
		}
	}
	t.Fatal("unsigned-char source missing from workspace")
}

func TestBuildCObjectReportsMissingHeader(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{{Path: "/repo/case/main.c", Src: []byte("#include <missing.h>\nint main(void) { return 0; }\n")}}}
	result := BuildFromFS([]string{"-c", "-o", "main.o", "main.c"}, "/repo/case", "/std", fs)
	if result.Ok || result.Sources.Error != SourceErrCInclude || result.Diagnostic.Phase != "preprocessor" || result.Diagnostic.Code != "RENVO-CPP-001" || result.Diagnostic.Line != 1 {
		t.Fatalf("missing C header result = %#v", result)
	}
}

func TestBuildCObjectReportsPreprocessorFailure(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{{Path: "/repo/case/main.c", Src: []byte("#if 1 / 0\nint main(void) { return 0; }\n#endif\n")}}}
	result := BuildFromFS([]string{"-c", "-o", "main.o", "main.c"}, "/repo/case", "/std", fs)
	if result.Ok || result.Sources.Error != SourceErrCPreprocess || result.Diagnostic.Phase != "preprocessor" ||
		result.Diagnostic.Code != "RENVO-CPP-002" || result.Diagnostic.Line != 1 {
		t.Fatalf("C preprocessor failure result = %#v", result)
	}
}

func TestBuildCObjectCapturesMakeDependencies(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/case/main.c", Src: []byte("#include <api.h>\nint main(void) { return value(); }\n")},
		{Path: "/repo/include/api.h", Src: []byte("int value(void);\n")},
	}}
	args := NormalizeCCompilerCommand([]string{"renvo", "cc", "-Wp,-MMD,obj/.main.o.d", "-I/repo/include", "-c", "main.c", "-o", "obj/main.o"})
	result := BuildFromFS(args[1:], "/repo/case", "/std", fs)
	if !result.Ok {
		t.Fatalf("dependency build failed: %#v", result)
	}
	want := "obj/main.o: main.c /repo/include/api.h\n"
	if got := string(CDependencyOutput(result.Options)); got != want {
		t.Fatalf("dependency output = %q, want %q", got, want)
	}
}
