package rbe

import (
	"bytes"
	"testing"
)

func TestParsePlainRTGUnchanged(t *testing.T) {
	source := []byte("definition 1\nunit tiny\nimplements direct_emitter_v1\n")
	bundle := Parse(source)
	if !bundle.Ok || len(bundle.Files) != 0 || !bytes.Equal(bundle.Definition, source) {
		t.Fatalf("Parse = %#v", bundle)
	}
}

func TestParseLibraryAdditions(t *testing.T) {
	source := []byte("definition 1\nunit tiny\nimplements direct_emitter_v1\n" +
		"@stdlib \"syscall/v7.go\"\npackage syscall\nconst V7 = true\n@endstdlib\n" +
		"@stdlib \"syscall/v7.rtgasm\"\nrtgasm 1\n@endstdlib\n")
	bundle := Parse(source)
	if !bundle.Ok || len(bundle.Files) != 2 {
		t.Fatalf("Parse = %#v", bundle)
	}
	if string(bundle.Definition) != "definition 1\nunit tiny\nimplements direct_emitter_v1\n" {
		t.Fatalf("definition = %q", bundle.Definition)
	}
	if bundle.Files[0].Path != "syscall/v7.go" || string(bundle.Files[0].Source) != "package syscall\nconst V7 = true\n" {
		t.Fatalf("first file = %#v", bundle.Files[0])
	}
}

func TestParseRejectsUnsafeDuplicateAndUnterminatedSections(t *testing.T) {
	tests := []string{
		"@stdlib \"../escape.go\"\nx\n@endstdlib\n",
		"@stdlib \"os/a.go\"\nx\n@endstdlib\n@stdlib \"os/a.go\"\ny\n@endstdlib\n",
		"@stdlib \"os/a.go\"\nx\n",
		"@stdlib os/a.go\nx\n@endstdlib\n",
	}
	for _, source := range tests {
		if result := Parse([]byte(source)); result.Ok || result.Message == "" {
			t.Fatalf("Parse(%q) = %#v", source, result)
		}
	}
}

func TestValidLibraryPath(t *testing.T) {
	valid := []string{"os/file.go", "net/http/transport_renvo.go"}
	for _, path := range valid {
		if !ValidLibraryPath(path) {
			t.Errorf("ValidLibraryPath(%q) = false", path)
		}
	}
	invalid := []string{"", "file.go", "/os/file.go", "../file.go", "os/../file.go", "os\\file.go", "os//file.go"}
	for _, path := range invalid {
		if ValidLibraryPath(path) {
			t.Errorf("ValidLibraryPath(%q) = true", path)
		}
	}
}
