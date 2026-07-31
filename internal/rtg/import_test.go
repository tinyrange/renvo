package rtg

import "testing"

func TestParseImportsClosesNestedDefinitionWithoutChangingIdentity(t *testing.T) {
	root := []byte(`definition 1
unit tiny
implements direct_emitter_v1
@import "arch.rtg"
@import "target.rtg"
`)
	files := map[string][]byte{
		"/defs/arch.rtg": []byte(`arch tiny64 {
	endian = little
	word_bits = 64
	reject = [move]
}`),
		"/defs/target.rtg": []byte(`@import "runtime.rtg"
abi tiny_abi { arch = tiny64 }
format tiny_image { address_bits = 64 }
target tiny/64 {
	family = native_v1
	os = linux
	arch = tiny64
	abi = tiny_abi
	runtime = tiny_runtime
	executable = tiny_image
}`),
		"/defs/runtime.rtg": []byte(`runtime tiny_runtime { operations = [print] }`),
	}
	imported := ParseImports(root, "/defs/root.rtg", importTestLoader{files: files})
	if !imported.Ok {
		t.Fatalf("ParseImports failed: %#v", imported.Diagnostics)
	}
	closed := Parse([]byte(`definition 1
unit tiny
implements direct_emitter_v1
arch tiny64 {
	endian = little
	word_bits = 64
	reject = [move]
}
runtime tiny_runtime { operations = [print] }
abi tiny_abi { arch = tiny64 }
format tiny_image { address_bits = 64 }
target tiny/64 {
	family = native_v1
	os = linux
	arch = tiny64
	abi = tiny_abi
	runtime = tiny_runtime
	executable = tiny_image
}`), "closed.rtg")
	if !closed.Ok {
		t.Fatalf("closed fixture failed: %#v", closed.Diagnostics)
	}
	if imported.Hash != closed.Hash {
		t.Fatalf("imports changed semantic identity: %x != %x", imported.Hash, closed.Hash)
	}
	if _, ok := imported.Declaration(DeclRuntime, "tiny_runtime"); !ok {
		t.Fatal("nested import did not contribute declarations")
	}
}

func TestParseImportsReportsImportedSourceLocation(t *testing.T) {
	root := []byte("definition 1\nunit tiny\nimplements direct_emitter_v1\n@import \"arch.rtg\"\n")
	imported := []byte("arch tiny64 {\n\tunknown_fact = 1\n}\n")
	document := ParseImports(root, "/defs/root.rtg", fixedImportLoader{
		source: imported, filename: "/defs/arch.rtg",
	})
	resolved := Resolve(document)
	if resolved.Ok || len(resolved.Diagnostics) == 0 {
		t.Fatal("invalid imported declaration resolved")
	}
	diagnostic := resolved.Diagnostics[0]
	if diagnostic.Filename != "/defs/arch.rtg" ||
		diagnostic.Span.Start.Line != 2 ||
		diagnostic.Span.Start.Column != 2 {
		t.Fatalf("import diagnostic = %#v", diagnostic)
	}
}

func TestParseImportsProvidesFileScopedHelperPackages(t *testing.T) {
	root := []byte(`definition 1
unit tiny
implements direct_emitter_v1
@import "alpha.rtg"
@import "beta.rtg"
`)
	files := map[string][]byte{
		"/defs/alpha.rtg": []byte(`go backend {
	func encode(out *RTGEmitter) { out.Byte(1) }
	func encodeFacts(out *RTGEmitter, register RTGRegister, condition RTGCondition) {}
}
arch alpha {
	registers = [r0]
	conditions {
		eq = 0
	}
	sequences {
		emit(out:emitter) {
			encode(out)
			beta.encode(out)
		}
		emitFacts(out:emitter) {
			encodeFacts(out,r0,eq)
		}
	}
	exports {
		publicAlpha = sequence emit
	}
}`),
		"/defs/beta.rtg": []byte(`go backend {
	func encode(out *RTGEmitter) { out.Byte(2) }
}
arch beta {
	sequences {
		emit(out:emitter) {
			call encode(out)
		}
	}
}`),
	}
	document := ParseImports(root, "/defs/root.rtg", importTestLoader{files: files})
	if !document.Ok {
		t.Fatalf("ParseImports failed: %#v", document.Diagnostics)
	}
	alpha, ok := document.Declaration(DeclArch, "alpha")
	if !ok {
		t.Fatal("alpha architecture is missing")
	}
	sequences := architectureSequences(alpha)
	if len(sequences) != 2 || sequences[0].Name != "alphaPackageEmit" {
		t.Fatalf("alpha sequences = %#v", sequences)
	}
	if got := sequences[0].Steps[0].Tokens; len(got) != 4 ||
		got[0] != "alphaPackageEncode" {
		t.Fatalf("local call = %#v", got)
	}
	if got := sequences[0].Steps[1].Tokens; len(got) != 4 ||
		got[0] != "betaPackageEncode" {
		t.Fatalf("qualified call = %#v", got)
	}
	if got := sequences[1].Steps[0].Tokens; len(got) != 8 ||
		got[0] != "alphaPackageEncodeFacts" ||
		got[4] != "alphaR0" || got[6] != "alphaEQ" {
		t.Fatalf("architecture facts call = %#v", got)
	}
	exports := architectureExports(alpha)
	if len(exports) != 1 || exports[0].External != "publicAlpha" ||
		exports[0].Local != "alphaPackageEmit" {
		t.Fatalf("exports = %#v", exports)
	}
	names := embeddedGoFunctionNames(document)
	for _, want := range []string{
		"alphaPackageEncode", "alphaPackageEncodeFacts", "betaPackageEncode",
	} {
		if stringIndex(names, want) < 0 {
			t.Fatalf("embedded names %v do not contain %s", names, want)
		}
	}
}

func TestParseImportsRejectsVirtualPackageCollision(t *testing.T) {
	root := []byte(`definition 1
unit tiny
implements direct_emitter_v1
@import "one/shared.rtg"
@import "two/shared.rtg"
`)
	files := map[string][]byte{
		"/defs/one/shared.rtg": []byte(`arch one { endian = little }`),
		"/defs/two/shared.rtg": []byte(`arch two { endian = little }`),
	}
	document := ParseImports(root, "/defs/root.rtg", importTestLoader{files: files})
	if document.Ok || len(document.Diagnostics) != 1 ||
		document.Diagnostics[0].Code != "RTG-IMPORT-007" {
		t.Fatalf("package collision diagnostics = %#v", document.Diagnostics)
	}
}

func TestParseImportsRejectsCycleAtImportSite(t *testing.T) {
	root := []byte("@import \"child.rtg\"\n")
	document := ParseImports(root, "/defs/root.rtg", cycleImportLoader{root: root})
	if document.Ok || len(document.Diagnostics) != 1 {
		t.Fatalf("cycle parse = %#v", document)
	}
	diagnostic := document.Diagnostics[0]
	if diagnostic.Code != "RTG-IMPORT-006" ||
		diagnostic.Filename != "/defs/child.rtg" {
		t.Fatalf("cycle diagnostic = %#v", diagnostic)
	}
}

type importTestLoader struct {
	files map[string][]byte
}

func (loader importTestLoader) LoadImport(
	importingFilename string, importPath string,
) ImportSource {
	path := importTestJoin(importingFilename, importPath)
	source, ok := loader.files[path]
	return ImportSource{Source: source, Filename: path, Ok: ok}
}

type fixedImportLoader struct {
	source   []byte
	filename string
}

func (loader fixedImportLoader) LoadImport(_ string, _ string) ImportSource {
	return ImportSource{Source: loader.source, Filename: loader.filename, Ok: true}
}

type cycleImportLoader struct {
	root []byte
}

func (loader cycleImportLoader) LoadImport(
	importingFilename string, _ string,
) ImportSource {
	if importingFilename == "/defs/root.rtg" {
		return ImportSource{
			Source:   []byte("@import \"root.rtg\"\n"),
			Filename: "/defs/child.rtg",
			Ok:       true,
		}
	}
	return ImportSource{Source: loader.root, Filename: "/defs/root.rtg", Ok: true}
}

func TestParseWithoutLoaderRejectsImport(t *testing.T) {
	document := Parse([]byte(`@import "child.rtg"`), "root.rtg")
	if document.Ok || len(document.Diagnostics) != 1 ||
		document.Diagnostics[0].Code != "RTG-IMPORT-004" {
		t.Fatalf("Parse import without loader = %#v", document.Diagnostics)
	}
	document = ParseImports([]byte(`@import "child.rtg"`), "root.rtg", nil)
	if document.Ok || len(document.Diagnostics) != 1 ||
		document.Diagnostics[0].Code != "RTG-IMPORT-004" {
		t.Fatalf("ParseImports without loader = %#v", document.Diagnostics)
	}
}

func TestParseImportsRequiresPortableRelativePath(t *testing.T) {
	for _, path := range []string{"", "/absolute.rtg", `C:\defs\machine.rtg`, `dir\machine.rtg`} {
		document := ParseImports([]byte(`@import "`+path+`"`), "root.rtg",
			fixedImportLoader{source: []byte(""), filename: "unused.rtg"})
		if document.Ok || len(document.Diagnostics) != 1 ||
			document.Diagnostics[0].Code != "RTG-IMPORT-003" {
			t.Fatalf("import path %q diagnostics = %#v", path, document.Diagnostics)
		}
	}
}

func importTestJoin(importingFilename string, importPath string) string {
	slash := len(importingFilename) - 1
	for slash >= 0 && importingFilename[slash] != '/' {
		slash--
	}
	return importingFilename[:slash+1] + importPath
}
