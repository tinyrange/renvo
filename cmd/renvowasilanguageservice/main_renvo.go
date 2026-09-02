//go:build renvo_wasi_language_service

package main

import (
	"renvo.dev/internal/c11"
	"renvo.dev/internal/check"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/languageservice"
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

type analysisOptions struct {
	mode      string
	target    string
	file      string
	packageAt string
	offset    int
	tags      []string
	language  string
}

func analysisInt(text string) (int, bool) {
	if text == "" {
		return 0, false
	}
	value := 0
	for i := 0; i < len(text); i++ {
		if text[i] < '0' || text[i] > '9' {
			return 0, false
		}
		value = value*10 + int(text[i]-'0')
	}
	return value, true
}

func analysisParse(args []string) (analysisOptions, bool) {
	options := analysisOptions{mode: "analyze", target: "wasi/wasm32", file: "main.go", packageAt: "."}
	for i := 1; i < len(args); i++ {
		if args[i] == "analyze" || args[i] == "complete" || args[i] == "signature" || args[i] == "definition" || args[i] == "references" || args[i] == "hover" || args[i] == "imports" {
			options.mode = args[i]
			continue
		}
		if (args[i] == "-target" || args[i] == "-file" || args[i] == "-offset" || args[i] == "-tags" || args[i] == "-language") && i+1 < len(args) {
			name := args[i]
			i++
			if name == "-target" {
				options.target = args[i]
			} else if name == "-file" {
				options.file = args[i]
			} else if name == "-tags" {
				options.tags = append(options.tags, args[i])
			} else if name == "-language" {
				options.language = args[i]
			} else {
				value, ok := analysisInt(args[i])
				if !ok {
					return options, false
				}
				options.offset = value
			}
			continue
		}
		if len(args[i]) == 0 || args[i][0] == '-' {
			return options, false
		}
		options.packageAt = args[i]
	}
	return options, true
}

func analysisEscape(value string) string {
	var out []byte
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case '\\':
			out = append(out, '\\', '\\')
		case '\t':
			out = append(out, '\\', 't')
		case '\n':
			out = append(out, '\\', 'n')
		case '\r':
			out = append(out, '\\', 'r')
		default:
			out = append(out, value[i])
		}
	}
	return string(out)
}

func analysisDecimal(value int) string {
	if value == 0 {
		return "0"
	}
	if value < 0 {
		return "-" + analysisDecimal(-value)
	}
	var reverse [24]byte
	count := 0
	for value > 0 {
		reverse[count] = byte('0' + value%10)
		count++
		value /= 10
	}
	out := make([]byte, count)
	for i := 0; i < count; i++ {
		out[i] = reverse[count-i-1]
	}
	return string(out)
}

func analysisLine(fields ...string) {
	var out []byte
	for i := 0; i < len(fields); i++ {
		if i > 0 {
			out = append(out, '\t')
		}
		out = append(out, analysisEscape(fields[i])...)
	}
	out = append(out, '\n')
	print(string(out))
}

func analysisDiagnostic(d driver.Diagnostic) {
	if !d.Valid() {
		return
	}
	analysisLine("D", d.Path, analysisDecimal(d.Start), analysisDecimal(d.End),
		analysisDecimal(d.Line), analysisDecimal(d.Column), d.Code, d.Message)
}

func analysisSourceFailure(source driver.SourceResult) {
	analysisDiagnostic(driver.SourceDiagnostic(source))
}

func analysisCompletion(item check.CompletionItem) {
	analysisLine("C", item.Name, item.Detail, analysisDecimal(item.Kind), item.Signature, item.Documentation)
}

func analysisHover(hover check.HoverInfo) {
	if hover.Ok {
		analysisLine("H", hover.Signature, hover.Documentation, analysisDecimal(hover.Start), analysisDecimal(hover.End))
	}
}

func analysisKeywordCompletions(files []load.SourceFile, path string, offset int) {
	source := analysisFindSource(files, path)
	items := check.CompleteKeywords(source, offset)
	for i := 0; i < len(items); i++ {
		analysisCompletion(items[i])
	}
}

func analysisImports(source []byte, offset int) {
	imports := languageservice.ParseImports(source)
	for i := 0; i < len(imports); i++ {
		analysisLine("I", imports[i].Name, imports[i].Path)
	}
	context := languageservice.ImportPathAt(source, offset)
	if context.Ok {
		closed := "0"
		if context.Closed {
			closed = "1"
		}
		analysisLine("P", context.Prefix, analysisDecimal(context.ReplaceStart), string([]byte{context.Quote}), closed)
	}
	selector := languageservice.SelectorAt(source, offset)
	if selector.Ok {
		analysisLine("Q", selector.Base, selector.Prefix, analysisDecimal(selector.ReplaceStart))
	}
}

func analysisSignature(help check.SignatureHelp) {
	if !help.Ok {
		return
	}
	fields := []string{"S", analysisDecimal(help.ActiveParameter), help.Label}
	for i := 0; i < len(help.Parameters); i++ {
		label := help.Parameters[i].Name
		if help.Parameters[i].Type != "" {
			label += " " + help.Parameters[i].Type
		}
		fields = append(fields, label)
	}
	analysisLine(fields...)
}

func analysisLocation(location check.SourceLocation) {
	analysisLine("L", location.Path, analysisDecimal(location.Start), analysisDecimal(location.End))
}

func analysisCLocation(location c11.LanguageLocation) {
	analysisLine("L", location.Path, analysisDecimal(location.Start), analysisDecimal(location.End))
}

func analysisCCompletion(item c11.LanguageCompletion) {
	analysisLine("C", item.Name, item.Detail, analysisDecimal(item.Kind), item.Signature, item.Documentation)
}

func analysisCHover(hover c11.LanguageHover) {
	if hover.Ok {
		analysisLine("H", hover.Signature, hover.Documentation, analysisDecimal(hover.Start), analysisDecimal(hover.End))
	}
}

func analysisCSignature(help c11.LanguageSignature) {
	if !help.Ok {
		return
	}
	fields := []string{"S", analysisDecimal(help.ActiveParameter), help.Label}
	for i := 0; i < len(help.Parameters); i++ {
		label := help.Parameters[i].Type
		if help.Parameters[i].Name != "" {
			if label != "" {
				label += " "
			}
			label += help.Parameters[i].Name
		}
		fields = append(fields, label)
	}
	analysisLine(fields...)
}

type analysisCIncludeReader struct {
	fs    driver.RenvoFS
	paths []string
}

func (r analysisCIncludeReader) ReadInclude(from string, name string, angled bool) ([]byte, string, bool) {
	if !angled {
		path := load.JoinPath(load.DirPath(from), name)
		if src, ok := r.fs.ReadFile(path); ok {
			return src, path, true
		}
	}
	for i := 0; i < len(r.paths); i++ {
		path := load.JoinPath(r.paths[i], name)
		if src, ok := r.fs.ReadFile(path); ok {
			return src, path, true
		}
	}
	return nil, name, false
}

func (r analysisCIncludeReader) ReadIncludeNext(from string, name string, angled bool) ([]byte, string, bool) {
	fromDir := load.DirPath(from)
	start := 0
	for i := 0; i < len(r.paths); i++ {
		if load.CleanPath(r.paths[i]) == load.CleanPath(fromDir) {
			start = i + 1
			break
		}
	}
	for i := start; i < len(r.paths); i++ {
		path := load.JoinPath(r.paths[i], name)
		if src, ok := r.fs.ReadFile(path); ok {
			return src, path, true
		}
	}
	return nil, name, false
}

func analysisCFiles(workDir string, stdRoot string, target string, queryPath string, sources []load.SourceFile) ([]c11.LanguageFile, []c11.LanguageDiagnostic) {
	libcRoot := load.JoinPath(load.DirPath(stdRoot), "libc")
	libcInclude := load.JoinPath(libcRoot, "include")
	reader := analysisCIncludeReader{fs: driver.RenvoFS{}, paths: []string{workDir, libcInclude}}
	var files []c11.LanguageFile
	var diagnostics []c11.LanguageDiagnostic
	var implementations []string
	if entries, ok := reader.fs.ReadDir(workDir); ok {
		for i := 0; i < len(entries); i++ {
			if !entries[i].IsDir && analysisEndsWith(entries[i].Name, ".h") {
				path := load.JoinPath(workDir, entries[i].Name)
				if src, read := reader.fs.ReadFile(path); read {
					files = analysisCAppendFile(files, path, src, false)
				}
			}
		}
	}
	queryPath = load.CleanPath(queryPath)
	if analysisEndsWith(queryPath, ".h") {
		if src, ok := reader.fs.ReadFile(queryPath); ok {
			files = analysisCAppendFile(files, queryPath, src, false)
		}
	}
	for i := 0; i < len(sources); i++ {
		if !analysisCPath(sources[i].Path) {
			continue
		}
		files = analysisCAppendFile(files, sources[i].Path, sources[i].Src, false)
		processed := c11.Preprocess(c11.PreprocessConfig{Path: sources[i].Path, Source: sources[i].Src,
			Reader: reader, EmitIncludes: true, EmitQuotedIncludes: true, TrackOrigins: true})
		if !processed.Ok {
			path := processed.ErrorPath
			if path == "" {
				path = sources[i].Path
			}
			shared := driver.CPreprocessorDiagnostic(processed.Error, path, processed.Line, processed.Detail)
			diagnostics = append(diagnostics, c11.LanguageDiagnostic{Path: path, Line: processed.Line, Column: 1,
				Code: shared.Code, Message: shared.Message})
			continue
		}
		for j := 0; j < len(processed.Dependencies); j++ {
			if src, ok := reader.fs.ReadFile(processed.Dependencies[j]); ok {
				files = analysisCAppendFile(files, processed.Dependencies[j], src, true)
			}
			dependency := load.CleanPath(processed.Dependencies[j])
			if load.DirPath(dependency) == load.CleanPath(libcInclude) && analysisEndsWith(dependency, ".h") {
				name := load.BasePath(dependency)
				implementation := load.JoinPath(load.JoinPath(libcRoot, "src"), name[:len(name)-2]+".c")
				if _, ok := reader.fs.ReadFile(implementation); ok && !analysisContains(implementations, implementation) {
					implementations = append(implementations, implementation)
				}
			}
		}
		checked := c11.CheckObjectForDataModel(processed.Source, analysisCDataModel(target))
		if !checked.Ok {
			shared := driver.CTranslatorDiagnostic(checked.Error, sources[i].Path, checked.ErrorAt, 1, 1)
			diagnostic := c11.LanguageDiagnostic{Path: sources[i].Path, Start: checked.ErrorAt,
				End: checked.ErrorAt + 1, Line: 1, Column: 1, Code: shared.Code, Message: shared.Message}
			if origin, found := processed.OriginAt(checked.ErrorAt); found {
				diagnostic.Path, diagnostic.Start, diagnostic.End = origin.Path, origin.Start, origin.End
				if original, read := reader.fs.ReadFile(origin.Path); read {
					diagnostic.Line, diagnostic.Column = analysisSourceLineColumn(original, origin.Start)
				}
			}
			diagnostics = append(diagnostics, diagnostic)
		}
	}
	// Index the implementation paired with each included standard header. The
	// compiler already bundles these files; exposing their original source lets
	// navigation prefer a function body over the public header declaration.
	for i := 0; i < len(implementations); i++ {
		if src, ok := reader.fs.ReadFile(implementations[i]); ok {
			files = analysisCAppendFile(files, implementations[i], src, true)
		}
	}
	return files, diagnostics
}

func analysisContains(values []string, value string) bool {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return true
		}
	}
	return false
}

func analysisCDataModel(target string) int {
	if target == "windows/amd64" || target == "windows/arm64" {
		return c11.DataModelLLP64
	}
	if analysisEndsWith(target, "/386") || analysisEndsWith(target, "/arm") || analysisEndsWith(target, "/wasm32") ||
		analysisEndsWith(target, "/vm32") || analysisEndsWith(target, "/riscv32") || analysisEndsWith(target, "/xtensa_lx7") {
		return c11.DataModelILP32
	}
	return c11.DataModelLP64
}

func analysisEndsWith(value string, suffix string) bool {
	return len(value) >= len(suffix) && value[len(value)-len(suffix):] == suffix
}

func analysisSourceLineColumn(source []byte, offset int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(source) {
		offset = len(source)
	}
	line, column := 1, 1
	for i := 0; i < offset; i++ {
		if source[i] == '\n' {
			line, column = line+1, 1
		} else {
			column++
		}
	}
	return line, column
}

func analysisCAppendFile(files []c11.LanguageFile, path string, source []byte, suppressDiagnostics bool) []c11.LanguageFile {
	path = load.CleanPath(path)
	for i := 0; i < len(files); i++ {
		if load.CleanPath(files[i].Path) == path {
			return files
		}
	}
	return append(files, c11.LanguageFile{Path: path, Source: source, SuppressDiagnostics: suppressDiagnostics})
}

func analysisCPath(path string) bool {
	return len(path) > 2 && path[len(path)-2:] == ".c" || len(path) > 2 && path[len(path)-2:] == ".h"
}

func analysisGoPath(path string) bool {
	return len(path) > 3 && path[len(path)-3:] == ".go"
}

func analysisIdentifierAt(files []load.SourceFile, path string, offset int) string {
	name, _, _ := analysisIdentifierSpan(files, path, offset)
	return name
}

func analysisIdentifierSpan(files []load.SourceFile, path string, offset int) (string, int, int) {
	src := analysisFindSource(files, path)
	if offset < 0 || offset > len(src) {
		return "", offset, offset
	}
	start, end := offset, offset
	for start > 0 && analysisIdentifierByte(src[start-1]) {
		start--
	}
	for end < len(src) && analysisIdentifierByte(src[end]) {
		end++
	}
	return string(src[start:end]), start, end
}

func analysisIdentifierByte(value byte) bool {
	return value == '_' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9'
}

func analysisGoFunctions(files []load.SourceFile, name string) []check.SourceLocation {
	var locations []check.SourceLocation
	for i := 0; i < len(files); i++ {
		if !analysisGoPath(files[i].Path) {
			continue
		}
		parsed := syntax.ParseFile(files[i].Src)
		if !parsed.Ok {
			continue
		}
		for j := 0; j < len(parsed.Funcs); j++ {
			tok := parsed.Tokens[parsed.Funcs[j].NameTok]
			if string(syntax.TokenText(parsed.Src, tok)) == name {
				locations = append(locations, check.SourceLocation{Path: files[i].Path, Start: syntax.TokenStart(tok), End: syntax.TokenEnd(tok)})
			}
		}
	}
	return locations
}

func analysisGoIdentifiers(files []load.SourceFile, name string) []check.SourceLocation {
	var locations []check.SourceLocation
	for i := 0; i < len(files); i++ {
		if !analysisGoPath(files[i].Path) {
			continue
		}
		parsed := syntax.ParseFile(files[i].Src)
		if !parsed.Ok {
			continue
		}
		for j := 0; j < len(parsed.Tokens); j++ {
			tok := parsed.Tokens[j]
			if tok.KindLine&255 == syntax.TokenIdent && string(syntax.TokenText(parsed.Src, tok)) == name {
				locations = append(locations, check.SourceLocation{Path: files[i].Path, Start: syntax.TokenStart(tok), End: syntax.TokenEnd(tok)})
			}
		}
	}
	return locations
}

func analysisGoOnlySources(files []load.SourceFile) []load.SourceFile {
	result := make([]load.SourceFile, 0, len(files))
	for i := 0; i < len(files); i++ {
		if !analysisCPath(files[i].Path) {
			result = append(result, files[i])
		}
	}
	return result
}

func analysisCallIdentifier(files []load.SourceFile, path string, offset int) string {
	src := analysisFindSource(files, path)
	parsed := syntax.ParseFile(src)
	paren, bracket, brace := 0, 0, 0
	for i := len(parsed.Tokens) - 1; i >= 0; i-- {
		tok := parsed.Tokens[i]
		if syntax.TokenStart(tok) >= offset || tok.KindLine&255 == syntax.TokenEOF {
			continue
		}
		text := string(syntax.TokenText(src, tok))
		if text == ")" {
			paren++
		} else if text == "]" {
			bracket++
		} else if text == "}" {
			brace++
		} else if text == "(" {
			if paren > 0 {
				paren--
			} else if bracket == 0 && brace == 0 && i > 0 && parsed.Tokens[i-1].KindLine&255 == syntax.TokenIdent {
				return string(syntax.TokenText(src, parsed.Tokens[i-1]))
			}
		} else if text == "[" && bracket > 0 {
			bracket--
		} else if text == "{" && brace > 0 {
			brace--
		}
	}
	return ""
}

func analysisCProject(options analysisOptions) bool {
	return options.language == "c" || analysisCPath(options.file)
}

func analysisFindSource(files []load.SourceFile, path string) []byte {
	path = load.CleanPath(path)
	for i := 0; i < len(files); i++ {
		if load.CleanPath(files[i].Path) == path {
			return files[i].Src
		}
	}
	return nil
}

func appMain(args []string, env []string) int {
	workDir := "."
	stdRoot := "./std"
	for i := 0; i < len(env); i++ {
		if len(env[i]) > 4 && env[i][:4] == "PWD=" {
			workDir = env[i][4:]
		} else if len(env[i]) > 14 && env[i][:14] == "RENVO_STDROOT=" {
			stdRoot = env[i][14:]
		}
	}
	options, ok := analysisParse(args)
	if !ok {
		analysisLine("E", "invalid analysis request")
		return 2
	}
	if options.mode == "imports" {
		source, found := (driver.RenvoFS{}).ReadFile(load.JoinPath(workDir, options.file))
		if !found {
			analysisLine("E", "source file is unavailable")
			return 0
		}
		analysisImports(source, options.offset)
		return 0
	}
	sources := driver.CollectSourcesForTargetTags(workDir, stdRoot, options.packageAt,
		options.target, options.tags, driver.RenvoFS{})
	if !sources.Ok {
		if options.mode == "complete" {
			analysisKeywordCompletions(sources.Files, options.file, options.offset)
		}
		analysisSourceFailure(sources)
		return 0
	}
	var cAnalysis c11.LanguageAnalysis
	var cPreprocessDiagnostics []c11.LanguageDiagnostic
	if analysisCProject(options) {
		cFiles, diagnostics := analysisCFiles(workDir, stdRoot, options.target, options.file, sources.Files)
		cPreprocessDiagnostics = diagnostics
		cAnalysis = c11.AnalyzeLanguage(cFiles)
		if options.mode == "analyze" {
			for i := 0; i < len(cPreprocessDiagnostics); i++ {
				diagnostic := cPreprocessDiagnostics[i]
				analysisLine("D", diagnostic.Path, analysisDecimal(diagnostic.Start), analysisDecimal(diagnostic.End),
					analysisDecimal(diagnostic.Line), analysisDecimal(diagnostic.Column), diagnostic.Code, diagnostic.Message)
			}
			for i := 0; i < len(cAnalysis.Diagnostics()); i++ {
				diagnostic := cAnalysis.Diagnostics()[i]
				analysisLine("D", diagnostic.Path, analysisDecimal(diagnostic.Start), analysisDecimal(diagnostic.End),
					analysisDecimal(diagnostic.Line), analysisDecimal(diagnostic.Column), diagnostic.Code, diagnostic.Message)
			}
			return 0
		}
		if analysisCPath(options.file) {
			path := load.CleanPath(options.file)
			if options.mode == "complete" {
				items := cAnalysis.Complete(path, options.offset)
				for i := 0; i < len(items); i++ {
					analysisCCompletion(items[i])
				}
			} else if options.mode == "signature" {
				analysisCSignature(cAnalysis.Signature(path, options.offset))
			} else if options.mode == "hover" {
				analysisCHover(cAnalysis.Hover(path, options.offset))
			} else if options.mode == "definition" || options.mode == "references" {
				if options.mode == "definition" {
					include := cAnalysis.IncludeAt(path, options.offset)
					if include.Ok {
						reader := analysisCIncludeReader{fs: driver.RenvoFS{}, paths: []string{workDir,
							load.JoinPath(load.DirPath(stdRoot), "libc/include")}}
						if _, target, ok := reader.ReadInclude(path, include.Name, include.Angled); ok {
							analysisCLocation(c11.LanguageLocation{Path: target})
						}
						return 0
					}
				}
				navigation := cAnalysis.Navigate(path, options.offset)
				if navigation.Ok {
					goDefinitions := analysisGoFunctions(sources.Files, navigation.Name)
					if options.mode == "definition" {
						if navigation.DefinitionIsDeclaration && len(goDefinitions) > 0 {
							analysisLocation(goDefinitions[0])
						} else {
							analysisCLocation(navigation.Definition)
						}
					} else {
						for i := 0; i < len(navigation.References); i++ {
							analysisCLocation(navigation.References[i])
						}
						goReferences := analysisGoIdentifiers(sources.Files, navigation.Name)
						for i := 0; i < len(goReferences); i++ {
							analysisLocation(goReferences[i])
						}
					}
				}
			}
			return 0
		}
	}
	goSources := sources.Files
	if analysisCProject(options) {
		goSources = analysisGoOnlySources(sources.Files)
	}
	result := languageservice.AnalyzeWorkspace(workDir, stdRoot, options.packageAt, goSources)
	if options.mode == "analyze" {
		analysisDiagnostic(result.Diagnostic)
		return 0
	}
	if !result.Workspace.Ok {
		if options.mode == "complete" {
			analysisKeywordCompletions(sources.Files, options.file, options.offset)
		}
		return 0
	}
	program := result.Program
	if !program.Ok {
		program = check.CheckGraphBestEffort(result.Workspace.Graph)
	}
	if options.mode == "signature" {
		help := check.SignatureHelpProgram(result.Workspace.Graph, program,
			load.CleanPath(options.file), options.offset)
		if help.Ok {
			analysisSignature(help)
		} else if analysisCProject(options) {
			name := analysisCallIdentifier(sources.Files, options.file, options.offset)
			analysisCSignature(cAnalysis.SignatureName(name))
		}
		return 0
	}
	if options.mode == "definition" || options.mode == "references" {
		navigation := check.NavigateProgram(result.Workspace.Graph, program,
			load.CleanPath(options.file), options.offset)
		name := analysisIdentifierAt(sources.Files, options.file, options.offset)
		cNavigation := cAnalysis.NavigateName(name)
		if options.mode == "definition" {
			if navigation.Ok {
				analysisLocation(navigation.Definition)
			} else if cNavigation.Ok {
				analysisCLocation(cNavigation.Definition)
			}
		} else {
			if navigation.Ok {
				for i := 0; i < len(navigation.References); i++ {
					analysisLocation(navigation.References[i])
				}
			}
			if cNavigation.Ok {
				for i := 0; i < len(cNavigation.References); i++ {
					analysisCLocation(cNavigation.References[i])
				}
			}
		}
		return 0
	}
	if options.mode == "hover" {
		hover := check.HoverProgram(result.Workspace.Graph, program,
			load.CleanPath(options.file), options.offset)
		if hover.Ok {
			analysisHover(hover)
		} else if analysisCProject(options) {
			name, start, end := analysisIdentifierSpan(sources.Files, options.file, options.offset)
			analysisCHover(cAnalysis.HoverName(name, start, end))
		}
		return 0
	}
	source := analysisFindSource(sources.Files, options.file)
	if languageservice.ImportPathAt(source, options.offset).Ok {
		return 0
	}
	completions := check.CompleteProgram(result.Workspace.Graph, program,
		load.CleanPath(options.file), options.offset)
	for i := 0; i < len(completions); i++ {
		analysisCompletion(completions[i])
	}
	if analysisCProject(options) {
		prefix, _, _ := analysisIdentifierSpan(sources.Files, options.file, options.offset)
		items := cAnalysis.CompleteGlobals(prefix)
		for i := 0; i < len(items); i++ {
			if items[i].Kind != c11.LanguageKeyword {
				analysisCCompletion(items[i])
			}
		}
	}
	return 0
}
