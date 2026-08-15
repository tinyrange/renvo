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
	want := "obj/main.o: /repo/case/main.c /repo/include/api.h\n"
	if got := string(CDependencyOutput(result.Options)); got != want {
		t.Fatalf("dependency output = %q, want %q", got, want)
	}
}
