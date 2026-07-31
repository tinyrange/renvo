package rtg

import (
	"crypto/sha256"
	goparser "go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
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

func TestAppendQuotedEmitsGoSourceForArbitraryBytes(t *testing.T) {
	value := string([]byte{0, 'A', '"', '\\', '\n', 0x7f, 0x80, 0xff})
	got := string(appendQuoted(nil, value))
	want := `"\x00A\"\\\x0a\x7f\x80\xff"`
	if got != want {
		t.Fatalf("appendQuoted = %q, want %q", got, want)
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

func TestEmbeddedGoIsTypeCheckedAgainstBackendAPI(t *testing.T) {
	tests := []string{
		`go backend { func emit() { missingAlgorithm() } }`,
		`go backend { func helper(value int) {} func emit() { helper() } }`,
	}
	prefix := "definition 1\nunit demo\nimplements direct_emitter_v1\narch a {}\n"
	for i := 0; i < len(tests); i++ {
		document := Parse([]byte(prefix+tests[i]), "bad.rtg")
		if document.Ok || len(document.Diagnostics) == 0 ||
			document.Diagnostics[0].Code != "RTG-GO-009" {
			t.Errorf("Parse case %d = ok %v diagnostics %#v", i, document.Ok, document.Diagnostics)
		}
	}
}

func TestTruncatedBlocksFail(t *testing.T) {
	document := Parse([]byte("definition 1\nunit demo\nimplements direct_emitter_v1\narch a { width = 64"), "truncated.rtg")
	if document.Ok || len(document.Diagnostics) == 0 || document.Diagnostics[0].Code != "RTG-PARSE-014" {
		t.Fatalf("Parse = ok %v diagnostics %#v", document.Ok, document.Diagnostics)
	}
}

func TestTruncationAtEveryDeclarationTokenBoundaryFails(t *testing.T) {
	source := []byte(`definition 1
unit truncation
implements direct_emitter_v1
arch a {
	registers {
		primary = r0
	}
	word_bits = 64
}
`)
	complete := Parse(source, "complete.rtg")
	if !complete.Ok {
		t.Fatalf("complete Parse failed: %#v", complete.Diagnostics)
	}
	open := -1
	close := -1
	for i := 0; i < len(complete.Tokens); i++ {
		text := tokenText(source, complete.Tokens[i])
		if open < 0 && text == "{" {
			open = i
		}
		if text == "}" {
			close = i
		}
	}
	if open < 0 || close <= open {
		t.Fatal("test declaration token bounds were not found")
	}
	for i := open + 1; i <= close; i++ {
		offsets := []int{complete.Tokens[i].Start, complete.Tokens[i].End}
		for j := 0; j < len(offsets); j++ {
			offset := offsets[j]
			if offset >= complete.Tokens[close].End {
				continue
			}
			document := Parse(source[:offset], "truncated.rtg")
			if document.Ok {
				t.Fatalf("Parse accepted truncation at token %d offset %d", i, offset)
			}
		}
	}
}

func TestInvalidUTF8FailsPrecisely(t *testing.T) {
	source := append([]byte("system bad { target = \""), 0xff)
	source = append(source, []byte("\" }\n")...)
	document := Parse(source, "invalid-utf8.rtg")
	if document.Ok || !hasDiagnosticCode(document.Diagnostics, "RTG-SCAN-003") {
		t.Fatalf("Parse = ok %v diagnostics %#v", document.Ok, document.Diagnostics)
	}
	if document.Diagnostics[0].Span.Start.Offset != len("system bad { target = \"") {
		t.Fatalf("invalid UTF-8 span = %#v", document.Diagnostics[0].Span)
	}
}

func TestDefinitionResourceLimits(t *testing.T) {
	oversize := Parse(make([]byte, maxDefinitionBytes+1), "oversize.rtg")
	if oversize.Ok || !hasDiagnosticCode(oversize.Diagnostics, "RTG-LIMIT-001") {
		t.Fatalf("oversize Parse = ok %v diagnostics %#v", oversize.Ok, oversize.Diagnostics)
	}

	tokens := Parse([]byte(strings.Repeat("x ", maxDefinitionTokens+1)), "tokens.rtg")
	if tokens.Ok || !hasDiagnosticCode(tokens.Diagnostics, "RTG-LIMIT-002") {
		t.Fatalf("token-heavy Parse = ok %v diagnostics %#v", tokens.Ok, tokens.Diagnostics)
	}

	var declarations strings.Builder
	for i := 0; i <= maxDefinitionDecls; i++ {
		declarations.WriteString("system \"s")
		declarations.WriteString(strconv.Itoa(i))
		declarations.WriteString("\" {}\n")
	}
	tooManyDeclarations := Parse([]byte(declarations.String()), "declarations.rtg")
	if tooManyDeclarations.Ok || !hasDiagnosticCode(tooManyDeclarations.Diagnostics, "RTG-LIMIT-003") {
		t.Fatalf("declaration-heavy Parse = ok %v diagnostics %#v",
			tooManyDeclarations.Ok, tooManyDeclarations.Diagnostics)
	}

	var nesting strings.Builder
	nesting.WriteString("system nested {\n")
	for i := 0; i <= maxStatementNesting; i++ {
		nesting.WriteString("level {\n")
	}
	for i := 0; i <= maxStatementNesting; i++ {
		nesting.WriteString("}\n")
	}
	nesting.WriteString("}\n")
	tooDeep := Parse([]byte(nesting.String()), "nested.rtg")
	if tooDeep.Ok || !hasDiagnosticCode(tooDeep.Diagnostics, "RTG-LIMIT-004") {
		t.Fatalf("deeply nested Parse = ok %v diagnostics %#v", tooDeep.Ok, tooDeep.Diagnostics)
	}

	table := "system table {\nvalues = [" +
		strings.Repeat("value,", maxFieldValueTokens/2+1) + "]\n}\n"
	tooLargeTable := Parse([]byte(table), "table.rtg")
	if tooLargeTable.Ok || !hasDiagnosticCode(tooLargeTable.Diagnostics, "RTG-LIMIT-005") {
		t.Fatalf("oversized table Parse = ok %v diagnostics %#v",
			tooLargeTable.Ok, tooLargeTable.Diagnostics)
	}

	statements := "system statements {\n" +
		strings.Repeat("value\n", maxStatementsPerBody+1) + "}\n"
	tooManyStatements := Parse([]byte(statements), "statements.rtg")
	if tooManyStatements.Ok || !hasDiagnosticCode(tooManyStatements.Diagnostics, "RTG-LIMIT-006") {
		t.Fatalf("statement-heavy Parse = ok %v diagnostics %#v",
			tooManyStatements.Ok, tooManyStatements.Diagnostics)
	}
}

func TestABIRequiresExactlyOneArchitectureSource(t *testing.T) {
	resolved := Resolve(Parse([]byte(`definition 1
unit ambiguous_abi
implements direct_emitter_v1
arch a {}
abi base { arch = a }
abi ambiguous {
	arch = a
	internal = base
}
target test/ambiguous {
	os = test
	arch = a
	abi = ambiguous
}
`), "ambiguous-abi.rtg"))
	if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, "RTG-VALIDATE-027") {
		t.Fatalf("Resolve = ok %v diagnostics %#v", resolved.Ok, resolved.Diagnostics)
	}
}

func TestABIInheritanceCycleIsDiagnosed(t *testing.T) {
	resolved := Resolve(Parse([]byte(`definition 1
unit cyclic_abi
implements direct_emitter_v1
arch a {}
abi first { internal = second }
abi second { internal = first }
target test/cycle {
	os = test
	arch = a
	abi = first
}
`), "cyclic-abi.rtg"))
	if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, "RTG-VALIDATE-028") {
		t.Fatalf("Resolve = ok %v diagnostics %#v", resolved.Ok, resolved.Diagnostics)
	}
}

func TestArchitectureRelocationHookMustResolve(t *testing.T) {
	resolved := Resolve(Parse([]byte(`definition 1
unit bad_hook
implements direct_emitter_v1
arch a {
	patch_relocations = go missing
}`), "bad-hook.rtg"))
	if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, "RTG-VALIDATE-019") {
		t.Fatalf("Resolve = ok %v diagnostics %#v", resolved.Ok, resolved.Diagnostics)
	}
}

func TestDirectEmitterV1HasCompleteEffectContracts(t *testing.T) {
	ensureDirectEmitterV1()
	if len(directEmitterV1) == 0 {
		t.Fatal("direct_emitter_v1 is empty")
	}
	for _, operation := range directEmitterV1 {
		if operation.Control == "" || len(operation.Preserves) == 0 {
			t.Errorf("%s has incomplete control or preservation effects", operation.Name)
		}
		if hasPrefixText(operation.Name, "load.") || hasPrefixText(operation.Name, "store.") {
			if operation.Memory == "" {
				t.Errorf("%s has no memory width/signedness/alignment contract", operation.Name)
			}
		}
		if operation.Name == "call" || operation.Name == "jump" ||
			operation.Name == "jump_condition" {
			if operation.Relocation == "" {
				t.Errorf("%s has no label/relocation effect", operation.Name)
			}
		}
	}
}

func TestDeclarativeFixedWordInstructionForms(t *testing.T) {
	ensureDirectEmitterV1()
	source := `definition 1
unit word
implements direct_emitter_v1

arch fixed32 {
	alias = "fixed"
	endian = little
	word_bits = 32
	pointer_bits = 32
	forms {
		rr = word32(destination:reg@0+5, source:reg@16)
		branch = word32(label:label@reloc)
		half = word16(destination:reg@4, address:address_base@0)
	}
	instructions {
		move_word rr 0x8b000000
		jump_word branch 0x14000000
		load_half half 0x8000
	}
	sequences {
		emitPair(out:emitter, value:int) {
			call RTGUint32(out, value)
			call RTGUint32(out, value)
		}
	}
	exports {
		renvoEmitPair = sequence emitPair
	}
	bind {
		move = move_word
		jump = jump_word
	}
	reject = [`
	for i := 0; i < len(directEmitterV1); i++ {
		if directEmitterV1[i].Name != "move" && directEmitterV1[i].Name != "jump" {
			source += directEmitterV1[i].Name + ","
		}
	}
	source += `]
}
abi fixed_abi { arch = fixed32 }
runtime fixed_runtime { operation print { builtin = true } }
format fixed_image { address_bits = 32 }
target test/fixed {
	family = native_v1
	os = test
	arch = fixed32
	abi = fixed_abi
	runtime = fixed_runtime
	executable = fixed_image
}
`
	document := Parse([]byte(source), "word32.rtg")
	resolved := ResolveDefinitions(document)
	if !resolved.Ok {
		t.Fatalf("Resolve failed: %#v", resolved.Diagnostics)
	}
	generated := GenerateArchitectureBackend(resolved, "fixed32", "backend")
	if !generated.Ok {
		t.Fatalf("GenerateArchitectureBackend failed: %#v", generated.Diagnostics)
	}
	text := string(generated.Source)
	for _, want := range []string{
		"func rtgWordMoveWord(out *renvoAsm, destination RTGRegister, source RTGRegister)",
		"renvoRTGUint32(out, 0x8b000000|(destination.Code)|(destination.Code<<5)|(source.Code<<16))",
		"func rtgWordJumpWord(out *renvoAsm, label int)",
		"renvoRTGReloc(out, label)",
		"func rtgWordLoadHalf(out *renvoAsm, destination RTGRegister, address renvoRTGAddress)",
		"renvoAsmEmit16(out, 0x8000|(destination.Code<<4)|(address.Base.Code))",
		"func renvoEmitPair(out *renvoAsm, value int)",
		"renvoAsmEmit32(out,value)",
	} {
		if !containsText(text, want) {
			t.Errorf("generated source missing %q:\n%s", want, text)
		}
	}

	bigEndian := ResolveDefinitions(Parse(
		[]byte(replaceOnce(source, "endian = little", "endian = big")),
		"word32-big-endian.rtg"))
	if !bigEndian.Ok {
		t.Fatalf("big-endian Resolve failed: %#v", bigEndian.Diagnostics)
	}
	bigGenerated := GenerateArchitectureBackend(bigEndian, "fixed32", "backend")
	if !bigGenerated.Ok {
		t.Fatalf("big-endian generation failed: %#v", bigGenerated.Diagnostics)
	}
	bigText := string(bigGenerated.Source)
	for _, want := range []string{
		"value := 0x8000|(destination.Code<<4)|(address.Base.Code)",
		"renvoRTGByte(out, byte(value>>8))",
		"renvoRTGByte(out, byte(value>>0))",
	} {
		if !containsText(bigText, want) {
			t.Errorf("big-endian generated source missing %q:\n%s", want, bigText)
		}
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
	reject = [
		move, address,
		load.native, load.i32, load.u32, load.i16, load.u16, load.i8, load.u8,
		store.native, store.u32, store.u16, store.u8,
		add, subtract, multiply, bit_and, bit_or, bit_xor, compare, test,
		increment, decrement,
		shift_left_immediate, shift_right_unsigned_immediate, shift_right_signed_immediate,
		call, call_indirect, jump, jump_condition, set_condition, return, leave,
		host_syscall, move_immediate, variable_shift, signed_divide, copy_bytes
	]
	exports {
		renvoTinyAddTwo = go addTwo
	}
}
abi tiny_abi { arch = tiny64 }
runtime tiny_runtime { operation print { builtin = true } }
format tiny_image { address_bits = 64 }
target test/tiny64 {
	family = native_v1
	os = test
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
	for _, want := range []string{"// target: test/tiny64", "func rtgTinyAddOne", "func renvoTinyAddTwo", "rtgTinyAddOne(rtgTinyAddOne(value))"} {
		if !containsText(string(generated.Source), want) {
			t.Errorf("generated source missing %q:\n%s", want, generated.Source)
		}
	}
	second := GenerateFixedBackend(resolved, "test/tiny64")
	if string(generated.Source) != string(second.Source) {
		t.Fatal("fixed generation is not deterministic")
	}
}

func TestTargetSemanticIdentityIgnoresUnreachableCatalogDeclarations(t *testing.T) {
	base := Resolve(Parse([]byte(testMachineDefinition), "tiny.rtg"))
	if !base.Ok {
		t.Fatalf("base Resolve failed: %#v", base.Diagnostics)
	}
	extendedSource := testMachineDefinition + `
go backend {
	func unrelatedArchitectureHelper(value int) int { return value * 7 }
}
`
	extended := Resolve(Parse([]byte(extendedSource), "tiny-extended.rtg"))
	if !extended.Ok {
		t.Fatalf("extended Resolve failed: %#v", extended.Diagnostics)
	}
	if base.Document.Hash == extended.Document.Hash {
		t.Fatal("catalog identity did not include the unrelated declaration")
	}
	if base.Targets[0].Descriptor.Definition != extended.Targets[0].Descriptor.Definition {
		t.Fatalf("unrelated declaration changed target identity: %s != %s",
			HashText(base.Targets[0].Descriptor.Definition),
			HashText(extended.Targets[0].Descriptor.Definition))
	}
}

func TestTargetSemanticIdentityTracksReachableSemanticsNotFormatting(t *testing.T) {
	base := Resolve(Parse([]byte(testMachineDefinition), "tiny.rtg"))
	formattedSource := replaceOnce(testMachineDefinition,
		"func addOne(value int) int { return value + 1 }",
		"func addOne( value int ) int {\n\t\t// same operation\n\t\treturn value + 1\n\t}")
	formatted := Resolve(Parse([]byte(formattedSource), "tiny-formatted.rtg"))
	changedSource := replaceOnce(testMachineDefinition,
		"return value + 1", "return value + 2")
	changed := Resolve(Parse([]byte(changedSource), "tiny-changed.rtg"))
	if !base.Ok || !formatted.Ok || !changed.Ok {
		t.Fatalf("Resolve failed: base=%#v formatted=%#v changed=%#v",
			base.Diagnostics, formatted.Diagnostics, changed.Diagnostics)
	}
	if base.Targets[0].Descriptor.Definition != formatted.Targets[0].Descriptor.Definition {
		t.Fatal("Go formatting or comments changed target semantic identity")
	}
	if base.Targets[0].Descriptor.Definition == changed.Targets[0].Descriptor.Definition {
		t.Fatal("reachable Go behavior did not change target semantic identity")
	}

	runtimeChanged := replaceOnce(testMachineDefinition,
		"runtime tiny_runtime { operation print { builtin = true } }",
		"runtime tiny_runtime { operations = [print] }")
	runtime := Resolve(Parse([]byte(runtimeChanged), "tiny-runtime.rtg"))
	if !runtime.Ok {
		t.Fatalf("runtime Resolve failed: %#v", runtime.Diagnostics)
	}
	if base.Targets[0].Descriptor.Definition == runtime.Targets[0].Descriptor.Definition {
		t.Fatal("reachable runtime declaration did not change target semantic identity")
	}
}

func TestTargetMetricsSeparateReachableAndCatalogGo(t *testing.T) {
	source := testMachineDefinition + `
go backend {
	func unrelatedMetricsHelper(value int) int { return value * 7 }
}
`
	resolved := Resolve(Parse([]byte(source), "tiny-metrics.rtg"))
	if !resolved.Ok {
		t.Fatalf("Resolve failed: %#v", resolved.Diagnostics)
	}
	metrics := MeasureTarget(resolved.Document, resolved.Targets[0])
	if metrics.DeclarativeBytes == 0 || metrics.ReachableGoBytes == 0 ||
		metrics.ReachableGoDecls != 2 {
		t.Fatalf("metrics = %#v", metrics)
	}
	if metrics.CatalogGoBytes <= metrics.ReachableGoBytes ||
		metrics.CatalogGoDecls <= metrics.ReachableGoDecls {
		t.Fatalf("unreachable Go was not separated: %#v", metrics)
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

func TestGeneratePreparedUsesStableUnitSymbols(t *testing.T) {
	resolved := Resolve(Parse([]byte(testMachineDefinition), "tiny.rtg"))
	generated := GeneratePreparedBackend(resolved, "test/tiny64")
	if !generated.Ok {
		t.Fatalf("GeneratePreparedBackend failed: %#v", generated.Diagnostics)
	}
	text := string(generated.Source)
	for _, want := range []string{"package main", "var RTGTarget", "func rtgTinyAddOne", "rtgTinyAddOne(rtgTinyAddOne(value))"} {
		if !containsText(text, want) {
			t.Errorf("prepared source missing %q:\n%s", want, text)
		}
	}
	if containsText(text, "func addOne") {
		t.Fatalf("prepared generation retained an unmangled embedded symbol:\n%s", text)
	}
}
func TestResolveRejectsUnknownCompositionReference(t *testing.T) {
	source := replaceOnce(testMachineDefinition, "abi = tiny_abi", "abi = missing")
	resolved := Resolve(Parse([]byte(source), "bad.rtg"))
	if resolved.Ok || len(resolved.Diagnostics) == 0 || resolved.Diagnostics[0].Code != "RTG-RESOLVE-010" {
		t.Fatalf("Resolve = ok %v diagnostics %#v", resolved.Ok, resolved.Diagnostics)
	}
}

func TestResolveRequiresExplicitTargetOS(t *testing.T) {
	source := replaceOnce(testMachineDefinition, "\tos = test\n", "")
	resolved := Resolve(Parse([]byte(source), "missing-os.rtg"))
	if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, "RTG-RESOLVE-015") {
		t.Fatalf("Resolve = ok %v diagnostics %#v", resolved.Ok, resolved.Diagnostics)
	}
}

func TestResolveRejectsUnknownAndDuplicateFields(t *testing.T) {
	source := replaceOnce(testMachineDefinition,
		"pointer_bits = 64",
		"pointer_bits = 64\n\tpointer_bits = 64\n\tmystery = true")
	resolved := Resolve(Parse([]byte(source), "bad.rtg"))
	if resolved.Ok || len(resolved.Diagnostics) < 2 {
		t.Fatalf("Resolve = ok %v diagnostics %#v", resolved.Ok, resolved.Diagnostics)
	}
	if resolved.Diagnostics[0].Code != "RTG-VALIDATE-060" ||
		resolved.Diagnostics[1].Code != "RTG-VALIDATE-061" {
		t.Fatalf("diagnostics = %#v", resolved.Diagnostics)
	}
}

func TestResolveRejectsUnknownInstructionAlgorithm(t *testing.T) {
	source := replaceOnce(testMachineDefinition,
		"arch tiny64 {",
		"arch tiny64 {\n\tinstructions { add = go missingAlgorithm }")
	resolved := Resolve(Parse([]byte(source), "bad.rtg"))
	if resolved.Ok || len(resolved.Diagnostics) == 0 ||
		resolved.Diagnostics[0].Code != "RTG-VALIDATE-011" {
		t.Fatalf("Resolve = ok %v diagnostics %#v", resolved.Ok, resolved.Diagnostics)
	}
}

func TestResolveRejectsUnknownBackendContract(t *testing.T) {
	source := []byte(`definition 1
unit tiny
implements direct_emitter_v2
arch tiny64 {
	endian = little
	word_bits = 64
}
abi tiny_abi { arch = tiny64 }
runtime tiny_runtime { operations = [print] }
format tiny_image { kind = tiny address_bits = 64 byte_order = little }
target tiny/tiny64 {
	family = native_v1
	os = tiny
	arch = tiny64
	abi = tiny_abi
	runtime = tiny_runtime
	executable = tiny_image
}`)
	resolved := Resolve(Parse(source, "unknown-contract.rtg"))
	if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, "RTG-VALIDATE-070") {
		t.Fatalf("diagnostics = %#v, want RTG-VALIDATE-070", resolved.Diagnostics)
	}
}

func TestResolveRejectsIncompleteDirectEmitterBinding(t *testing.T) {
	source := []byte(`definition 1
unit tiny
implements direct_emitter_v1
go backend {
	func move(out *RTGEmitter, destination RTGRegister, source RTGRegister) {}
}
arch tiny64 {
	endian = little
	word_bits = 64
	bind { move = go move }
}
abi tiny_abi { arch = tiny64 }
runtime tiny_runtime { operations = [print] }
format tiny_image { kind = tiny address_bits = 64 byte_order = little }
target tiny/tiny64 {
	family = native_v1
	os = tiny
	arch = tiny64
	abi = tiny_abi
	runtime = tiny_runtime
	executable = tiny_image
}`)
	resolved := Resolve(Parse(source, "incomplete-bind.rtg"))
	if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, "RTG-VALIDATE-075") {
		t.Fatalf("diagnostics = %#v, want RTG-VALIDATE-075", resolved.Diagnostics)
	}
}

func TestDirectEmitterContractGeneratesTypedDirectWrappers(t *testing.T) {
	source := completeDirectEmitterDefinition()
	resolved := Resolve(Parse([]byte(source), "complete-bind.rtg"))
	if !resolved.Ok {
		t.Fatalf("Resolve failed: %#v\n%s", resolved.Diagnostics, source)
	}
	generated := GenerateFixedBackend(resolved, "test/tiny64")
	if !generated.Ok {
		t.Fatalf("Generate failed: %#v", generated.Diagnostics)
	}
	text := string(generated.Source)
	for _, want := range []string{
		"func rtgTinyDirectMove(",
		"rtgTinyOperation0(",
		"func rtgTinyDirectLoadNative(",
		"func rtgTinyDirectHostSyscall(",
		"func rtgTinyDirectCopyBytes(",
	} {
		if !containsText(text, want) {
			t.Errorf("generated source missing %q", want)
		}
	}
	fset := token.NewFileSet()
	if _, err := goparser.ParseFile(fset, "generated.go", generated.Source, 0); err != nil {
		t.Fatalf("generated direct emitter source does not parse: %v", err)
	}
}

func TestResolveRejectsDirectEmitterSignatureMismatch(t *testing.T) {
	source := completeDirectEmitterDefinition()
	source = replaceOnce(source,
		"func operation0(p0 *RTGEmitter, p1 RTGRegister, p2 RTGRegister) {}",
		"func operation0(p0 *RTGEmitter, p1 int, p2 RTGRegister) {}")
	resolved := Resolve(Parse([]byte(source), "signature-mismatch.rtg"))
	if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, "RTG-VALIDATE-076") {
		t.Fatalf("diagnostics = %#v, want RTG-VALIDATE-076", resolved.Diagnostics)
	}
}

func TestResolveRejectsABIHookContractErrors(t *testing.T) {
	hooks := []struct {
		name       string
		parameters []string
	}{
		{"frame_start", []string{"*RTGEmitter"}},
		{"frame_finish", []string{"*RTGEmitter", "int", "int"}},
		{"push_register", []string{"*RTGEmitter", "RTGRegister"}},
		{"push_immediate", []string{"*RTGEmitter", "int"}},
		{"pop_register", []string{"*RTGEmitter", "RTGRegister"}},
		{"frame_load", []string{"*RTGEmitter", "RTGRegister", "int"}},
		{"frame_store", []string{"*RTGEmitter", "int", "RTGRegister"}},
		{"frame_address", []string{"*RTGEmitter", "RTGRegister", "int"}},
		{"store_param_word", []string{"*RTGEmitter", "int", "int"}},
		{"call_word_count", []string{"*RTGEmitter", "RTGLabel", "int"}},
	}
	for _, hook := range hooks {
		t.Run(hook.name, func(t *testing.T) {
			source := completeDirectEmitterDefinition()
			parameters := ""
			for i := 0; i < len(hook.parameters); i++ {
				if i != 0 {
					parameters += ", "
				}
				typ := hook.parameters[i]
				if i == len(hook.parameters)-1 {
					typ = "string"
				}
				parameters += "p" + testDecimal(i) + " " + typ
			}
			source = replaceOnce(source, "}\narch tiny64 {",
				"\tfunc abiHook("+parameters+") {}\n}\narch tiny64 {")
			source = replaceOnce(source, "abi tiny_abi { arch = tiny64 }",
				"abi tiny_abi {\n\tarch = tiny64\n\t"+hook.name+" = go abiHook\n}")
			resolved := Resolve(Parse([]byte(source), "abi-"+hook.name+".rtg"))
			if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, "RTG-VALIDATE-026") {
				t.Fatalf("diagnostics = %#v, want RTG-VALIDATE-026", resolved.Diagnostics)
			}
		})
	}

	source := replaceOnce(completeDirectEmitterDefinition(),
		"abi tiny_abi { arch = tiny64 }",
		"abi tiny_abi {\n\tarch = tiny64\n\tframe_start = operation0\n}")
	resolved := Resolve(Parse([]byte(source), "abi-invalid-binding.rtg"))
	if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, "RTG-VALIDATE-025") {
		t.Fatalf("diagnostics = %#v, want RTG-VALIDATE-025", resolved.Diagnostics)
	}
}

func completeDirectEmitterDefinition() string {
	ensureDirectEmitterV1()
	source := "definition 1\nunit tiny\nimplements direct_emitter_v1\n"
	source += "go backend {\n"
	for i := 0; i < len(directEmitterV1); i++ {
		source += "\tfunc operation" + testDecimal(i) + "("
		for parameter := 0; parameter < len(directEmitterV1[i].Parameters); parameter++ {
			if parameter != 0 {
				source += ", "
			}
			source += "p" + testDecimal(parameter) + " " + directEmitterV1[i].Parameters[parameter]
		}
		source += ") {}\n"
	}
	source += "}\narch tiny64 {\n\tendian = little\n\tword_bits = 64\n\tbind {\n"
	for i := 0; i < len(directEmitterV1); i++ {
		source += "\t\t" + directEmitterV1[i].Name + " = go operation" + testDecimal(i) + "\n"
	}
	source += "\t}\n}\n"
	source += "abi tiny_abi { arch = tiny64 }\n"
	source += "runtime tiny_runtime { operations = [print] }\n"
	source += "format tiny_image { kind = tiny address_bits = 64 byte_order = little }\n"
	source += "target test/tiny64 {\n"
	source += "\tfamily = native_v1\n\tos = test\n\tarch = tiny64\n\tabi = tiny_abi\n\truntime = tiny_runtime\n\texecutable = tiny_image\n}\n"
	return source
}

func TestResolveRequiresKnownBackendFamily(t *testing.T) {
	missing := replaceOnce(testMachineDefinition, "\tfamily = native_v1\n", "")
	resolved := Resolve(Parse([]byte(missing), "missing-family.rtg"))
	if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, "RTG-RESOLVE-022") {
		t.Fatalf("missing family diagnostics = %#v", resolved.Diagnostics)
	}
	unknown := replaceOnce(testMachineDefinition, "family = native_v1", "family = mysterious")
	resolved = Resolve(Parse([]byte(unknown), "unknown-family.rtg"))
	if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, "RTG-RESOLVE-023") {
		t.Fatalf("unknown family diagnostics = %#v", resolved.Diagnostics)
	}
}

func TestResolveSeparatesNativeAndStructuredFamilies(t *testing.T) {
	nativeStructured := replaceOnce(testMachineDefinition, "alias = \"tiny\"", "alias = \"wasm32\"")
	resolved := Resolve(Parse([]byte(nativeStructured), "native-structured.rtg"))
	if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, "RTG-RESOLVE-024") {
		t.Fatalf("native structured diagnostics = %#v", resolved.Diagnostics)
	}
	structuredNative := replaceOnce(testMachineDefinition, "family = native_v1", "family = structured32")
	resolved = Resolve(Parse([]byte(structuredNative), "structured-native.rtg"))
	if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, "RTG-RESOLVE-025") {
		t.Fatalf("structured native diagnostics = %#v", resolved.Diagnostics)
	}
}

func testDecimal(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	at := len(digits)
	for value > 0 {
		at--
		digits[at] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[at:])
}

func hasDiagnosticCode(diagnostics []Diagnostic, code string) bool {
	for i := 0; i < len(diagnostics); i++ {
		if diagnostics[i].Code == code {
			return true
		}
	}
	return false
}

func TestResolveRejectsDuplicateRegisters(t *testing.T) {
	source := replaceOnce(testMachineDefinition,
		"arch tiny64 {",
		"arch tiny64 {\n\tregisters gpr = [r0, r1, r0]")
	resolved := Resolve(Parse([]byte(source), "bad.rtg"))
	if resolved.Ok || len(resolved.Diagnostics) == 0 ||
		resolved.Diagnostics[0].Code != "RTG-VALIDATE-010" {
		t.Fatalf("Resolve = ok %v diagnostics %#v", resolved.Ok, resolved.Diagnostics)
	}
}

func TestNativeMachineSchemaModelsAddressSpacesAndRegisterSubsets(t *testing.T) {
	source := replaceOnce(testMachineDefinition, "registers gpr = [r0, r1, r0]",
		"registers gpr = [r0, r1, r0]")
	source = replaceOnce(source, "arch tiny64 {", `arch tiny64 {
	registers gpr8 = [r0, r1, r2, r3]
	register_class immediate = [r2, r3]
	register_group pairs {
		p0 = [r0, r1]
		p1 = [r2, r3]
	}
	address_space data {
		address_bits = 16
		pointer_bits = 16
		unit_bits = 8
		pointer_words = 1
	}
	address_space code {
		address_bits = 32
		pointer_bits = 16
		unit_bits = 8
		pointer_words = 2
	}`)
	resolved := Resolve(Parse([]byte(source), "native-model.rtg"))
	if !resolved.Ok {
		t.Fatalf("Resolve failed: %#v", resolved.Diagnostics)
	}
}

func TestNativeMachineSchemaRejectsUnknownRegisterAndIncompleteAddressSpace(t *testing.T) {
	source := replaceOnce(testMachineDefinition, "arch tiny64 {", `arch tiny64 {
	registers gpr8 = [r0, r1]
	register_class immediate = [r1, missing]
	address_space data {
		address_bits = 16
		pointer_bits = 16
	}`)
	resolved := Resolve(Parse([]byte(source), "bad-native-model.rtg"))
	if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, "RTG-VALIDATE-083") ||
		!hasDiagnosticCode(resolved.Diagnostics, "RTG-VALIDATE-096") ||
		!hasDiagnosticCode(resolved.Diagnostics, "RTG-VALIDATE-097") {
		t.Fatalf("diagnostics = %#v", resolved.Diagnostics)
	}
}

func TestNativeSchemaStressFixturesGenerateDirectGo(t *testing.T) {
	fixtures := []struct {
		file   string
		target string
	}{
		{"testdata/native_8086.rtg", "fixture/8086"},
		{"testdata/native_avr.rtg", "fixture/avr"},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.target, func(t *testing.T) {
			source, err := os.ReadFile(fixture.file)
			if err != nil {
				t.Fatal(err)
			}
			resolved := Resolve(Parse(source, fixture.file))
			if !resolved.Ok {
				t.Fatalf("Resolve failed: %#v", resolved.Diagnostics)
			}
			generated := GenerateFixedBackend(resolved, fixture.target)
			if !generated.Ok {
				t.Fatalf("Generate failed: %#v", generated.Diagnostics)
			}
			if _, err := goparser.ParseFile(token.NewFileSet(), "generated.go",
				generated.Source, goparser.AllErrors); err != nil {
				t.Fatalf("generated source does not parse: %v\n%s", err, generated.Source)
			}
			for _, operation := range []string{
				"DirectMove", "DirectLoadNative", "DirectStoreNative",
				"DirectAdd", "DirectCall", "DirectJumpCondition",
			} {
				if !containsText(string(generated.Source), operation) {
					t.Errorf("generated source is missing representative operation %s", operation)
				}
			}
		})
	}
}

func TestResolveRejectsMismatchedABIArchitecture(t *testing.T) {
	source := replaceOnce(testMachineDefinition,
		"abi tiny_abi { arch = tiny64 }",
		`arch other {
	endian = little
	word_bits = 64
	reject = [
		move, address,
		load.native, load.i32, load.u32, load.i16, load.u16, load.i8, load.u8,
		store.native, store.u32, store.u16, store.u8,
		add, subtract, multiply, bit_and, bit_or, bit_xor, compare, test,
		increment, decrement,
		shift_left_immediate, shift_right_unsigned_immediate, shift_right_signed_immediate,
		call, call_indirect, jump, jump_condition, set_condition, return, leave,
		host_syscall, move_immediate, variable_shift, signed_divide, copy_bytes
	]
}
abi tiny_abi { arch = other }`)
	resolved := Resolve(Parse([]byte(source), "bad.rtg"))
	if resolved.Ok || len(resolved.Diagnostics) == 0 ||
		resolved.Diagnostics[0].Code != "RTG-VALIDATE-050" {
		t.Fatalf("Resolve = ok %v diagnostics %#v", resolved.Ok, resolved.Diagnostics)
	}
}

func TestResolveRejectsUnknownRuntimeOperationBlock(t *testing.T) {
	source := replaceOnce(completeDirectEmitterDefinition(),
		"runtime tiny_runtime { operations = [print] }",
		"runtime tiny_runtime { operation mystery { unsupported = \"not implemented\" } }")
	resolved := Resolve(Parse([]byte(source), "unknown-runtime-operation.rtg"))
	if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, "RTG-VALIDATE-030") {
		t.Fatalf("diagnostics = %#v, want RTG-VALIDATE-030", resolved.Diagnostics)
	}
}

func TestResolveRejectsDuplicateRuntimeOperationBlocks(t *testing.T) {
	source := replaceOnce(completeDirectEmitterDefinition(),
		"runtime tiny_runtime { operations = [print] }",
		"runtime tiny_runtime {\n\toperation print { builtin = true }\n\toperation print { builtin = true }\n}")
	resolved := Resolve(Parse([]byte(source), "duplicate-runtime-operation.rtg"))
	if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, "RTG-VALIDATE-033") {
		t.Fatalf("diagnostics = %#v, want RTG-VALIDATE-033", resolved.Diagnostics)
	}
}

func TestResolveRejectsMismatchedFormatByteOrder(t *testing.T) {
	source := replaceOnce(completeDirectEmitterDefinition(),
		"format tiny_image { kind = tiny address_bits = 64 byte_order = little }",
		"format tiny_image { kind = tiny address_bits = 64 byte_order = big }")
	resolved := Resolve(Parse([]byte(source), "mismatched-format-byte-order.rtg"))
	if resolved.Ok || !hasDiagnosticCode(resolved.Diagnostics, "RTG-VALIDATE-052") {
		t.Fatalf("diagnostics = %#v, want RTG-VALIDATE-052", resolved.Diagnostics)
	}
}

func TestDeclarativeInstructionGeneratesDirectWrapper(t *testing.T) {
	source := replaceOnce(testMachineDefinition,
		"arch tiny64 {",
		"arch tiny64 {\n\tinstructions { increment = go addOne }")
	resolved := Resolve(Parse([]byte(source), "tiny.rtg"))
	if !resolved.Ok {
		t.Fatalf("Resolve failed: %#v", resolved.Diagnostics)
	}
	generated := GenerateFixedBackend(resolved, "test/tiny64")
	if !generated.Ok {
		t.Fatalf("Generate failed: %#v", generated.Diagnostics)
	}
	text := string(generated.Source)
	for _, want := range []string{
		"func rtgTinyIncrement(value int) int",
		"result := rtgTinyAddOne(value)",
	} {
		if !containsText(text, want) {
			t.Errorf("generated source missing %q:\n%s", want, text)
		}
	}
}

func TestInstructionFormSpecializesConstantsIntoDirectWrapper(t *testing.T) {
	source := replaceOnce(testMachineDefinition,
		"func addOne(value int) int { return value + 1 }",
		`func addOne(value int) int { return value + 1 }
	func registerForm(opcode uint32, destination uint32, source uint32, width int) uint32 {
		return opcode | destination | source | uint32(width)
	}`)
	source = replaceOnce(source,
		"arch tiny64 {",
		`arch tiny64 {
	forms { rr = go registerForm }
	instructions { add64 rr 0x8b }`)
	resolved := Resolve(Parse([]byte(source), "tiny.rtg"))
	if !resolved.Ok {
		t.Fatalf("Resolve failed: %#v", resolved.Diagnostics)
	}
	generated := GenerateFixedBackend(resolved, "test/tiny64")
	if !generated.Ok {
		t.Fatalf("Generate failed: %#v", generated.Diagnostics)
	}
	text := string(generated.Source)
	for _, want := range []string{
		"func rtgTinyAdd64(destination uint32, source uint32) uint32",
		"result := rtgTinyRegisterForm(0x8b, destination, source, 64)",
	} {
		if !containsText(text, want) {
			t.Errorf("generated source missing %q:\n%s", want, text)
		}
	}
	if _, err := goparser.ParseFile(token.NewFileSet(), "generated.go", generated.Source, goparser.AllErrors); err != nil {
		t.Fatalf("specialized source does not parse: %v\n%s", err, generated.Source)
	}
}

func TestFixedInstructionFormUsesWideEmitterWrites(t *testing.T) {
	source := replaceOnce(testMachineDefinition,
		"arch tiny64 {",
		`arch tiny64 {
	forms { fixed = bytes }
	instructions {
		trap fixed [0x01, 0x02, 0x03, 0x04]
		stop fixed [0xff]
	}`)
	resolved := Resolve(Parse([]byte(source), "tiny.rtg"))
	if !resolved.Ok {
		t.Fatalf("Resolve failed: %#v", resolved.Diagnostics)
	}
	generated := GenerateFixedBackend(resolved, "test/tiny64")
	if !generated.Ok {
		t.Fatalf("Generate failed: %#v", generated.Diagnostics)
	}
	text := string(generated.Source)
	for _, want := range []string{
		"func rtgTinyTrap(out *RTGEmitter)",
		"RTGUint32(out, 0x04<<8|0x03<<8|0x02<<8|0x01)",
		"func rtgTinyStop(out *RTGEmitter)",
		"RTGByte(out, 0xff)",
	} {
		if !containsText(text, want) {
			t.Errorf("generated source missing %q:\n%s", want, text)
		}
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
