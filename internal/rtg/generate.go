package rtg

import "renvo.dev/internal/syntax"

const GeneratorVersion = 1

type GenerateResult struct {
	Source      []byte
	Descriptor  TargetDescriptor
	Manifest    []string
	Diagnostics []Diagnostic
	Ok          bool
}

func GenerateFixedBackend(resolved ResolveResult, targetName string) GenerateResult {
	if !resolved.Ok {
		diagnostics := make([]Diagnostic, len(resolved.Diagnostics))
		copy(diagnostics, resolved.Diagnostics)
		return GenerateResult{Diagnostics: diagnostics}
	}
	target, ok := lookupResolvedTarget(resolved, targetName)
	if !ok {
		return GenerateResult{Diagnostics: []Diagnostic{{
			Filename: resolved.Document.Filename,
			Span:     sourceSpan(resolved.Document.Source, 0, 0),
			Code:     "RTG-GENERATE-001",
			Message:  "definition does not export target " + targetName,
		}}}
	}
	manifest := []string{resolved.Document.Unit + " " + HashText(resolved.Document.Hash)}
	source := generateHeader(manifest, target.Descriptor.Name)
	source = appendDescriptorSource(source, target.Descriptor)
	source = appendBackendAPI(source)
	source = appendArchitectureFacts(source, resolved.Document, target.Arch, true)
	source = appendMangledEmbeddedGo(source, resolved.Document, false)
	source = appendArchitectureBindings(source, resolved.Document, target.Arch, true, false)
	return GenerateResult{Source: source, Descriptor: target.Descriptor, Manifest: manifest, Ok: true}
}

// GenerateArchitectureBackend emits the architecture-owned algorithms and
// declarative instruction bindings without a target descriptor. Built-in
// compilers use this form for checked-in architecture source shared by several
// ABI/runtime/format compositions.
func GenerateArchitectureBackend(resolved ResolveResult, archName string, packageName string) GenerateResult {
	return generateArchitectureBackend(resolved, archName, packageName, true)
}

func GenerateStatefulArchitectureBackend(resolved ResolveResult, archName string, packageName string) GenerateResult {
	return generateArchitectureBackend(resolved, archName, packageName, false)
}

func generateArchitectureBackend(resolved ResolveResult, archName string, packageName string, nativeEmitter bool) GenerateResult {
	if !resolved.Ok {
		diagnostics := make([]Diagnostic, len(resolved.Diagnostics))
		copy(diagnostics, resolved.Diagnostics)
		return GenerateResult{Diagnostics: diagnostics}
	}
	arch, ok := resolved.Document.Declaration(DeclArch, archName)
	if !ok {
		return GenerateResult{Diagnostics: []Diagnostic{{
			Filename: resolved.Document.Filename,
			Span:     sourceSpan(resolved.Document.Source, 0, 0),
			Code:     "RTG-GENERATE-004",
			Message:  "definition does not declare architecture " + archName,
		}}}
	}
	manifest := []string{resolved.Document.Unit + " " + HashText(resolved.Document.Hash)}
	source := generateHeaderPackage(manifest, "arch/"+archName, packageName)
	source = appendArchitectureFacts(source, resolved.Document, arch, true)
	source = appendMangledEmbeddedGo(source, resolved.Document, nativeEmitter)
	source = appendArchitectureBindings(source, resolved.Document, arch, true, nativeEmitter)
	return GenerateResult{Source: source, Manifest: manifest, Ok: true}
}

func GenerateArchitectureKernel(packageName string) GenerateResult {
	source := generateHeaderPackage(nil, "architecture-kernel", packageName)
	source = appendArchitectureBackendAPI(source)
	return GenerateResult{Source: source, Ok: true}
}

// GeneratePreparedBackend emits a closed definition as one package-main source
// file for the host-bootstrap preparation pipeline. Unlike distributable Go
// output it preserves authored names: the prepared backend is isolated in its
// own linked image, so names cannot collide with another definition.
func GeneratePreparedBackend(resolved ResolveResult, targetName string) GenerateResult {
	if !resolved.Ok {
		diagnostics := make([]Diagnostic, len(resolved.Diagnostics))
		copy(diagnostics, resolved.Diagnostics)
		return GenerateResult{Diagnostics: diagnostics}
	}
	target, ok := lookupResolvedTarget(resolved, targetName)
	if !ok {
		return GenerateResult{Diagnostics: []Diagnostic{{
			Filename: resolved.Document.Filename,
			Span:     sourceSpan(resolved.Document.Source, 0, 0),
			Code:     "RTG-GENERATE-001",
			Message:  "definition does not export target " + targetName,
		}}}
	}
	manifest := []string{resolved.Document.Unit + " " + HashText(resolved.Document.Hash)}
	source := generateHeaderPackage(manifest, target.Descriptor.Name, "main")
	source = appendDescriptorSource(source, target.Descriptor)
	source = appendBackendAPI(source)
	source = appendArchitectureFacts(source, resolved.Document, target.Arch, false)
	for i := 0; i < len(resolved.Document.Declarations); i++ {
		declaration := resolved.Document.Declarations[i]
		if declaration.Kind != DeclGo {
			continue
		}
		source = append(source, '\n')
		source = append(source, dedentGoSource(declaration.GoSource)...)
		source = append(source, '\n')
	}
	source = appendArchitectureBindings(source, resolved.Document, target.Arch, false, false)
	return GenerateResult{Source: source, Descriptor: target.Descriptor, Manifest: manifest, Ok: true}
}

func GenerateUniversalBackend(definitions []ResolveResult) GenerateResult {
	var diagnostics []Diagnostic
	var targets []ResolvedTarget
	var manifest []string
	for i := 0; i < len(definitions); i++ {
		if !definitions[i].Ok {
			diagnostics = append(diagnostics, definitions[i].Diagnostics...)
			continue
		}
		manifest = append(manifest, definitions[i].Document.Unit+" "+HashText(definitions[i].Document.Hash))
		for j := 0; j < len(definitions[i].Targets); j++ {
			for k := 0; k < len(targets); k++ {
				if targetNameCollides(definitions[i].Targets[j].Descriptor, targets[k].Descriptor) {
					diagnostics = append(diagnostics, resolveDiagnostic(
						definitions[i].Document,
						definitions[i].Targets[j].Declaration,
						"RTG-GENERATE-002",
						"duplicate canonical target or alias "+definitions[i].Targets[j].Descriptor.Name,
					))
				}
			}
			targets = append(targets, definitions[i].Targets[j])
		}
	}
	if len(diagnostics) != 0 {
		return GenerateResult{Diagnostics: diagnostics}
	}
	sortStrings(manifest)
	sortResolvedTargets(targets)
	source := generateHeader(manifest, "universal")
	source = append(source, "type RTGTargetDescriptor struct { Name string; OS string; ISA string; WordBits int; PointerBits int; Endian string; ABI string; Runtime string; Executable string; Object string; Definition string; Version int }\n"...)
	source = append(source, "func RTGLookupTarget(name string) (RTGTargetDescriptor, bool) {\n"...)
	source = append(source, "switch name {\n"...)
	for i := 0; i < len(targets); i++ {
		source = append(source, "case "...)
		source = appendQuoted(source, targets[i].Descriptor.Name)
		for j := 0; j < len(targets[i].Descriptor.Aliases); j++ {
			source = append(source, ", "...)
			source = appendQuoted(source, targets[i].Descriptor.Aliases[j])
		}
		source = append(source, ":\nreturn "...)
		source = appendDescriptorLiteral(source, targets[i].Descriptor)
		source = append(source, ", true\n"...)
	}
	source = append(source, "}\nreturn RTGTargetDescriptor{}, false\n}\n"...)
	source = appendBackendAPI(source)
	for i := 0; i < len(definitions); i++ {
		for j := 0; j < len(definitions[i].Document.Declarations); j++ {
			declaration := definitions[i].Document.Declarations[j]
			if declaration.Kind == DeclArch {
				source = appendArchitectureFacts(source, definitions[i].Document, declaration, true)
			}
		}
		source = appendMangledEmbeddedGo(source, definitions[i].Document, false)
		for j := 0; j < len(definitions[i].Document.Declarations); j++ {
			declaration := definitions[i].Document.Declarations[j]
			if declaration.Kind == DeclArch {
				source = appendArchitectureBindings(source, definitions[i].Document, declaration, true, false)
			}
		}
	}
	return GenerateResult{Source: source, Manifest: manifest, Ok: true}
}

// appendArchitectureBackendAPI specializes the emitter onto the assembler
// already owned by a checked-in built-in backend. Embedded algorithms retain
// their authored *RTGEmitter signatures, while Go sees the receiver as the
// existing *renvoAsm and introduces no temporary emitter state in hot paths.
func appendArchitectureBackendAPI(source []byte) []byte {
	source = appendBackendAPI(source)
	return append(source, `
func renvoRTGByte(out *renvoAsm, value byte) {
	renvoAsmEmit8(out, int(value))
}

func renvoRTGUint32(out *renvoAsm, value int) {
	renvoAsmEmit32(out, value)
}

func renvoRTGUint64(out *renvoAsm, value uint64) {
	renvoAsmEmit32(out, int(value))
	renvoAsmEmit32(out, int(value >> 32))
}

func renvoRTGPatchUint32(out *renvoAsm, at int, value int) {
	renvoPut32At(out.code, at, value)
}

func renvoRTGNewLabel(out *renvoAsm) int {
	return renvoAsmNewLabel(out)
}

func renvoRTGMark(out *renvoAsm, label int) {
	if label >= 0 {
		renvoAsmMarkLabel(out, label)
	}
}

func renvoRTGReloc(out *renvoAsm, label int) {
	if label >= 0 && len(out.code) >= 4 {
		renvoAsmAddReloc(out, len(out.code)-4, label)
	}
}
`...)
}

// appendBackendAPI emits the small target-neutral surface used by generated
// encoding algorithms. Keeping it in generated source makes a checked-in
// backend an ordinary, self-contained Go source file without increasing the
// size of legacy fixed-target compilers which do not use RTG definitions.
func appendBackendAPI(source []byte) []byte {
	return append(source, `
type RTGRegister struct {
	Code int
	Valid bool
}

type RTGLabel struct {
	Code int
	Valid bool
}

type RTGAddress struct {
	Target       RTGLabel
	Addend       int
	Base         RTGRegister
	Index        RTGRegister
	Displacement int
	Scale        int
}

type RTGCondition struct {
	Code       int
	SetOpcode  byte
	JumpOpcode byte
}
type RTGShiftDirection int

const (
	RTGShiftLeft RTGShiftDirection = 1
	RTGShiftRight RTGShiftDirection = 2
)

var RTGNoRegister = RTGRegister{}

type RTGEmitter struct {
	asm    *renvoAsm
	relocs []int
}

func renvoRTGEmitter(asm *renvoAsm) RTGEmitter {
	return RTGEmitter{asm: asm}
}

func renvoRTGLabel(code int) RTGLabel {
	return RTGLabel{Code: code, Valid: code >= 0}
}

func (out *RTGEmitter) Byte(value byte) {
	out.asm.code = append(out.asm.code, value)
}

func RTGByte(out *RTGEmitter, value byte) {
	out.Byte(value)
}

func (out *RTGEmitter) Uint32(value uint32) {
	out.asm.code = append(out.asm.code, byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24))
}

func RTGUint32(out *RTGEmitter, value int) {
	out.Uint32(uint32(value))
}

func (out *RTGEmitter) Uint64(value uint64) {
	out.Uint32(uint32(value))
	out.Uint32(uint32(value >> 32))
}

func RTGUint64(out *RTGEmitter, value uint64) {
	out.Uint64(value)
}

func (out *RTGEmitter) PatchUint32(at int, value int) {
	renvoPut32At(out.asm.code, at, value)
}

func RTGPatchUint32(out *RTGEmitter, at int, value int) {
	out.PatchUint32(at, value)
}

func (out *RTGEmitter) NewLabel() RTGLabel {
	return RTGLabel{Code: renvoAsmNewLabel(out.asm), Valid: true}
}

func (out *RTGEmitter) Mark(label RTGLabel) {
	if label.Valid {
		renvoAsmMarkLabel(out.asm, label.Code)
	}
}

func (out *RTGEmitter) Rel32(label RTGLabel) {
	out.Rel32Addend(label, 0)
}

func (out *RTGEmitter) Rel32Addend(label RTGLabel, addend int) {
	at := len(out.asm.code)
	out.Uint32(uint32(0))
	if label.Valid {
		out.relocs = append(out.relocs, at, label.Code, addend)
	}
}

func (out *RTGEmitter) Reloc(label RTGLabel) {
	if label.Valid && len(out.asm.code) >= 4 {
		renvoAsmAddReloc(out.asm, len(out.asm.code)-4, label.Code)
	}
}

func RTGReloc(out *RTGEmitter, label RTGLabel) {
	out.Reloc(label)
}

func (out *RTGEmitter) Patch() {
	for i := 0; i+2 < len(out.relocs); i += 3 {
		at := out.relocs[i]
		label := out.relocs[i+1]
		addend := out.relocs[i+2]
		target := renvoAsmLabelPosition(out.asm, label)
		if at < 0 || at+4 > len(out.asm.code) || target < 0 {
			continue
		}
		renvoPut32At(out.asm.code, at, target+addend-(at+4))
	}
}

func RTGSignedFits(value int64, bits int) bool {
	if bits <= 0 {
		return false
	}
	if bits >= 63 {
		return true
	}
	limit := int64(1) << (bits - 1)
	return value >= -limit && value < limit
}

func RTGUnsignedFits(value uint64, bits int) bool {
	if bits <= 0 {
		return false
	}
	if bits >= 63 {
		return true
	}
	return value < uint64(1) << bits
}

func RTGLog2(value int) int {
	result := 0
	for value > 1 {
		value >>= 1
		result++
	}
	return result
}
`...)
}

func generateHeader(manifest []string, target string) []byte {
	return generateHeaderPackage(manifest, target, "backend")
}

func generateHeaderPackage(manifest []string, target string, packageName string) []byte {
	var source []byte
	source = append(source, "// Code generated by Renvo RTG; DO NOT EDIT.\n// generator: "...)
	source = appendDecimalFrame(source, GeneratorVersion)
	source = append(source, "\n// target: "...)
	source = append(source, target...)
	source = append(source, '\n')
	for i := 0; i < len(manifest); i++ {
		source = append(source, "// unit: "...)
		source = append(source, manifest[i]...)
		source = append(source, '\n')
	}
	source = append(source, "package "...)
	source = append(source, packageName...)
	source = append(source, "\n\n"...)
	return source
}

func trimBlockNewlines(source []byte) []byte {
	start := 0
	for start < len(source) && (source[start] == '\n' || source[start] == '\r') {
		start++
	}
	end := len(source)
	for end > start && (source[end-1] == '\n' || source[end-1] == '\r') {
		end--
	}
	return source[start:end]
}

func appendDescriptorSource(source []byte, descriptor TargetDescriptor) []byte {
	source = append(source, "type RTGTargetDescriptor struct { Name string; OS string; ISA string; WordBits int; PointerBits int; Endian string; ABI string; Runtime string; Executable string; Object string; Definition string; Version int }\n"...)
	source = append(source, "var RTGTarget = "...)
	source = appendDescriptorLiteral(source, descriptor)
	source = append(source, '\n')
	return source
}

func appendDescriptorLiteral(source []byte, descriptor TargetDescriptor) []byte {
	source = append(source, "RTGTargetDescriptor{"...)
	values := []string{
		descriptor.Name, descriptor.OS, descriptor.ISA, descriptor.Endian,
		descriptor.ABI, descriptor.Runtime, descriptor.Executable, descriptor.Object,
		HashText(descriptor.Definition),
	}
	names := []string{"Name:", "OS:", "ISA:", "Endian:", "ABI:", "Runtime:", "Executable:", "Object:", "Definition:"}
	for i := 0; i < len(values); i++ {
		source = append(source, names[i]...)
		source = appendQuoted(source, values[i])
		source = append(source, ',')
	}
	source = append(source, "WordBits:"...)
	source = appendDecimalFrame(source, descriptor.WordBits)
	source = append(source, ",PointerBits:"...)
	source = appendDecimalFrame(source, descriptor.PointerBits)
	source = append(source, ",Version:"...)
	source = appendDecimalFrame(source, descriptor.Version)
	source = append(source, '}')
	return source
}

func appendEmbeddedGo(source []byte, document Document) []byte {
	names := embeddedGoNames(document)
	prefix := "rtg" + exportedName(document.Unit)
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		if declaration.Kind != DeclGo {
			continue
		}
		source = append(source, '\n')
		source = appendRewrittenGo(source, dedentGoSource(declaration.GoSource), names, prefix)
		source = append(source, '\n')
	}
	return source
}

func appendMangledEmbeddedGo(source []byte, document Document, nativeEmitter bool) []byte {
	names := embeddedGoNames(document)
	prefix := "rtg" + exportedName(document.Unit)
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		if declaration.Kind != DeclGo {
			continue
		}
		source = append(source, '\n')
		source = appendRewrittenGoMode(source, dedentGoSource(declaration.GoSource), names, prefix, nativeEmitter, &document)
		source = append(source, '\n')
	}
	return source
}

func dedentGoSource(source []byte) []byte {
	source = trimBlockNewlines(source)
	indent := -1
	lineStart := 0
	for lineStart < len(source) {
		lineEnd := lineStart
		for lineEnd < len(source) && source[lineEnd] != '\n' {
			lineEnd++
		}
		at := lineStart
		for at < lineEnd && (source[at] == ' ' || source[at] == '\t') {
			at++
		}
		if at < lineEnd && (indent < 0 || at-lineStart < indent) {
			indent = at - lineStart
		}
		lineStart = lineEnd + 1
	}
	if indent <= 0 {
		return source
	}
	out := make([]byte, 0, len(source))
	lineStart = 0
	for lineStart < len(source) {
		lineEnd := lineStart
		for lineEnd < len(source) && source[lineEnd] != '\n' {
			lineEnd++
		}
		remove := indent
		for remove > 0 && lineStart < lineEnd &&
			(source[lineStart] == ' ' || source[lineStart] == '\t') {
			lineStart++
			remove--
		}
		out = append(out, source[lineStart:lineEnd]...)
		if lineEnd < len(source) {
			out = append(out, '\n')
		}
		lineStart = lineEnd + 1
	}
	return out
}

func embeddedGoNames(document Document) []string {
	var names []string
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		if declaration.Kind != DeclGo {
			continue
		}
		wrapped := append([]byte("package backend\n"), declaration.GoSource...)
		file := syntax.ParseFile(wrapped)
		if !file.Ok {
			continue
		}
		for j := 0; j < len(file.Decls); j++ {
			names = append(names, string(syntax.TokenText(wrapped, file.Tokens[file.Decls[j].NameTok])))
		}
		for j := 0; j < len(file.Funcs); j++ {
			names = append(names, string(syntax.TokenText(wrapped, file.Tokens[file.Funcs[j].NameTok])))
		}
	}
	sortStrings(names)
	return names
}

func appendRewrittenGo(out []byte, source []byte, names []string, prefix string) []byte {
	return appendRewrittenGoMode(out, source, names, prefix, false, nil)
}

func appendRewrittenGoMode(out []byte, source []byte, names []string, prefix string, nativeEmitter bool, document *Document) []byte {
	tokens, diagnostics := scan(source, "")
	if len(diagnostics) != 0 {
		return out
	}
	last := 0
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if token.Kind == TokenEOF {
			break
		}
		out = append(out, source[last:token.Start]...)
		text := tokenText(source, token)
		if nativeEmitter && token.Kind == TokenIdent && text == "RTGEmitter" {
			out = append(out, "renvoAsm"...)
		} else if nativeEmitter && token.Kind == TokenIdent && text == "RTGLabel" {
			out = append(out, "int"...)
		} else if nativeEmitter && token.Kind == TokenIdent && nativeEmitterFunction(text) != "" {
			out = append(out, nativeEmitterFunction(text)...)
		} else if document != nil && token.Kind == TokenIdent {
			replacement, found := generatedArchitectureOutput(*document, text)
			if found {
				out = append(out, replacement...)
			} else if stringIndex(names, text) >= 0 {
				out = append(out, prefix...)
				out = append(out, exportedName(text)...)
			} else {
				out = append(out, source[token.Start:token.End]...)
			}
		} else if token.Kind == TokenIdent && stringIndex(names, text) >= 0 {
			out = append(out, prefix...)
			out = append(out, exportedName(text)...)
		} else {
			out = append(out, source[token.Start:token.End]...)
		}
		last = token.End
	}
	out = append(out, source[last:]...)
	return out
}

func nativeEmitterFunction(name string) string {
	if name == "RTGByte" {
		return "renvoRTGByte"
	}
	if name == "RTGUint32" {
		return "renvoRTGUint32"
	}
	if name == "RTGUint64" {
		return "renvoRTGUint64"
	}
	if name == "RTGPatchUint32" {
		return "renvoRTGPatchUint32"
	}
	if name == "RTGReloc" {
		return "renvoRTGReloc"
	}
	if name == "RTGNewLabel" {
		return "renvoRTGNewLabel"
	}
	if name == "RTGMark" {
		return "renvoRTGMark"
	}
	return ""
}

func appendQuoted(out []byte, value string) []byte {
	out = append(out, '"')
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' || value[i] == '"' {
			out = append(out, '\\')
		}
		out = append(out, value[i])
	}
	return append(out, '"')
}

func exportedName(name string) string {
	var out []byte
	upper := true
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if ch == '_' || ch == '-' || ch == '/' || ch == '.' {
			upper = true
			continue
		}
		if ch >= 'a' && ch <= 'z' && upper {
			ch -= 'a' - 'A'
		}
		if isIdentPart(ch) {
			out = append(out, ch)
			upper = false
		} else {
			upper = true
		}
	}
	if len(out) == 0 {
		return "Unit"
	}
	return string(out)
}

func lookupResolvedTarget(resolved ResolveResult, name string) (ResolvedTarget, bool) {
	for i := 0; i < len(resolved.Targets); i++ {
		if resolved.Targets[i].Descriptor.Name == name || stringIndex(resolved.Targets[i].Descriptor.Aliases, name) >= 0 {
			return resolved.Targets[i], true
		}
	}
	return ResolvedTarget{}, false
}

func sortResolvedTargets(targets []ResolvedTarget) {
	for i := 1; i < len(targets); i++ {
		value := targets[i]
		at := i
		for at > 0 && stringLess(value.Descriptor.Name, targets[at-1].Descriptor.Name) {
			targets[at] = targets[at-1]
			at--
		}
		targets[at] = value
	}
}

func stringIndex(values []string, value string) int {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return i
		}
	}
	return -1
}
