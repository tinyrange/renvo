//go:build renvo_wasi_language_service

package main

import (
	"renvo.dev/internal/check"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/languageservice"
	"renvo.dev/internal/load"
)

type analysisOptions struct {
	mode      string
	target    string
	file      string
	packageAt string
	offset    int
	tags      []string
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
		if args[i] == "analyze" || args[i] == "complete" || args[i] == "signature" || args[i] == "definition" || args[i] == "references" {
			options.mode = args[i]
			continue
		}
		if (args[i] == "-target" || args[i] == "-file" || args[i] == "-offset" || args[i] == "-tags") && i+1 < len(args) {
			name := args[i]
			i++
			if name == "-target" {
				options.target = args[i]
			} else if name == "-file" {
				options.file = args[i]
			} else if name == "-tags" {
				options.tags = append(options.tags, args[i])
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
	message := "workspace source loading failed"
	if source.Error == driver.SourceErrMissingModule {
		message = "go.mod was not found"
	} else if source.Error == driver.SourceErrImport || source.Error == driver.SourceErrDependencyMissing || source.Error == driver.SourceErrStandardPackage {
		message = "import could not be resolved"
	} else if source.Error == driver.SourceErrParse {
		message = "source syntax is invalid"
	}
	analysisLine("D", source.ErrorPath, analysisDecimal(source.ErrorOffset),
		analysisDecimal(source.ErrorOffset+1), "1", "1", "RENVO-LOAD-009", message)
}

func analysisCompletion(item check.CompletionItem) {
	analysisLine("C", item.Name, item.Detail, analysisDecimal(item.Kind), item.Signature)
}

func analysisKeywordCompletions(files []load.SourceFile, path string, offset int) {
	source := analysisFindSource(files, path)
	items := check.CompleteKeywords(source, offset)
	for i := 0; i < len(items); i++ {
		analysisCompletion(items[i])
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
	sources := driver.CollectSourcesForTargetTags(workDir, stdRoot, options.packageAt,
		options.target, options.tags, driver.RenvoFS{})
	if !sources.Ok {
		if options.mode == "complete" {
			analysisKeywordCompletions(sources.Files, options.file, options.offset)
		}
		analysisSourceFailure(sources)
		return 0
	}
	result := languageservice.AnalyzeWorkspace(workDir, stdRoot, options.packageAt, sources.Files)
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
	if options.mode == "signature" {
		var help check.SignatureHelp
		if result.Program.Ok {
			help = check.SignatureHelpProgram(result.Workspace.Graph, result.Program,
				load.CleanPath(options.file), options.offset)
		} else {
			help = check.SignatureHelpGraph(result.Workspace.Graph,
				load.CleanPath(options.file), options.offset)
		}
		analysisSignature(help)
		return 0
	}
	if options.mode == "definition" || options.mode == "references" {
		if !result.Program.Ok {
			return 0
		}
		navigation := check.NavigateProgram(result.Workspace.Graph, result.Program,
			load.CleanPath(options.file), options.offset)
		if !navigation.Ok {
			return 0
		}
		if options.mode == "definition" {
			analysisLocation(navigation.Definition)
		} else {
			for i := 0; i < len(navigation.References); i++ {
				analysisLocation(navigation.References[i])
			}
		}
		return 0
	}
	source := analysisFindSource(sources.Files, options.file)
	if languageservice.ImportPathAt(source, options.offset).Ok {
		return 0
	}
	var completions []check.CompletionItem
	if result.Program.Ok {
		completions = check.CompleteProgram(result.Workspace.Graph, result.Program,
			load.CleanPath(options.file), options.offset)
	} else {
		completions = check.CompleteGraph(result.Workspace.Graph,
			load.CleanPath(options.file), options.offset)
	}
	for i := 0; i < len(completions); i++ {
		analysisCompletion(completions[i])
	}
	return 0
}
