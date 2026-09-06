package driver

import (
	"bytes"
	"debug/elf"
	"testing"
	"testing/fstest"
)

func TestCompileCCommands(t *testing.T) {
	for _, name := range []string{"main.c", "main.i", "generated/main.c", "generated/main.i"} {
		t.Run(name, func(t *testing.T) {
			fs := memorySourceFS{files: fstest.MapFS{name: &fstest.MapFile{Data: []byte("int main(void) { int total = 0; for (int i = 0; i < 42; i++) total++; return total; }\n")}}}
			r, err := CompileCommand(&CommandRequest{Filesystem: fs, Args: []string{"cc", name}, Target: "windows/386", ArenaSize: 1 << 20})
			if err != nil {
				t.Fatal(err)
			}
			if !r.Ok {
				t.Fatalf("%+v", r.Diagnostic)
			}
			if !bytes.HasPrefix(r.Binary, []byte("MZ")) {
				t.Fatal("missing Windows executable")
			}
		})
	}
}

func TestPreprocessAndCompileInMemory(t *testing.T) {
	fs := memorySourceFS{files: fstest.MapFS{
		"main.c":          {Data: []byte("#include \"value.h\"\nint main(void) { return VALUE; }\n")},
		"include/value.h": {Data: []byte("#define VALUE 42\n")},
		"flags.rsp":       {Data: []byte("-E -P -Iinclude -MMD -MF main.d main.c -o main.i")},
	}}
	r, err := CompileCommand(&CommandRequest{Filesystem: fs, Args: []string{"cc", "@flags.rsp"}, Target: "windows/386"})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Ok || !bytes.Contains(r.Binary, []byte("return 42")) || !bytes.Contains(r.Outputs["main.d"], []byte("include/value.h")) {
		t.Fatalf("preprocessing failed: %+v", r)
	}
	fs.files["main.i"] = &fstest.MapFile{Data: r.Binary}
	r, err = CompileCommand(&CommandRequest{Filesystem: fs, Args: []string{"cc", "-DVALUE=99", "main.i"}, Target: "windows/386", ArenaSize: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Ok || !bytes.HasPrefix(r.Binary, []byte("MZ")) {
		t.Fatalf("preprocessed compilation failed: %+v", r.Diagnostic)
	}
}

func TestCompileAndLinkObjectsInMemory(t *testing.T) {
	fs := memorySourceFS{files: fstest.MapFS{
		"main.c":   {Data: []byte("extern int answer(void); int main(void) { return answer(); }\n")},
		"answer.c": {Data: []byte("int answer(void) { return 42; }\n")},
	}}
	for _, name := range []string{"main", "answer"} {
		r, err := CompileCommand(&CommandRequest{Filesystem: fs, Args: []string{"cc", "-c", name + ".c", "-o", name + ".o"}, Target: "linux/amd64", ArenaSize: 1 << 20})
		if err != nil {
			t.Fatal(err)
		}
		if !r.Ok {
			t.Fatalf("object compilation: %+v", r.Diagnostic)
		}
		fs.files[name+".o"] = &fstest.MapFile{Data: r.Binary}
	}
	r, err := CompileCommand(&CommandRequest{Filesystem: fs, Args: []string{"cc", "main.o", "answer.o", "-o", "app"}, Target: "linux/amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Ok {
		t.Fatalf("object link: %+v", r.Diagnostic)
	}
	image, err := elf.NewFile(bytes.NewReader(r.Binary))
	if err != nil {
		t.Fatal(err)
	}
	defer image.Close()
	if image.Type != elf.ET_EXEC || image.Entry == 0 {
		t.Fatalf("invalid linked ELF: %+v", image.FileHeader)
	}
}

func TestPlanMakeDependenciesAndRebuild(t *testing.T) {
	source := []byte("CC := renvo cc\n.PHONY: all\nall: app\napp: main.o\n\t$(CC) $^ -o $@\nmain.o: main.c\n\t$(CC) -c $< -o $@\n")
	commands, err := PlanMake(source, nil, func(name string) bool { return name == "main.c" || name == "main.o" || name == "app" })
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 2 || commands[0].Target != "main.o" || commands[1].Target != "app" {
		t.Fatalf("incorrect dependency plan: %+v", commands)
	}
}

func TestPreprocessedSourceSkipsMacroExpansion(t *testing.T) {
	fs := memorySourceFS{files: fstest.MapFS{
		"main.i": {Data: []byte("int VALUE = 42; int main(void) { return VALUE; }\n")},
	}}
	r, err := CompileCommand(&CommandRequest{Filesystem: fs, Args: []string{"cc", "-DVALUE=99", "main.i"}, Target: "windows/386", ArenaSize: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Ok {
		t.Fatalf("preprocessed input was expanded again: %+v", r.Diagnostic)
	}
}
