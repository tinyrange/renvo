package main

import (
	"renvo.dev/ide"
	"renvo.dev/internal/arena"
	"renvo.dev/internal/check"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/load"
)

type completionOverlayFS struct {
	path string
	data []byte
}

func (fs completionOverlayFS) ReadDir(path string) ([]driver.DirEntry, bool) {
	return completionReadDir(path)
}

func (fs completionOverlayFS) ReadFile(path string) ([]byte, bool) {
	if load.CleanPath(path) == fs.path {
		return fs.data, true
	}
	return completionReadFile(path)
}

func (fs completionOverlayFS) PathExists(path string) bool {
	_, ok := fs.ReadFile(path)
	return ok
}

func (f *MainForm) completeEditor(source []byte, caret int) []ide.Completion {
	mark := arena.Mark()
	result, ok := f.analyzeEditorSource(source)
	if !ok || !result.Workspace.Ok {
		arena.Reset(mark)
		return nil
	}
	path := load.CleanPath(f.currentPath)
	queryAt := completionQueryStart(source, caret)
	var semantic []check.CompletionItem
	if result.Program.Ok {
		semantic = check.CompleteProgram(result.Workspace.Graph, result.Program, path, queryAt)
	} else {
		semantic = check.CompleteGraph(result.Workspace.Graph, path, queryAt)
	}
	staged := arena.PersistBytes(stageCompletionItems(semantic))
	arena.Reset(mark)
	return restoreCompletionItems(staged)
}

func completionQueryStart(source []byte, caret int) int {
	if caret > len(source) {
		caret = len(source)
	}
	if caret < 0 {
		caret = 0
	}
	for caret > 0 {
		value := source[caret-1]
		if value != '_' && (value < 'a' || value > 'z') && (value < 'A' || value > 'Z') && (value < '0' || value > '9') {
			break
		}
		caret--
	}
	return caret
}

func (f *MainForm) signatureEditor(source []byte, caret int, out *ide.SignatureHelp) {
	mark := arena.Mark()
	result, ok := f.analyzeEditorSource(source)
	if !ok || !result.Workspace.Ok {
		arena.Reset(mark)
		return
	}
	var help check.SignatureHelp
	if result.Program.Ok {
		help = check.SignatureHelpProgram(result.Workspace.Graph, result.Program, load.CleanPath(f.currentPath), caret)
	} else {
		help = check.SignatureHelpGraph(result.Workspace.Graph, load.CleanPath(f.currentPath), caret)
	}
	staged := arena.PersistBytes(stageSignatureHelp(help))
	arena.Reset(mark)
	*out = restoreSignatureHelp(staged)
}

func stageCompletionItems(items []check.CompletionItem) []byte {
	data := appendCompletionStageInt(nil, len(items))
	for i := 0; i < len(items); i++ {
		data = appendCompletionStageText(data, items[i].Name)
		data = appendCompletionStageText(data, items[i].Detail)
		data = appendCompletionStageInt(data, items[i].Kind)
		data = appendCompletionStageText(data, items[i].Signature)
		data = appendCompletionStageInt(data, len(items[i].Parameters))
		for j := 0; j < len(items[i].Parameters); j++ {
			data = appendCompletionStageText(data, items[i].Parameters[j].Name)
			data = appendCompletionStageText(data, items[i].Parameters[j].Type)
		}
	}
	return data
}

func restoreCompletionItems(data []byte) []ide.Completion {
	at := 0
	count, ok := readCompletionStageInt(data, &at)
	if !ok || count < 0 || count > len(data) {
		return nil
	}
	items := make([]ide.Completion, 0, count)
	for i := 0; i < count; i++ {
		name, nameOK := readCompletionStageText(data, &at)
		detail, detailOK := readCompletionStageText(data, &at)
		kind, kindOK := readCompletionStageInt(data, &at)
		signature, signatureOK := readCompletionStageText(data, &at)
		parameterCount, parametersOK := readCompletionStageInt(data, &at)
		if !nameOK || !detailOK || !kindOK || !signatureOK || !parametersOK || parameterCount < 0 || parameterCount > len(data) {
			return nil
		}
		parameters := make([]ide.SignatureParameter, 0, parameterCount)
		for j := 0; j < parameterCount; j++ {
			name, firstOK := readCompletionStageText(data, &at)
			typ, secondOK := readCompletionStageText(data, &at)
			if !firstOK || !secondOK {
				return nil
			}
			parameters = append(parameters, ide.SignatureParameter{Name: name, Type: typ})
		}
		items = append(items, ide.Completion{Text: name, Detail: detail, Kind: kind, Signature: signature, Parameters: parameters})
	}
	return items
}

func stageSignatureHelp(help check.SignatureHelp) []byte {
	data := appendCompletionStageInt(nil, help.ActiveParameter)
	data = appendCompletionStageText(data, help.Label)
	data = appendCompletionStageInt(data, len(help.Parameters))
	for i := 0; i < len(help.Parameters); i++ {
		data = appendCompletionStageText(data, help.Parameters[i].Name)
		data = appendCompletionStageText(data, help.Parameters[i].Type)
	}
	if help.Ok {
		data = append(data, 1)
	} else {
		data = append(data, 0)
	}
	return data
}

func restoreSignatureHelp(data []byte) ide.SignatureHelp {
	at := 0
	active, activeOK := readCompletionStageInt(data, &at)
	label, labelOK := readCompletionStageText(data, &at)
	count, countOK := readCompletionStageInt(data, &at)
	if !activeOK || !labelOK || !countOK || count < 0 || count > len(data) {
		return ide.SignatureHelp{}
	}
	parameters := make([]ide.SignatureParameter, 0, count)
	for i := 0; i < count; i++ {
		name, nameOK := readCompletionStageText(data, &at)
		typ, typeOK := readCompletionStageText(data, &at)
		if !nameOK || !typeOK {
			return ide.SignatureHelp{}
		}
		parameters = append(parameters, ide.SignatureParameter{Name: name, Type: typ})
	}
	if at >= len(data) {
		return ide.SignatureHelp{}
	}
	return ide.SignatureHelp{Ok: data[at] != 0, Label: label, Parameters: parameters, ActiveParameter: active}
}

func appendCompletionStageInt(data []byte, value int) []byte {
	return append(data, byte(value), byte(value>>8), byte(value>>16), byte(value>>24))
}

func appendCompletionStageText(data []byte, value string) []byte {
	data = appendCompletionStageInt(data, len(value))
	return append(data, value...)
}

func readCompletionStageInt(data []byte, at *int) (int, bool) {
	if *at < 0 || *at+4 > len(data) {
		return 0, false
	}
	value := int(data[*at]) | int(data[*at+1])<<8 | int(data[*at+2])<<16 | int(data[*at+3])<<24
	if value >= 1<<31 {
		value -= 1 << 32
	}
	*at += 4
	return value, true
}

func readCompletionStageText(data []byte, at *int) (string, bool) {
	length, ok := readCompletionStageInt(data, at)
	if !ok || length < 0 || *at+length > len(data) {
		return "", false
	}
	value := string(data[*at : *at+length])
	*at += length
	return value, true
}

func completionEnv(env []string, key string) string {
	prefix := key + "="
	for i := 0; i < len(env); i++ {
		if workspaceHasPrefix(env[i], prefix) {
			return env[i][len(prefix):]
		}
	}
	return ""
}
