package rtg

import (
	"crypto/sha256"
	goparser "go/parser"
	"go/token"
	"testing"
)

func TestSHA256MatchesStandardLibrary(t *testing.T) {
	inputs := [][]byte{
		nil,
		[]byte("abc"),
		[]byte("a definition long enough to exercise more than one SHA-256 block................................................................"),
	}
	for i := 0; i < len(inputs); i++ {
		got := sha256Bytes(inputs[i])
		want := sha256.Sum256(inputs[i])
		if got != want {
			t.Fatalf("sha256Bytes(%q) = %s, want %x", inputs[i], HashText(got), want)
		}
	}
}

func TestParseSystem(t *testing.T) {
	source := []byte(`# ordinary hosted profile
system "small-linux-amd64" {
	target = "linux/amd64"
	binary = 2MiB
	arena = 32MiB
}`)
	profile, diagnostic, ok := ParseSystem(source, "small.rtg")
	if !ok {
		t.Fatalf("ParseSystem failed: %#v", diagnostic)
	}
	if profile.Name != "small-linux-amd64" || profile.Target != "linux/amd64" ||
		profile.BinaryLimit != 2*1024*1024 || profile.ArenaSize != 32*1024*1024 {
		t.Fatalf("profile = %#v", profile)
	}
}

func TestCanonicalIgnoresFormattingCommentsAndTopLevelOrder(t *testing.T) {
	left := Parse([]byte(`definition 1
unit demo
implements direct_emitter_v1
arch a { width = 64 }
target linux/demo { arch = a }
go backend {
	func emit(value int) int { return value + 1 }
}`), "left.rtg")
	right := Parse([]byte(`// moved declarations and changed formatting
definition 01
unit demo
implements direct_emitter_v1
go backend { func emit(value int) int {
	/* irrelevant */ return value+1
} }
target linux/demo {
	arch=a
}
arch a { width=064 }
`), "right.rtg")
	if !left.Ok || !right.Ok {
		t.Fatalf("parse failures: left=%#v right=%#v", left.Diagnostics, right.Diagnostics)
	}
	if left.Hash != right.Hash {
		t.Fatalf("format-only changes changed identity: %s != %s", HashText(left.Hash), HashText(right.Hash))
	}
}

func TestCanonicalIncludesExecutableGo(t *testing.T) {
	base := `definition 1
unit demo
implements direct_emitter_v1
arch a { width = 64 }
go backend { func emit(value int) int { return value + %s } }`
	left := Parse([]byte(replaceOnce(base, "%s", "1")), "same.rtg")
	right := Parse([]byte(replaceOnce(base, "%s", "2")), "same.rtg")
	if !left.Ok || !right.Ok {
		t.Fatalf("parse failures: left=%#v right=%#v", left.Diagnostics, right.Diagnostics)
	}
	if left.Hash == right.Hash {
		t.Fatal("executable embedded-Go change did not change identity")
	}
}

func TestEmbeddedGoRestrictions(t *testing.T) {
	tests := []struct {
		source string
		code   string
	}{
		{source: `go backend { import "fmt" }`, code: "RTG-GO-006"},
		{source: "go backend {\n//go:noinline\nfunc emit() {}\n}", code: "RTG-GO-004"},
		{source: `go backend { func RTGEmit() {} }`, code: "RTG-GO-007"},
	}
	prefix := "definition 1\nunit demo\nimplements direct_emitter_v1\narch a {}\n"
	for i := 0; i < len(tests); i++ {
		document := Parse([]byte(prefix+tests[i].source), "bad.rtg")
		if document.Ok || len(document.Diagnostics) == 0 || document.Diagnostics[0].Code != tests[i].code {
			t.Errorf("Parse case %d = ok %v diagnostics %#v, want %s", i, document.Ok, document.Diagnostics, tests[i].code)
		}
	}
}

func TestTruncatedBlocksFail(t *testing.T) {
	document := Parse([]byte("definition 1\nunit demo\nimplements direct_emitter_v1\narch a { width = 64"), "truncated.rtg")
	if document.Ok || len(document.Diagnostics) == 0 || document.Diagnostics[0].Code != "RTG-PARSE-014" {
		t.Fatalf("Parse = ok %v diagnostics %#v", document.Ok, document.Diagnostics)
	}
}

const testMachineDefinition = `definition 1
unit tiny
implements direct_emitter_v1

go backend {
	func addOne(value int) int { return value + 1 }
	func addTwo(value int) int { return addOne(addOne(value)) }
}

arch tiny64 {
	alias = "tiny"
	endian = little
	word_bits = 64
	pointer_bits = 64
}
abi tiny_abi { arch = tiny64 }
runtime tiny_runtime { operation print { builtin = true } }
format tiny_image { address_bits = 64 }
target test/tiny64 {
	arch = tiny64
	abi = tiny_abi
	runtime = tiny_runtime
	executable = tiny_image
	aliases = ["test/tiny"]
	build_tags = ["test", "tiny64"]
	capabilities = [hosted, executable]
}
`

func TestResolveAndGenerateFixedBackend(t *testing.T) {
	document := Parse([]byte(testMachineDefinition), "tiny.rtg")
	resolved := ResolveDefinitions(document)
	if !resolved.Ok {
		t.Fatalf("Resolve failed: %#v", resolved.Diagnostics)
	}
	if len(resolved.Targets) != 1 {
		t.Fatalf("targets = %#v", resolved.Targets)
	}
	descriptor := resolved.Targets[0].Descriptor
	if descriptor.Name != "test/tiny64" || descriptor.OS != "test" || descriptor.ISA != "tiny" ||
		descriptor.WordBits != 64 || descriptor.PointerBits != 64 || descriptor.Endian != "little" ||
		descriptor.Executable != "tiny_image" {
		t.Fatalf("descriptor = %#v", descriptor)
	}
	generated := GenerateFixedBackend(resolved, "test/tiny")
	if !generated.Ok {
		t.Fatalf("GenerateFixedBackend failed: %#v", generated.Diagnostics)
	}
	if _, err := goparser.ParseFile(token.NewFileSet(), "generated.go", generated.Source, goparser.AllErrors); err != nil {
		t.Fatalf("generated source does not parse: %v\n%s", err, generated.Source)
	}
	for _, want := range []string{"// target: test/tiny64", "func rtgTinyAddOne", "rtgTinyAddOne(rtgTinyAddOne(value))"} {
		if !containsText(string(generated.Source), want) {
			t.Errorf("generated source missing %q:\n%s", want, generated.Source)
		}
	}
	second := GenerateFixedBackend(resolved, "test/tiny64")
	if string(generated.Source) != string(second.Source) {
		t.Fatal("fixed generation is not deterministic")
	}
}

func TestGenerateUniversalSortsManifestAndTargets(t *testing.T) {
	first := Resolve(Parse([]byte(testMachineDefinition), "tiny.rtg"))
	secondSource := replaceOnce(testMachineDefinition, "unit tiny", "unit other")
	secondSource = replaceOnce(secondSource, "test/tiny64", "other/tiny64")
	secondSource = replaceOnce(secondSource, `"test/tiny"`, `"other/tiny"`)
	second := Resolve(Parse([]byte(secondSource), "other.rtg"))
	generated := GenerateUniversalBackend([]ResolveResult{second, first})
	if !generated.Ok {
		t.Fatalf("GenerateUniversalBackend failed: %#v", generated.Diagnostics)
	}
	if len(generated.Manifest) != 2 || generated.Manifest[0][:5] != "other" {
		t.Fatalf("manifest = %#v", generated.Manifest)
	}
	if _, err := goparser.ParseFile(token.NewFileSet(), "generated.go", generated.Source, goparser.AllErrors); err != nil {
		t.Fatalf("universal source does not parse: %v\n%s", err, generated.Source)
	}
}

func TestGeneratePreparedPreservesCompilerSymbols(t *testing.T) {
	resolved := Resolve(Parse([]byte(testMachineDefinition), "tiny.rtg"))
	generated := GeneratePreparedBackend(resolved, "test/tiny64")
	if !generated.Ok {
		t.Fatalf("GeneratePreparedBackend failed: %#v", generated.Diagnostics)
	}
	text := string(generated.Source)
	for _, want := range []string{"package main", "var RTGTarget", "func addOne", "addOne(addOne(value))"} {
		if !containsText(text, want) {
			t.Errorf("prepared source missing %q:\n%s", want, text)
		}
	}
	if containsText(text, "rtgTinyAddOne") {
		t.Fatalf("prepared generation unexpectedly rewrote isolated symbol:\n%s", text)
	}
}
func TestResolveRejectsUnknownCompositionReference(t *testing.T) {
	source := replaceOnce(testMachineDefinition, "abi = tiny_abi", "abi = missing")
	resolved := Resolve(Parse([]byte(source), "bad.rtg"))
	if resolved.Ok || len(resolved.Diagnostics) == 0 || resolved.Diagnostics[0].Code != "RTG-RESOLVE-010" {
		t.Fatalf("Resolve = ok %v diagnostics %#v", resolved.Ok, resolved.Diagnostics)
	}
}

func replaceOnce(text string, old string, replacement string) string {
	for i := 0; i+len(old) <= len(text); i++ {
		if text[i:i+len(old)] == old {
			return text[:i] + replacement + text[i+len(old):]
		}
	}
	return text
}

func containsText(text string, fragment string) bool {
	for i := 0; i+len(fragment) <= len(text); i++ {
		if text[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
