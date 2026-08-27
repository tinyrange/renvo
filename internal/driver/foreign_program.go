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

type foreignTarget struct {
	Binding unit.TargetBinding
	Tags    []string
	InPlace bool
	Ok      bool
}

type foreignDirectiveArguments struct {
	Start [3]int
	End   [3]int
	Count int
}

func prepareForeignPrograms(options *Options, workDir string, stdRoot string, moduleCache string, sources *SourceResult, fs SourceFS, result *foreignPreparation) {
	directives, ok := scanForeignDirectives(sources.Files, sources.Root.Dir, result)
	if !ok {
		return
	}
	if len(directives) == 0 {
		result.Ok = true
		return
	}
	if options.Mode != ModeExecutable || options.CCompiler || options.Script {
		setForeignDiagnostic(&result.Diagnostic, directives[0], foreignErrMode, "renvo:compile is only valid in Go executable builds", nil)
		return
	}
	programs := make([]unit.ForeignProgram, 0, len(directives))
	for i := 0; i < len(directives); i++ {
		directive := directives[i]
		target := new(foreignTarget)
		resolveForeignTarget(options, workDir, directive.Target, fs, target)
		if !target.Ok {
			setForeignDiagnostic(&result.Diagnostic, directive, foreignErrTarget, "renvo:compile target is not available from the selected backend: "+directive.Target, nil)
			return
		}
		childTags := make([]string, len(options.Tags))
		copy(childTags, options.Tags)
		for tag := 0; tag < len(options.BackendBuildTags); tag++ {
			childTags = removeForeignTag(childTags, options.BackendBuildTags[tag])
		}
		for tag := 0; tag < len(target.Tags); tag++ {
			if findString(childTags, target.Tags[tag]) < 0 {
				childTags = append(childTags, target.Tags[tag])
			}
		}
		var childSources SourceResult
		if fs != nil {
			if len(options.Files) > 0 {
				childSources = CollectSourceFilesForTargetTagsWithModuleCache(workDir, stdRoot, options.Files, directive.Target, childTags, moduleCache, fs)
			} else {
				childSources = CollectSourcesForTargetTagsWithModuleCache(workDir, stdRoot, options.Package, directive.Target, childTags, moduleCache, fs)
			}
		} else {
			filtered, path, valid := filterSourcesForTargetTags(sources.Files, directive.Target, childTags)
			childSources = SourceResult{Files: filtered, Module: sources.Module, Root: sources.Root, Ok: valid, ErrorPath: path}
			if !valid {
				childSources.Error = SourceErrBuildConstraint
			}
		}
		if !childSources.Ok {
			failed := BuildResult{Error: BuildErrSource, ErrorPath: childSources.ErrorPath, Sources: childSources,
				ErrorAt: -1, ErrorPackage: -1, ErrorFile: -1, ErrorToken: -1}
			cause := diagnosticForBuild(failed)
			setForeignDiagnostic(&result.Diagnostic, directive, foreignErrBuild, "foreign frontend for "+directive.Target+" failed: "+cause.Message, &cause)
			return
		}
		rootArg := options.Package
		if len(options.Files) > 0 {
			rootArg = childSources.Root.Dir
		}
		built := pipeline.BuildUnit(workDir, stdRoot, rootArg, childSources.Files)
		if !built.Ok {
			failed := BuildResult{Error: BuildErrPipeline, Pipeline: built, Sources: childSources,
				ErrorAt: built.ErrorOffset, ErrorPackage: built.ErrorPackage, ErrorFile: built.ErrorFile, ErrorToken: built.ErrorToken}
			cause := diagnosticForBuild(failed)
			setForeignDiagnostic(&result.Diagnostic, directive, foreignErrBuild, "foreign frontend for "+directive.Target+" failed: "+cause.Message, &cause)
			return
		}
		entry := linkedForeignFunction(&built.Link.Program, directive.Entry)
		if entry < 0 {
			setForeignDiagnostic(&result.Diagnostic, directive, foreignErrEntrypoint, "foreign entry function was not found in the root package: "+directive.Entry, nil)
			return
		}
		childUnit, bound := unit.BindEntrypoint(built.Link.Data, entry)
		if !bound {
			setForeignDiagnostic(&result.Diagnostic, directive, foreignErrUnit, "could not bind the foreign entrypoint into its unit", nil)
			return
		}
		childUnit, bound = unit.BindTarget(childUnit, target.Binding)
		if !bound {
			setForeignDiagnostic(&result.Diagnostic, directive, foreignErrTarget, "foreign target has no backend binding: "+directive.Target, nil)
			return
		}
		programs = append(programs, unit.ForeignProgram{Name: directive.Name, Kind: directive.Kind,
			Target: target.Binding.Target, InPlace: target.InPlace, Unit: childUnit})
	}
	result.Programs = programs
	result.Ok = true
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

func linkedForeignFunction(program *unit.Program, name string) int {
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

func linkedForeignGlobal(program *unit.Program, name string) int {
	start, end := 0, len(program.Decls)
	for i := 0; i < len(program.Packages); i++ {
		pkg := program.Packages[i]
		if pkg.ImportPath == program.ImportPath {
			start, end = pkg.DeclStart, pkg.DeclEnd
			break
		}
	}
	for i := start; i < end; i++ {
		decl := program.Decls[i]
		if decl.Kind == unit.TokenVar && decl.NameStart >= 0 && decl.NameEnd <= len(program.Text) && string(program.Text[decl.NameStart:decl.NameEnd]) == name {
			return decl.NameStart
		}
	}
	return -1
}

func scanForeignDirectives(files []load.SourceFile, rootDir string, result *foreignPreparation) ([]foreignDirective, bool) {
	var directives []foreignDirective
	for i := 0; i < len(files); i++ {
		file := files[i]
		name := load.BasePath(file.Path)
		if load.DirPath(file.Path) != rootDir || len(name) < 3 || name[len(name)-3:] != ".go" ||
			!foreignSourceContainsDirective(file.Src) {
			continue
		}
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
				parsed := syntax.ParseFile(file.Src)
				if !parsed.Ok {
					break
				}
				fields := foreignDirectiveFields(file.Src, argsStart, lineEnd)
				base := foreignDirective{Path: file.Path, Offset: contentStart, Line: line, Column: contentStart - lineStart + 1}
				if fields.Count != 3 || string(file.Src[fields.Start[0]:fields.End[0]]) != "-t" {
					setForeignDiagnostic(&result.Diagnostic, base, foreignErrDirective, "expected //renvo:compile -t <target> <entry>", nil)
					return nil, false
				}
				base.Target = string(file.Src[fields.Start[1]:fields.End[1]])
				base.Entry = string(file.Src[fields.Start[2]:fields.End[2]])
				decl, kind, ok := foreignDirectiveDeclaration(parsed, lineEnd)
				if !ok {
					setForeignDiagnostic(&result.Diagnostic, base, foreignErrDeclaration, "renvo:compile must immediately precede one uninitialized package variable", nil)
					return nil, false
				}
				base.Name = string(syntax.TokenText(file.Src, parsed.Tokens[decl.NameTok]))
				base.Kind = kind
				if kind == 0 {
					setForeignDiagnostic(&result.Diagnostic, base, foreignErrType, "renvo:compile variable type must be []byte or uintptr", nil)
					return nil, false
				}
				for previous := 0; previous < len(directives); previous++ {
					if directives[previous].Name == base.Name {
						setForeignDiagnostic(&result.Diagnostic, base, foreignErrDeclaration, "duplicate renvo:compile variable: "+base.Name, nil)
						return nil, false
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
	return directives, true
}

func foreignSourceContainsDirective(src []byte) bool {
	for i := 0; i+13 <= len(src); i++ {
		if src[i] == 'r' && src[i+1] == 'e' && src[i+2] == 'n' &&
			src[i+3] == 'v' && src[i+4] == 'o' && src[i+5] == ':' &&
			src[i+6] == 'c' && src[i+7] == 'o' && src[i+8] == 'm' &&
			src[i+9] == 'p' && src[i+10] == 'i' && src[i+11] == 'l' && src[i+12] == 'e' {
			return true
		}
	}
	return false
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
		if fields.Count < 3 {
			fields.Start[fields.Count], fields.End[fields.Count] = fieldStart, start
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
	newline := src[after]
	if newline != '\r' && newline != '\n' {
		return false
	}
	after++
	if after < declaration && src[after] != newline && (src[after] == '\r' || src[after] == '\n') {
		after++
	}
	for after < declaration && (src[after] == ' ' || src[after] == '\t') {
		after++
	}
	return after == declaration
}

func setForeignDiagnostic(d *Diagnostic, directive foreignDirective, detail int, message string, cause *Diagnostic) {
	d.Phase = "foreign"
	d.Code = "RENVO-FOREIGN-00" + diagnosticIntText(detail)
	d.Message = message
	d.Path = directive.Path
	d.Start = directive.Offset
	d.End = directive.Offset
	if cause != nil && cause.Path != "" {
		d.Path, d.Start, d.End, d.Line, d.Column = cause.Path, cause.Start, cause.End, cause.Line, cause.Column
		return
	}
	d.Line, d.Column = directive.Line, directive.Column
}
