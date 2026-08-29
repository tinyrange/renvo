package rtgformat

import (
	"bytes"
	"strings"
	"testing"

	"renvo.dev/internal/rtg"
)

func TestSourceFormatsMetadataAndEmbeddedGo(t *testing.T) {
	input := []byte(`definition   1
unit fixture
implements direct_emitter_v1

# Keep this comment.
go backend {
func helper( value int)int {if value>0{return value};return 0}
}

arch fixture {
  registers integer=[r0,
  r1]
  forms {
    binary=word32(destination:reg @0 + 5,source:reg @16)
  }
}

format fixture_image {
  segment text {
    file=[headers.start,data.file_start)
    memory=[headers.start,data.memory_start)
  }
  segment data {
    file=[data.file_start,data.file_end)
  }
  sections when ! stripped {
    text progbits alloc|execute align=16
  }
  offset=stripped ? 0:sections.offset
  relocation label { number=2 kind=pc_relative width=32 addend=-4 }
}

target fixture-test/test {
  family=direct
  capabilities=[one,two]
  operation write { emit=go helper }
  sequence render(out:emitter) - > bytes
}
`)
	want := []byte(`definition 1
unit fixture
implements direct_emitter_v1

# Keep this comment.
go backend {
	func helper(value int) int {
		if value > 0 {
			return value
		}
		return 0
	}
}

arch fixture {
	registers integer = [r0,
		r1]
	forms {
		binary = word32(destination:reg@0+5, source:reg@16)
	}
}

format fixture_image {
	segment text {
		file = [headers.start, data.file_start)
		memory = [headers.start, data.memory_start)
	}
	segment data {
		file = [data.file_start, data.file_end)
	}
	sections when !stripped {
		text progbits alloc | execute align = 16
	}
	offset = stripped ? 0 : sections.offset
	relocation label {
		number = 2
		kind = pc_relative
		width = 32
		addend = -4
	}
}

target fixture-test/test {
	family = direct
	capabilities = [one, two]
	operation write { emit = go helper }
	sequence render(out:emitter) -> bytes
}
`)
	got, err := Source(input, "fixture.rtg", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("formatted source mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
	again, err := Source(got, "fixture.rtg", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again, got) {
		t.Fatalf("formatter is not idempotent\nfirst:\n%s\nsecond:\n%s", got, again)
	}
}

func TestSourceFormatsRBEStandardLibraryGo(t *testing.T) {
	input := []byte(`definition 1
unit fixture
implements direct_emitter_v1

@stdlib "fixture/value.go"
package fixture
func Value( )int{return 1}
@endstdlib
`)
	want := []byte(`definition 1
unit fixture
implements direct_emitter_v1

@stdlib "fixture/value.go"
package fixture

func Value() int { return 1 }
@endstdlib
`)
	got, err := Source(input, "fixture.rbe", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("formatted RBE mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSourceValidatesRTGSyntax(t *testing.T) {
	_, err := Source([]byte("definition 1\nunit fixture\nimplements direct_emitter_v1\nunknown value\n"), "bad.rtg", nil)
	if err == nil || !strings.Contains(err.Error(), "RTG-PARSE-002") {
		t.Fatalf("expected RTG syntax diagnostic, got %v", err)
	}
}

func TestSourceValidatesEmbeddedGoSyntax(t *testing.T) {
	_, err := Source([]byte("definition 1\nunit fixture\nimplements direct_emitter_v1\ngo backend { func broken( }\n"), "bad.rtg", nil)
	if err == nil || !strings.Contains(err.Error(), "RTG-GO-005") {
		t.Fatalf("expected embedded Go syntax diagnostic, got %v", err)
	}
}

func TestSourceValidatesRBEStandardLibraryGoSyntax(t *testing.T) {
	input := []byte("definition 1\nunit fixture\nimplements direct_emitter_v1\n" +
		"@stdlib \"fixture/value.go\"\npackage fixture\nfunc broken( }\n@endstdlib\n")
	_, err := Source(input, "bad.rbe", nil)
	if err == nil || !strings.Contains(err.Error(), `@stdlib "fixture/value.go"`) {
		t.Fatalf("expected standard-library Go syntax diagnostic, got %v", err)
	}
}

func TestSourcePreservesMetadataComments(t *testing.T) {
	input := []byte("definition 1\nunit fixture\nimplements direct_emitter_v1\n" +
		"target fixture/test {\n  value=one /* inline */\n  /* block {\n     comment } */\n}\n")
	want := []byte("definition 1\nunit fixture\nimplements direct_emitter_v1\n" +
		"target fixture/test {\n\tvalue = one /* inline */\n\t/* block {\n\tcomment } */\n}\n")
	got, err := Source(input, "comments.rtg", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("metadata comments changed\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSourceFormatsImportedFragmentWithoutClosedHeader(t *testing.T) {
	input := []byte("go backend {\nfunc helper( )int{return 1}\n}\n")
	got, err := Source(input, "fragment.rtg", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte("func helper() int { return 1 }")) {
		t.Fatalf("embedded Go was not formatted:\n%s", got)
	}
}

func TestSourceResolvesImportsDuringValidation(t *testing.T) {
	input := []byte("@import \"fragment.rtg\"\n")
	loader := importLoader{files: map[string][]byte{
		"fragment.rtg": []byte("definition 1\nunit fixture\nimplements direct_emitter_v1\n"),
	}}
	if _, err := Source(input, "root.rtg", loader); err != nil {
		t.Fatal(err)
	}
	delete(loader.files, "fragment.rtg")
	if _, err := Source(input, "root.rtg", loader); err == nil || !strings.Contains(err.Error(), "RTG-IMPORT-005") {
		t.Fatalf("expected missing import diagnostic, got %v", err)
	}
}

type importLoader struct {
	files map[string][]byte
}

func (loader importLoader) LoadImport(_ string, path string) rtg.ImportSource {
	source, ok := loader.files[path]
	return rtg.ImportSource{Source: source, Filename: path, Ok: ok}
}
