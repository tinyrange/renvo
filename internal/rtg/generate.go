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
	return generateFixedBackend(resolved, targetName, "backend")
}

func generateFixedBackend(resolved ResolveResult, targetName string, packageName string) GenerateResult {
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
	source := generateHeaderPackage(manifest, target.Descriptor.Name, packageName)
	source = appendDescriptorSource(source, target.Descriptor)
	source = appendEmbeddedGo(source, resolved.Document)
	return GenerateResult{Source: source, Descriptor: target.Descriptor, Manifest: manifest, Ok: true}
}

// GenerateBuiltinBackend emits one authored Go block without symbol rewriting.
// Built-in definitions use it to regenerate the existing compiler source-file
// boundaries in place. Those checked-in files already own globally unique
// compiler symbols; external fixed/universal outputs continue to use unit
// mangling so independently distributed definitions cannot collide.
func GenerateBuiltinBackend(resolved ResolveResult, targetName string, packageName string, goBlock int) GenerateResult {
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
	source := generateHeaderPackage(manifest, target.Descriptor.Name, packageName)
	blockAt := 0
	found := false
	for i := 0; i < len(resolved.Document.Declarations); i++ {
		declaration := resolved.Document.Declarations[i]
		if declaration.Kind != DeclGo {
			continue
		}
		if blockAt == goBlock {
			source = append(source, '\n')
			source = append(source, trimBlockNewlines(declaration.GoSource)...)
			source = append(source, '\n')
			found = true
			break
		}
		blockAt++
	}
	if !found {
		return GenerateResult{Diagnostics: []Diagnostic{{
			Filename: resolved.Document.Filename,
			Span:     sourceSpan(resolved.Document.Source, 0, 0),
			Code:     "RTG-GENERATE-003",
			Message:  "definition has no go backend block at requested index",
		}}}
	}
	return GenerateResult{Source: source, Descriptor: target.Descriptor, Manifest: manifest, Ok: true}
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
	for i := 0; i < len(definitions); i++ {
		source = appendEmbeddedGo(source, definitions[i].Document)
	}
	return GenerateResult{Source: source, Manifest: manifest, Ok: true}
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
		source = appendRewrittenGo(source, declaration.GoSource, names, prefix)
		source = append(source, '\n')
	}
	return source
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
		if token.Kind == TokenIdent && stringIndex(names, text) >= 0 {
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
