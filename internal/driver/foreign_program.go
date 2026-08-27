package driver

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/pipeline"
	"renvo.dev/internal/syntax"
	"renvo.dev/internal/unit"
)

const (
	foreignErrDirective = iota + 1
	foreignErrDeclaration
	foreignErrType
	foreignErrMode
	foreignErrTarget
	foreignErrBuild
	foreignErrEntrypoint
	foreignErrUnit
)

type foreignDirective struct {
	Name   string
	Target string
	Entry  string
	Kind   int
	Path   string
	Offset int
	Line   int
	Column int
}

type foreignPreparation struct {
	Programs   []unit.ForeignProgram
	Diagnostic Diagnostic
	Ok         bool
}

type foreignDirectiveArguments struct {
	Start0 int
	End0   int
	Start1 int
	End1   int
	Start2 int
	End2   int
	Count  int
}

func prepareForeignPrograms(options Options, workDir string, stdRoot string, moduleCache string, sources SourceResult, fs SourceFS) foreignPreparation {
	directives, diagnostic, ok := scanForeignDirectives(sources.Files, sources.Root.Dir)
	if !ok {
		return foreignPreparation{Diagnostic: diagnostic}
	}
	if len(directives) == 0 {
		return foreignPreparation{Ok: true}
	}
	if options.Mode != ModeExecutable || options.CCompiler || options.Script {
		return foreignPreparation{Diagnostic: foreignDiagnostic(directives[0], foreignErrMode, "renvo:compile is only valid in Go executable builds", nil)}
	}
	programs := make([]unit.ForeignProgram, 0, len(directives))
	for i := 0; i < len(directives); i++ {
		directive := directives[i]
		binding, inPlace, targetTags, supported := resolveForeignTarget(options, workDir, directive.Target, fs)
		if !supported {
			return foreignPreparation{Diagnostic: foreignDiagnostic(directive, foreignErrTarget, "renvo:compile target is not available from the selected backend: "+directive.Target, nil)}
		}
		childOptions := options
		childOptions.Tags = make([]string, len(options.Tags))
		copy(childOptions.Tags, options.Tags)
		childOptions.Target = directive.Target
		childOptions.TargetExplicit = true
		childOptions.Output = ""
		childOptions.EmitUnit = true
		childOptions.EmitImage = false
		childOptions.Strip = true
		childOptions.BinaryLimit = 0
		childOptions.System = ""
		childOptions.SystemName = ""
		childOptions.ArenaSize = 0
		for tag := 0; tag < len(options.BackendBuildTags); tag++ {
			childOptions.Tags = removeForeignTag(childOptions.Tags, options.BackendBuildTags[tag])
		}
		for tag := 0; tag < len(targetTags); tag++ {
			if findString(childOptions.Tags, targetTags[tag]) < 0 {
				childOptions.Tags = append(childOptions.Tags, targetTags[tag])
			}
		}
		var childSources SourceResult
		if fs != nil {
			if len(childOptions.Files) > 0 {
				childSources = CollectSourceFilesForTargetTagsWithModuleCache(workDir, stdRoot, childOptions.Files, childOptions.Target, childOptions.Tags, moduleCache, fs)
			} else {
				childSources = CollectSourcesForTargetTagsWithModuleCache(workDir, stdRoot, childOptions.Package, childOptions.Target, childOptions.Tags, moduleCache, fs)
			}
		} else {
			filtered, path, sourceError := filterSourcesForOptions(sources.Files, workDir, childOptions)
			childSources = SourceResult{Files: filtered, Module: sources.Module, Root: sources.Root, Ok: sourceError == SourceOK, Error: sourceError, ErrorPath: path}
		}
		if !childSources.Ok {
			failed := BuildResult{Error: BuildErrSource, ErrorPath: childSources.ErrorPath, Sources: childSources,
				ErrorAt: -1, ErrorPackage: -1, ErrorFile: -1, ErrorToken: -1}
			cause := diagnosticForBuild(failed)
			return foreignPreparation{Diagnostic: foreignDiagnostic(directive, foreignErrBuild, "foreign frontend for "+directive.Target+" failed: "+cause.Message, &cause)}
		}
		rootArg := childOptions.Package
		if len(childOptions.Files) > 0 {
			rootArg = childSources.Root.Dir
		}
		built := pipeline.BuildUnit(workDir, stdRoot, rootArg, childSources.Files)
		if !built.Ok {
			failed := BuildResult{Error: BuildErrPipeline, Pipeline: built, Sources: childSources,
				ErrorAt: built.ErrorOffset, ErrorPackage: built.ErrorPackage, ErrorFile: built.ErrorFile, ErrorToken: built.ErrorToken}
			cause := diagnosticForBuild(failed)
			return foreignPreparation{Diagnostic: foreignDiagnostic(directive, foreignErrBuild, "foreign frontend for "+directive.Target+" failed: "+cause.Message, &cause)}
		}
		entry := linkedForeignFunction(built.Link.Program, directive.Entry)
		if entry < 0 {
			return foreignPreparation{Diagnostic: foreignDiagnostic(directive, foreignErrEntrypoint, "foreign entry function was not found in the root package: "+directive.Entry, nil)}
		}
		childUnit, bound := unit.BindEntrypoint(built.Link.Data, entry)
		if !bound {
			return foreignPreparation{Diagnostic: foreignDiagnostic(directive, foreignErrUnit, "could not bind the foreign entrypoint into its unit", nil)}
		}
		childUnit, bound = unit.BindTarget(childUnit, binding)
		if !bound {
			return foreignPreparation{Diagnostic: foreignDiagnostic(directive, foreignErrTarget, "foreign target has no backend binding: "+directive.Target, nil)}
		}
		programs = append(programs, unit.ForeignProgram{Name: directive.Name, Kind: directive.Kind,
			Target: binding.Target, InPlace: inPlace, Unit: childUnit})
	}
	return foreignPreparation{Programs: programs, Ok: true}
}

func removeForeignTag(tags []string, unwanted string) []string {
	out := tags[:0]
	for i := 0; i < len(tags); i++ {
		if tags[i] != unwanted {
			out = append(out, tags[i])
		}
	}
	return out
}

func targetDescriptorCapability(capabilities []string, wanted string) bool {
	for i := 0; i < len(capabilities); i++ {
		if capabilities[i] == wanted {
			return true
		}
	}
	return false
}

func linkedForeignFunction(program unit.Program, name string) int {
	start, end := 0, len(program.Funcs)
	for i := 0; i < len(program.Packages); i++ {
		pkg := program.Packages[i]
		if pkg.ImportPath == program.ImportPath {
			start, end = pkg.FuncStart, pkg.FuncEnd
			break
		}
	}
	found := -1
	for i := start; i < end; i++ {
		fn := program.Funcs[i]
		if fn.NameStart >= 0 && fn.NameEnd <= len(program.Text) && string(program.Text[fn.NameStart:fn.NameEnd]) == name {
			if found >= 0 {
				return -1
			}
			found = i
		}
	}
	return found
}

func scanForeignDirectives(files []load.SourceFile, rootDir string) ([]foreignDirective, Diagnostic, bool) {
	var directives []foreignDirective
	for i := 0; i < len(files); i++ {
		file := files[i]
		name := load.BasePath(file.Path)
		if load.DirPath(file.Path) != rootDir || len(name) < 3 || name[len(name)-3:] != ".go" {
			continue
		}
		var parsed syntax.File
		parsedFile := false
		line := 1
		for lineStart := 0; lineStart < len(file.Src); {
			lineEnd := lineStart
			for lineEnd < len(file.Src) && file.Src[lineEnd] != '\n' && file.Src[lineEnd] != '\r' {
				lineEnd++
			}
			contentStart := lineStart
			for contentStart < lineEnd && (file.Src[contentStart] == ' ' || file.Src[contentStart] == '\t') {
				contentStart++
			}
			argsStart := foreignDirectiveArgsStart(file.Src, contentStart, lineEnd)
			if argsStart >= 0 {
				if !parsedFile {
					parsed = syntax.ParseFile(file.Src)
					parsedFile = true
				}
				if !parsed.Ok {
					break
				}
				fields := foreignDirectiveFields(file.Src, argsStart, lineEnd)
				base := foreignDirective{Path: file.Path, Offset: contentStart, Line: line, Column: contentStart - lineStart + 1}
				if fields.Count != 3 || string(file.Src[fields.Start0:fields.End0]) != "-t" {
					return nil, foreignDiagnostic(base, foreignErrDirective, "expected //renvo:compile -t <target> <entry>", nil), false
				}
				base.Target = string(file.Src[fields.Start1:fields.End1])
				base.Entry = string(file.Src[fields.Start2:fields.End2])
				decl, kind, ok := foreignDirectiveDeclaration(parsed, lineEnd)
				if !ok {
					return nil, foreignDiagnostic(base, foreignErrDeclaration, "renvo:compile must immediately precede one uninitialized package variable", nil), false
				}
				base.Name = string(syntax.TokenText(file.Src, parsed.Tokens[decl.NameTok]))
				base.Kind = kind
				if kind == 0 {
					return nil, foreignDiagnostic(base, foreignErrType, "renvo:compile variable type must be []byte or uintptr", nil), false
				}
				for previous := 0; previous < len(directives); previous++ {
					if directives[previous].Name == base.Name {
						return nil, foreignDiagnostic(base, foreignErrDeclaration, "duplicate renvo:compile variable: "+base.Name, nil), false
					}
				}
				directives = append(directives, base)
			}
			for lineEnd < len(file.Src) && (file.Src[lineEnd] == '\n' || file.Src[lineEnd] == '\r') {
				lineEnd++
			}
			lineStart = lineEnd
			line++
		}
	}
	return directives, Diagnostic{}, true
}

func foreignDirectiveArgsStart(src []byte, start int, end int) int {
	prefix := "//renvo:compile"
	if end-start >= len(prefix) && string(src[start:start+len(prefix)]) == prefix {
		return start + len(prefix)
	}
	prefix = "// renvo:compile"
	if end-start >= len(prefix) && string(src[start:start+len(prefix)]) == prefix {
		return start + len(prefix)
	}
	return -1
}

func foreignDirectiveFields(src []byte, start int, end int) foreignDirectiveArguments {
	var fields foreignDirectiveArguments
	for start < end {
		for start < end && (src[start] == ' ' || src[start] == '\t') {
			start++
		}
		if start >= end {
			break
		}
		fieldStart := start
		for start < end && src[start] != ' ' && src[start] != '\t' {
			start++
		}
		if fields.Count == 0 {
			fields.Start0, fields.End0 = fieldStart, start
		} else if fields.Count == 1 {
			fields.Start1, fields.End1 = fieldStart, start
		} else if fields.Count == 2 {
			fields.Start2, fields.End2 = fieldStart, start
		}
		fields.Count++
	}
	return fields
}

func foreignDirectiveDeclaration(file syntax.File, after int) (syntax.TopDecl, int, bool) {
	var empty syntax.TopDecl
	token := -1
	for i := 0; i < len(file.Tokens); i++ {
		if syntax.TokenStart(file.Tokens[i]) >= after {
			token = i
			break
		}
	}
	if token < 0 || file.Tokens[token].KindLine&255 != syntax.TokenVar || token+2 >= len(file.Tokens) ||
		file.Tokens[token+1].KindLine&255 != syntax.TokenIdent ||
		!foreignDirectiveImmediatelyPrecedes(file.Src, after, syntax.TokenStart(file.Tokens[token])) {
		return empty, 0, false
	}
	var decl syntax.TopDecl
	found := false
	for i := 0; i < len(file.Decls); i++ {
		if file.Decls[i].NameTok == token+1 && file.Decls[i].Kind == syntax.TokenVar {
			decl, found = file.Decls[i], true
			break
		}
	}
	if !found || decl.StartTok != token+1 {
		return empty, 0, false
	}
	typeStart := token + 2
	if decl.EndTok == typeStart+1 && string(syntax.TokenText(file.Src, file.Tokens[typeStart])) == "uintptr" {
		return decl, unit.ForeignProgramEntrypoint, true
	}
	if decl.EndTok == typeStart+3 && string(syntax.TokenText(file.Src, file.Tokens[typeStart])) == "[" &&
		string(syntax.TokenText(file.Src, file.Tokens[typeStart+1])) == "]" &&
		string(syntax.TokenText(file.Src, file.Tokens[typeStart+2])) == "byte" {
		return decl, unit.ForeignProgramBytes, true
	}
	return decl, 0, true
}

func foreignDirectiveImmediatelyPrecedes(src []byte, after int, declaration int) bool {
	if after < 0 || after >= len(src) || declaration <= after || declaration > len(src) {
		return false
	}
	if src[after] == '\r' {
		after++
		if after < declaration && src[after] == '\n' {
			after++
		}
	} else if src[after] == '\n' {
		after++
		if after < declaration && src[after] == '\r' {
			after++
		}
	} else {
		return false
	}
	for after < declaration && (src[after] == ' ' || src[after] == '\t') {
		after++
	}
	return after == declaration
}

func foreignDiagnostic(directive foreignDirective, detail int, message string, cause *Diagnostic) Diagnostic {
	d := Diagnostic{Phase: "foreign", Code: "RENVO-FOREIGN-00" + diagnosticIntText(detail), Message: message,
		Path: directive.Path, Start: directive.Offset, End: directive.Offset}
	if cause != nil && cause.Path != "" {
		d.Path, d.Start, d.End, d.Line, d.Column = cause.Path, cause.Start, cause.End, cause.Line, cause.Column
		return d
	}
	d.Line, d.Column = directive.Line, directive.Column
	return d
}
