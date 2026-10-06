//go:build !renvo

package driver

import (
	"bytes"
	"os"
	"testing"
	"testing/fstest"
)

func TestTargetFrontendVirtualBackendVocabulary(t *testing.T) {
	files := fstest.MapFS{}
	for _, name := range []string{"msdos.rtg", "bios_8086.rtg"} {
		source, err := os.ReadFile("../backends/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if name == "msdos.rtg" {
			// Neither public operation names nor target names are a frontend
			// dispatch table. An independent backend can choose both.
			source = bytes.Replace(source, []byte("move_immediate(destination:"), []byte("load_constant(destination:"), 1)
			source = bytes.ReplaceAll(source, []byte("= move_immediate("), []byte("= load_constant("))
			source = bytes.Replace(source, []byte("target msdos/8086"), []byte("target private/board"), 1)
		}
		files["definitions/"+name] = &fstest.MapFile{Data: source}
	}
	fs := memorySourceFS{files: files}
	v := ReadTargetVocabulary(files["definitions/msdos.rtg"].Data, "definitions/msdos.rtg", "private/board", commandImportLoader{fs})
	if !v.Ok || v.Target.Name != "private/board" || v.Target.WordBits != 16 {
		t.Fatalf("public vocabulary: %+v", v.Diagnostics)
	}
	blocks := []TargetBlock{{Name: "answer", Instructions: []TargetInstruction{
		{Operation: "load_constant", Operands: []TargetOperand{{Kind: "register", Value: "ax"}, {Kind: "int", Value: "42"}}},
		{Operation: "return"},
	}}}
	assembly, diagnostics := EncodeTargetAssembly(v, blocks, "answer.rtgasm")
	if len(diagnostics) != 0 {
		t.Fatalf("public frontend: %+v", diagnostics)
	}
	files["main.go"] = &fstest.MapFile{Data: []byte("package main\nfunc answer() int\nfunc appMain() int { return answer() }\n")}
	files["answer.rtgasm"] = &fstest.MapFile{Data: assembly}
	result, err := CompileCommand(&CommandRequest{Filesystem: fs, Target: "private/board", Args: []string{"-backend", "definitions/msdos.rtg", "-s", "-o", "image.com", "main.go", "answer.rtgasm"}})
	if err != nil || !result.Ok || !bytes.Contains(result.Binary, []byte{0xb8, 42, 0, 0xc3}) {
		t.Fatalf("public compile: %v %+v", err, result)
	}
}
