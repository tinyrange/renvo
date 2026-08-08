package repl

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/check"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/languageservice"
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

// Completion is one semantic frontend completion visible at the REPL cursor.
type Completion struct {
	Name         string
	Insert       string
	Detail       string
	Signature    string
	Kind         int
	ReplaceStart int
}

type CompletionParameter struct {
	Name string
	Type string
}

type SignatureHelp struct {
	Ok              bool
	Label           string
	Parameters      []CompletionParameter
	ActiveParameter int
}

type completionOverlayFS struct {
	base         driver.SourceFS
	sourcePath   string
	source       []byte
	modulePath   string
	moduleSource []byte
}

func (fs completionOverlayFS) ReadDir(path string) ([]driver.DirEntry, bool) {
	entries, ok := fs.base.ReadDir(path)
	return entries, ok
}

func (fs completionOverlayFS) ReadFile(path string) ([]byte, bool) {
	path = load.CleanPath(path)
	if path == fs.sourcePath {
		return fs.source, true
	}
	if fs.modulePath != "" && path == fs.modulePath {
		return fs.moduleSource, true
	}
	data, ok := fs.base.ReadFile(path)
	return data, ok
}

func (fs completionOverlayFS) PathExists(path string) bool {
	if _, ok := fs.ReadFile(path); ok {
		return true
	}
	return fs.base.PathExists(path)
}

// Complete runs the same graph/checker completion query used by the IDE over
// a synthetic source file containing the live REPL generation.
func (s *State) Complete(input string, cursor int, env []string) []Completion {
	importContext := languageservice.ImportPathAt([]byte(input), cursor)
	if importContext.Ok {
		paths := languageservice.CompleteStandardImportPaths(
			replCompletionStdRoot(env), replCompletionTarget(), nil,
			importContext.Prefix, replCompletionFS(),
		)
		items := make([]Completion, 0, len(paths))
		for i := 0; i < len(paths); i++ {
			insert := paths[i]
			if !importContext.Closed {
				insertBytes := []byte(insert)
				insertBytes = append(insertBytes, importContext.Quote)
				insert = string(insertBytes)
			}
			items = append(items, Completion{
				Name:         paths[i],
				Insert:       insert,
				Detail:       "standard package",
				Kind:         check.CompletionPackage,
				ReplaceStart: importContext.ReplaceStart,
			})
		}
		return items
	}

	analysis, path, query, mark, ok := s.completionAnalysis(input, cursor, env)
	if !ok {
		return nil
	}
	var items []check.CompletionItem
	if analysis.Program.Ok {
		items = check.CompleteProgram(analysis.Workspace.Graph, analysis.Program, path, query)
	} else {
		items = check.CompleteGraph(analysis.Workspace.Graph, path, query)
	}
	persistMark := arena.PersistMark()
	staged := arena.PersistBytes(replStageCompletions(items))
	arena.Reset(mark)
	completions := replRestoreCompletions(staged)
	arena.PersistReset(persistMark)
	return completions
}

// Signature returns the IDE signature-help result for the call containing the
// live REPL cursor.
func (s *State) Signature(input string, cursor int, env []string) SignatureHelp {
	analysis, path, query, mark, ok := s.completionAnalysis(input, cursor, env)
	if !ok {
		return SignatureHelp{}
	}
	var help check.SignatureHelp
	if analysis.Program.Ok {
		help = check.SignatureHelpProgram(analysis.Workspace.Graph, analysis.Program, path, query)
	} else {
		help = check.SignatureHelpGraph(analysis.Workspace.Graph, path, query)
	}
	persistMark := arena.PersistMark()
	staged := arena.PersistBytes(replStageSignature(help))
	arena.Reset(mark)
	result := replRestoreSignature(staged)
	arena.PersistReset(persistMark)
	return result
}

func (s *State) completionAnalysis(input string, cursor int, env []string) (languageservice.AnalysisResult, string, int, int, bool) {
	mark := arena.Mark()
	source, query := s.completionSource(input, cursor)
	if query < 0 {
		arena.Reset(mark)
		return languageservice.AnalysisResult{}, "", 0, mark, false
	}
	workDir := replEnvValue(env, "PWD")
	if workDir == "" {
		workDir = "."
	}
	workDir = load.CleanPath(workDir)
	base := replCompletionFS()
	fs := completionOverlayFS{
		base:       base,
		sourcePath: load.JoinPath(workDir, "renvo_repl_completion.go"),
		source:     source,
	}
	if !replHasCompletionModule(workDir, base) {
		fs.modulePath = load.JoinPath(workDir, "go.mod")
		fs.moduleSource = []byte("module renvo.dev/repl\n")
	}

	sources := driver.CollectSourceFilesForTargetTagsWithModuleCache(
		workDir, replCompletionStdRoot(env), []string{"renvo_repl_completion.go"}, replCompletionTarget(), nil,
		replCompletionModuleCache(env), fs,
	)
	if !sources.Ok && sources.Error != driver.SourceErrParse {
		arena.Reset(mark)
		return languageservice.AnalysisResult{}, "", 0, mark, false
	}
	analysis := languageservice.AnalyzeWorkspace(workDir, replCompletionStdRoot(env), ".", sources.Files)
	if !analysis.Workspace.Ok {
		arena.Reset(mark)
		return languageservice.AnalysisResult{}, "", 0, mark, false
	}
	return analysis, fs.sourcePath, query, mark, true
}

func (s *State) completionSource(input string, cursor int) ([]byte, int) {
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(input) {
		cursor = len(input)
	}
	declaration := false
	tokens := syntax.Scan([]byte(input))
	first := replFirstToken(tokens)
	if first >= 0 {
		kind := tokens[first].KindLine & 255
		declaration = kind == syntax.TokenFunc || kind == syntax.TokenType || kind == syntax.TokenConst
	}
	source, at := s.buildSourcePosition(replCloseCompletionInput(input), false, declaration, nil, true)
	if at < 0 {
		return nil, -1
	}
	return source, at + cursor
}

func replCloseCompletionInput(input string) string {
	var stack []byte
	quote := byte(0)
	escaped := false
	lineComment := false
	blockComment := false
	for i := 0; i < len(input); i++ {
		ch := input[i]
		next := byte(0)
		if i+1 < len(input) {
			next = input[i+1]
		}
		if lineComment {
			if ch == '\n' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if ch == '*' && next == '/' {
				blockComment = false
				i++
			}
			continue
		}
		if quote != 0 {
			if quote == '`' {
				if ch == '`' {
					quote = 0
				}
				continue
			}
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '/' && next == '/' {
			lineComment = true
			i++
		} else if ch == '/' && next == '*' {
			blockComment = true
			i++
		} else if ch == '"' || ch == '\'' || ch == '`' {
			quote = ch
		} else if ch == '(' {
			stack = append(stack, ')')
		} else if ch == '[' {
			stack = append(stack, ']')
		} else if ch == '{' {
			stack = append(stack, '}')
		} else if len(stack) > 0 && ch == stack[len(stack)-1] {
			stack = stack[:len(stack)-1]
		}
	}
	out := []byte(input)
	if blockComment {
		out = append(out, '*', '/')
	} else if quote != 0 {
		out = append(out, quote)
	}
	for i := len(stack) - 1; i >= 0; i-- {
		out = append(out, stack[i])
	}
	return string(out)
}

func replHasCompletionModule(workDir string, fs driver.SourceFS) bool {
	dir := workDir
	for {
		if _, ok := fs.ReadFile(load.JoinPath(dir, "go.mod")); ok {
			return true
		}
		next := load.DirPath(dir)
		if next == dir || dir == "." || dir == "/" {
			return false
		}
		dir = next
	}
}

func replStageCompletions(items []check.CompletionItem) []byte {
	count := 0
	for i := 0; i < len(items); i++ {
		if !replCompletionHidden(items[i].Name) {
			count++
		}
	}
	data := replAppendCompletionInt(nil, count)
	for i := 0; i < len(items); i++ {
		if replCompletionHidden(items[i].Name) {
			continue
		}
		data = replAppendCompletionText(data, items[i].Name)
		data = replAppendCompletionText(data, items[i].Detail)
		data = replAppendCompletionText(data, items[i].Signature)
		data = replAppendCompletionInt(data, items[i].Kind)
	}
	return data
}

func replCompletionHidden(name string) bool {
	prefix := "renvo_repl_"
	if len(name) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if name[i] != prefix[i] {
			return false
		}
	}
	return true
}

func replRestoreCompletions(data []byte) []Completion {
	at := 0
	count, ok := replReadCompletionInt(data, &at)
	if !ok || count < 0 || count > len(data) {
		return nil
	}
	out := make([]Completion, 0, count)
	for i := 0; i < count; i++ {
		name, nameOK := replReadCompletionText(data, &at)
		detail, detailOK := replReadCompletionText(data, &at)
		signature, signatureOK := replReadCompletionText(data, &at)
		kind, kindOK := replReadCompletionInt(data, &at)
		if !nameOK || !detailOK || !signatureOK || !kindOK {
			return nil
		}
		out = append(out, Completion{Name: name, Detail: detail, Signature: signature, Kind: kind, ReplaceStart: -1})
	}
	return out
}

func replStageSignature(help check.SignatureHelp) []byte {
	data := replAppendCompletionInt(nil, help.ActiveParameter)
	data = replAppendCompletionText(data, help.Label)
	data = replAppendCompletionInt(data, len(help.Parameters))
	for i := 0; i < len(help.Parameters); i++ {
		data = replAppendCompletionText(data, help.Parameters[i].Name)
		data = replAppendCompletionText(data, help.Parameters[i].Type)
	}
	if help.Ok {
		data = append(data, 1)
	} else {
		data = append(data, 0)
	}
	return data
}

func replRestoreSignature(data []byte) SignatureHelp {
	at := 0
	active, activeOK := replReadCompletionInt(data, &at)
	label, labelOK := replReadCompletionText(data, &at)
	count, countOK := replReadCompletionInt(data, &at)
	if !activeOK || !labelOK || !countOK || count < 0 || count > len(data) {
		return SignatureHelp{}
	}
	parameters := make([]CompletionParameter, 0, count)
	for i := 0; i < count; i++ {
		name, nameOK := replReadCompletionText(data, &at)
		typ, typeOK := replReadCompletionText(data, &at)
		if !nameOK || !typeOK {
			return SignatureHelp{}
		}
		parameters = append(parameters, CompletionParameter{Name: name, Type: typ})
	}
	if at >= len(data) {
		return SignatureHelp{}
	}
	return SignatureHelp{Ok: data[at] != 0, Label: label, Parameters: parameters, ActiveParameter: active}
}

func replAppendCompletionInt(data []byte, value int) []byte {
	return append(data, byte(value), byte(value>>8), byte(value>>16), byte(value>>24))
}

func replAppendCompletionText(data []byte, value string) []byte {
	data = replAppendCompletionInt(data, len(value))
	return append(data, value...)
}

func replReadCompletionInt(data []byte, at *int) (int, bool) {
	if *at < 0 || *at+4 > len(data) {
		return 0, false
	}
	value := int(data[*at])
	value = value | int(data[*at+1])<<8
	value = value | int(data[*at+2])<<16
	value = value | int(data[*at+3])<<24
	*at += 4
	return value, true
}

func replReadCompletionText(data []byte, at *int) (string, bool) {
	length, ok := replReadCompletionInt(data, at)
	if !ok || length < 0 || *at+length > len(data) {
		return "", false
	}
	value := string(data[*at : *at+length])
	*at += length
	return value, true
}
