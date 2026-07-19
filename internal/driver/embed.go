package driver

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

const (
	embedVarInvalid = iota
	embedVarString
	embedVarBytes
	embedVarFS
)

type sourceEmbedDirective struct {
	start    int
	end      int
	patterns []string
}

type sourceEmbedSpec struct {
	declStart int
	insertAt  int
	kind      int
	qualifier string
	patterns  []string
}

type sourceEmbedEdit struct {
	at   int
	text []byte
}

type sourceEmbedPath struct {
	name  string
	isDir bool
}

type sourceEmbedFile struct {
	name string
	data []byte
}

func expandSourceEmbeds(fs SourceFS, path string, moduleRoot string, src []byte) ([]byte, bool, int, string) {
	if !embedSourceContainsDirective(src) {
		return src, true, 0, ""
	}
	directives, directivesOK, directiveError := parseSourceEmbedDirectives(src)
	if !directivesOK {
		return src, false, directiveError, "go:embed"
	}
	if len(directives) == 0 {
		return src, true, 0, ""
	}
	file := syntax.ParseFile(src)
	if !file.Ok {
		return src, true, 0, ""
	}
	embedImport, imported := sourceEmbedImport(file)
	if !imported {
		return src, false, directives[0].start, "embed"
	}
	specs := make([]sourceEmbedSpec, 0, len(directives))
	for i := 0; i < len(directives); i++ {
		declIndex := sourceEmbedDirectiveDecl(file, directives[i])
		if declIndex < 0 {
			return src, false, directives[i].start, "go:embed"
		}
		decl := file.Decls[declIndex]
		specIndex := -1
		for j := 0; j < len(specs); j++ {
			if specs[j].declStart == decl.StartTok {
				specIndex = j
				break
			}
		}
		if specIndex < 0 {
			kind, qualifier, insertAt, ok := sourceEmbedDeclKind(file, decl, embedImport)
			if !ok {
				return src, false, directives[i].start, "go:embed"
			}
			specs = append(specs, sourceEmbedSpec{declStart: decl.StartTok, insertAt: insertAt, kind: kind, qualifier: qualifier})
			specIndex = len(specs) - 1
		}
		specs[specIndex].patterns = append(specs[specIndex].patterns, directives[i].patterns...)
	}
	edits := make([]sourceEmbedEdit, 0, len(specs))
	packageDir := load.DirPath(path)
	for i := 0; i < len(specs); i++ {
		files, ok, failedPattern := resolveSourceEmbedPatterns(fs, packageDir, moduleRoot, specs[i].patterns)
		if !ok {
			offset := directives[0].start
			for j := 0; j < len(directives); j++ {
				if sourceEmbedStringsContain(directives[j].patterns, failedPattern) {
					offset = directives[j].start
					break
				}
			}
			return src, false, offset, failedPattern
		}
		initializer, ok := sourceEmbedInitializer(specs[i], files)
		if !ok {
			return src, false, specs[i].insertAt, "go:embed"
		}
		edits = append(edits, sourceEmbedEdit{at: specs[i].insertAt, text: initializer})
	}
	sortSourceEmbedEdits(edits)
	out := make([]byte, 0, len(src)+sourceEmbedEditSize(edits))
	last := 0
	for i := 0; i < len(edits); i++ {
		if edits[i].at < last || edits[i].at > len(src) {
			return src, false, edits[i].at, "go:embed"
		}
		out = append(out, src[last:edits[i].at]...)
		out = append(out, " = "...)
		out = append(out, edits[i].text...)
		last = edits[i].at
	}
	out = append(out, src[last:]...)
	return out, true, 0, ""
}

func embedSourceContainsDirective(src []byte) bool {
	i := 0
	for i+10 <= len(src) {
		last := src[i+9]
		if last == 'd' {
			if src[i] == '/' && src[i+1] == '/' && src[i+2] == 'g' && src[i+3] == 'o' && src[i+4] == ':' && src[i+5] == 'e' && src[i+6] == 'm' && src[i+7] == 'b' && src[i+8] == 'e' {
				return true
			}
			i += 10
		} else if last == 'e' {
			i++
		} else if last == 'b' {
			i += 2
		} else if last == 'm' {
			i += 3
		} else if last == ':' {
			i += 5
		} else if last == 'o' {
			i += 6
		} else if last == 'g' {
			i += 7
		} else if last == '/' {
			i += 8
		} else {
			i += 10
		}
	}
	return false
}

func parseSourceEmbedDirectives(src []byte) ([]sourceEmbedDirective, bool, int) {
	var directives []sourceEmbedDirective
	for i := 0; i < len(src); {
		if src[i] == '"' || src[i] == '\'' {
			i = sourceEmbedSkipQuoted(src, i, src[i])
			continue
		}
		if src[i] == '`' {
			i++
			for i < len(src) && src[i] != '`' {
				i++
			}
			if i < len(src) {
				i++
			}
			continue
		}
		if src[i] != '/' || i+1 >= len(src) {
			i++
			continue
		}
		if src[i+1] == '*' {
			i += 2
			for i+1 < len(src) && (src[i] != '*' || src[i+1] != '/') {
				i++
			}
			if i+1 < len(src) {
				i += 2
			}
			continue
		}
		if src[i+1] != '/' {
			i++
			continue
		}
		lineEnd := i + 2
		for lineEnd < len(src) && src[lineEnd] != '\n' {
			lineEnd++
		}
		if sourceEmbedOnlyLineSpaceBefore(src, i) {
			const marker = "//go:embed"
			if sourceEmbedBytesEqualText(src, i, lineEnd, marker) && (i+len(marker) == lineEnd || src[i+len(marker)] == ' ' || src[i+len(marker)] == '\t' || src[i+len(marker)] == '\r') {
				patterns, ok := parseSourceEmbedPatterns(src, i+len(marker), lineEnd)
				if !ok || len(patterns) == 0 {
					return directives, false, i
				}
				directives = append(directives, sourceEmbedDirective{start: i, end: lineEnd, patterns: patterns})
			}
		}
		i = lineEnd
	}
	return directives, true, 0
}

func sourceEmbedSkipQuoted(src []byte, start int, quote byte) int {
	start++
	for start < len(src) {
		if src[start] == '\\' {
			start += 2
			continue
		}
		if src[start] == quote {
			return start + 1
		}
		start++
	}
	return start
}

func sourceEmbedOnlyLineSpaceBefore(src []byte, start int) bool {
	for start > 0 && src[start-1] != '\n' {
		start--
		if src[start] != ' ' && src[start] != '\t' && src[start] != '\r' {
			return false
		}
	}
	return true
}

func sourceEmbedBytesEqualText(src []byte, start int, end int, text string) bool {
	if start < 0 || end < start || start+len(text) > end || start+len(text) > len(src) {
		return false
	}
	for i := 0; i < len(text); i++ {
		if src[start+i] != text[i] {
			return false
		}
	}
	return true
}

func parseSourceEmbedPatterns(src []byte, start int, end int) ([]string, bool) {
	var patterns []string
	for start < end {
		for start < end && (src[start] == ' ' || src[start] == '\t' || src[start] == '\r') {
			start++
		}
		if start >= end {
			break
		}
		patternStart := start
		if src[start] == '"' {
			start++
			for start < end && src[start] != '"' {
				if src[start] == '\\' && start+1 < end {
					start += 2
				} else {
					start++
				}
			}
			if start >= end || src[start] != '"' {
				return patterns, false
			}
			start++
			value, ok := syntax.StringLiteralValue(src, syntax.Token{Kind: syntax.TokenString, Start: patternStart, End: start})
			if !ok {
				return patterns, false
			}
			patterns = append(patterns, value)
			continue
		}
		if src[start] == '`' {
			start++
			for start < end && src[start] != '`' {
				start++
			}
			if start >= end || src[start] != '`' {
				return patterns, false
			}
			start++
			patterns = append(patterns, string(src[patternStart+1:start-1]))
			continue
		}
		for start < end && src[start] != ' ' && src[start] != '\t' && src[start] != '\r' {
			start++
		}
		patterns = append(patterns, string(src[patternStart:start]))
	}
	return patterns, true
}

func sourceEmbedImport(file syntax.File) (string, bool) {
	for i := 0; i < len(file.Imports); i++ {
		decl := file.Imports[i]
		path, ok := syntax.StringLiteralValue(file.Src, file.Tokens[decl.PathTok])
		if !ok || path != "embed" {
			continue
		}
		name := "embed"
		if decl.NameTok >= 0 {
			name = string(syntax.TokenText(file.Src, file.Tokens[decl.NameTok]))
		}
		return name, true
	}
	return "", false
}

func sourceEmbedDirectiveDecl(file syntax.File, directive sourceEmbedDirective) int {
	best := -1
	bestStart := len(file.Src) + 1
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind != syntax.TokenVar || decl.StartTok < 0 || decl.StartTok >= len(file.Tokens) {
			continue
		}
		start := file.Tokens[decl.StartTok].Start
		if decl.StartTok > 0 && file.Tokens[decl.StartTok-1].Kind == syntax.TokenVar {
			start = file.Tokens[decl.StartTok-1].Start
		}
		if start < directive.end || start >= bestStart || !sourceEmbedGapAllowed(file.Src, directive.end, start) {
			continue
		}
		best = i
		bestStart = start
	}
	return best
}

func sourceEmbedGapAllowed(src []byte, start int, end int) bool {
	for start < end {
		c := src[start]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			start++
			continue
		}
		if c == '/' && start+1 < end && src[start+1] == '/' {
			start += 2
			for start < end && src[start] != '\n' {
				start++
			}
			continue
		}
		return false
	}
	return true
}

func sourceEmbedDeclKind(file syntax.File, decl syntax.TopDecl, embedImport string) (int, string, int, bool) {
	for i := 0; i < len(file.Decls); i++ {
		if file.Decls[i].StartTok == decl.StartTok && file.Decls[i].NameTok != decl.NameTok {
			return embedVarInvalid, "", 0, false
		}
	}
	start := decl.NameTok + 1
	end := decl.EndTok
	if start >= end || end > len(file.Tokens) {
		return embedVarInvalid, "", 0, false
	}
	for i := start; i < end; i++ {
		if sourceEmbedTokenChar(file, i, '=') {
			return embedVarInvalid, "", 0, false
		}
	}
	insertAt := file.Tokens[end-1].End
	if end-start == 1 && sourceEmbedTokenText(file, start) == "string" {
		return embedVarString, "", insertAt, true
	}
	if end-start == 3 && sourceEmbedTokenChar(file, start, '[') && sourceEmbedTokenChar(file, start+1, ']') && sourceEmbedTokenText(file, start+2) == "byte" {
		return embedVarBytes, "", insertAt, true
	}
	if end-start == 3 && file.Tokens[start].Kind == syntax.TokenIdent && sourceEmbedTokenChar(file, start+1, '.') && sourceEmbedTokenText(file, start+2) == "FS" {
		qualifier := sourceEmbedTokenText(file, start)
		if qualifier == embedImport && qualifier != "_" && qualifier != "." {
			return embedVarFS, qualifier, insertAt, true
		}
	}
	return embedVarInvalid, "", 0, false
}

func sourceEmbedTokenText(file syntax.File, tok int) string {
	if tok < 0 || tok >= len(file.Tokens) {
		return ""
	}
	return string(syntax.TokenText(file.Src, file.Tokens[tok]))
}

func sourceEmbedTokenChar(file syntax.File, tok int, want byte) bool {
	if tok < 0 || tok >= len(file.Tokens) {
		return false
	}
	token := file.Tokens[tok]
	return token.Kind == syntax.TokenOperator && token.End == token.Start+1 && file.Src[token.Start] == want
}

func resolveSourceEmbedPatterns(fs SourceFS, packageDir string, moduleRoot string, patterns []string) ([]sourceEmbedFile, bool, string) {
	paths, ok := collectSourceEmbedPaths(fs, packageDir, moduleRoot)
	if !ok {
		return nil, false, "go:embed"
	}
	var names []string
	for i := 0; i < len(patterns); i++ {
		pattern := patterns[i]
		glob := pattern
		all := false
		if len(glob) > 4 && glob[:4] == "all:" {
			glob = glob[4:]
			all = true
		}
		if !validSourceEmbedPattern(glob) {
			return nil, false, pattern
		}
		matched := 0
		for j := 0; j < len(paths); j++ {
			matches, valid := sourceEmbedPathMatch(glob, paths[j].name)
			if !valid {
				return nil, false, pattern
			}
			if !matches {
				continue
			}
			if paths[j].isDir {
				prefix := paths[j].name + "/"
				for k := 0; k < len(paths); k++ {
					if paths[k].isDir || !sourceEmbedHasPrefix(paths[k].name, prefix) || !sourceEmbedWalkFileAllowed(paths[k].name[len(prefix):], all) {
						continue
					}
					names = appendSourceEmbedUnique(names, paths[k].name)
					matched++
				}
			} else {
				names = appendSourceEmbedUnique(names, paths[j].name)
				matched++
			}
		}
		if matched == 0 {
			return nil, false, pattern
		}
	}
	sortSourceEmbedStrings(names)
	files := make([]sourceEmbedFile, 0, len(names))
	for i := 0; i < len(names); i++ {
		data, readable := fs.ReadFile(load.JoinPath(packageDir, names[i]))
		if !readable {
			return nil, false, names[i]
		}
		files = append(files, sourceEmbedFile{name: names[i], data: data})
	}
	return files, true, ""
}

func collectSourceEmbedPaths(fs SourceFS, packageDir string, moduleRoot string) ([]sourceEmbedPath, bool) {
	var out []sourceEmbedPath
	return collectSourceEmbedDir(fs, packageDir, moduleRoot, "", out)
}

func collectSourceEmbedDir(fs SourceFS, packageDir string, moduleRoot string, relative string, out []sourceEmbedPath) ([]sourceEmbedPath, bool) {
	dir := packageDir
	if relative != "" {
		dir = load.JoinPath(packageDir, relative)
		if _, nested := fs.ReadFile(load.JoinPath(dir, "go.mod")); nested {
			return out, true
		}
	}
	if _, within := load.RelPath(moduleRoot, dir); !within {
		return out, false
	}
	entries, ok := fs.ReadDir(dir)
	if !ok {
		return out, false
	}
	sortDirEntries(entries)
	for i := 0; i < len(entries); i++ {
		name := entries[i].Name
		if name == "" || name == "." || name == ".." || name == "vendor" {
			continue
		}
		rel := name
		if relative != "" {
			rel = relative + "/" + name
		}
		out = append(out, sourceEmbedPath{name: rel, isDir: entries[i].IsDir})
		if entries[i].IsDir {
			var childOK bool
			out, childOK = collectSourceEmbedDir(fs, packageDir, moduleRoot, rel, out)
			if !childOK {
				return out, false
			}
		}
	}
	return out, true
}

func validSourceEmbedPattern(pattern string) bool {
	if pattern == "" || pattern == "." || pattern[0] == '/' || pattern[len(pattern)-1] == '/' {
		return false
	}
	start := 0
	for i := 0; i <= len(pattern); i++ {
		if i < len(pattern) && pattern[i] != '/' {
			continue
		}
		if i == start || i-start == 1 && pattern[start] == '.' || i-start == 2 && pattern[start] == '.' && pattern[start+1] == '.' {
			return false
		}
		start = i + 1
	}
	_, valid := sourceEmbedPathMatch(pattern, "")
	return valid
}

func sourceEmbedPathMatch(pattern string, name string) (bool, bool) {
	patternStart := 0
	nameStart := 0
	for {
		patternEnd := sourceEmbedStringByte(pattern, '/', patternStart)
		nameEnd := sourceEmbedStringByte(name, '/', nameStart)
		matched, valid := sourceEmbedSegmentMatch(pattern[patternStart:patternEnd], name[nameStart:nameEnd])
		if !valid || !matched {
			return false, valid
		}
		patternDone := patternEnd == len(pattern)
		nameDone := nameEnd == len(name)
		if patternDone || nameDone {
			return patternDone && nameDone, true
		}
		patternStart = patternEnd + 1
		nameStart = nameEnd + 1
	}
}

func sourceEmbedSegmentMatch(pattern string, name string) (bool, bool) {
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '*' {
			for i+1 < len(pattern) && pattern[i+1] == '*' {
				i++
			}
			if i+1 == len(pattern) {
				return true, true
			}
			for skipped := 0; skipped <= len(name); skipped++ {
				matched, valid := sourceEmbedSegmentMatch(pattern[i+1:], name[skipped:])
				if !valid {
					return false, false
				}
				if matched {
					return true, true
				}
			}
			return false, true
		}
		if len(name) == 0 {
			return false, sourceEmbedPatternRemainderValid(pattern[i:])
		}
		if pattern[i] == '?' {
			name = name[1:]
			continue
		}
		if pattern[i] == '[' {
			matched, next, valid := sourceEmbedClassMatch(pattern, i+1, name[0])
			if !valid {
				return false, false
			}
			if !matched {
				return false, true
			}
			i = next - 1
			name = name[1:]
			continue
		}
		literal := pattern[i]
		if literal == '\\' {
			i++
			if i >= len(pattern) {
				return false, false
			}
			literal = pattern[i]
		}
		if literal != name[0] {
			return false, sourceEmbedPatternRemainderValid(pattern[i+1:])
		}
		name = name[1:]
	}
	return len(name) == 0, true
}

func sourceEmbedClassMatch(pattern string, start int, value byte) (bool, int, bool) {
	negated := false
	if start < len(pattern) && pattern[start] == '^' {
		negated = true
		start++
	}
	matched := false
	ranges := 0
	for start < len(pattern) && pattern[start] != ']' {
		low, next, ok := sourceEmbedClassChar(pattern, start)
		if !ok {
			return false, start, false
		}
		start = next
		high := low
		if start < len(pattern) && pattern[start] == '-' {
			high, start, ok = sourceEmbedClassChar(pattern, start+1)
			if !ok {
				return false, start, false
			}
		}
		if low <= value && value <= high {
			matched = true
		}
		ranges++
	}
	if start >= len(pattern) || pattern[start] != ']' || ranges == 0 {
		return false, start, false
	}
	if negated {
		matched = !matched
	}
	return matched, start + 1, true
}

func sourceEmbedClassChar(pattern string, start int) (byte, int, bool) {
	if start >= len(pattern) || pattern[start] == ']' || pattern[start] == '-' {
		return 0, start, false
	}
	if pattern[start] == '\\' {
		start++
		if start >= len(pattern) {
			return 0, start, false
		}
	}
	return pattern[start], start + 1, true
}

func sourceEmbedPatternRemainderValid(pattern string) bool {
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '\\' {
			i++
			if i >= len(pattern) {
				return false
			}
		} else if pattern[i] == '[' {
			_, next, valid := sourceEmbedClassMatch(pattern, i+1, 0)
			if !valid {
				return false
			}
			i = next - 1
		}
	}
	return true
}

func sourceEmbedStringByte(value string, want byte, start int) int {
	for start < len(value) && value[start] != want {
		start++
	}
	return start
}

func sourceEmbedHasPrefix(value string, prefix string) bool {
	if len(prefix) > len(value) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if value[i] != prefix[i] {
			return false
		}
	}
	return true
}

func sourceEmbedWalkFileAllowed(relative string, all bool) bool {
	if all {
		return true
	}
	start := 0
	for i := 0; i <= len(relative); i++ {
		if i < len(relative) && relative[i] != '/' {
			continue
		}
		if i > start && (relative[start] == '.' || relative[start] == '_') {
			return false
		}
		start = i + 1
	}
	return true
}

func appendSourceEmbedUnique(values []string, value string) []string {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return values
		}
	}
	return append(values, value)
}

func sortSourceEmbedStrings(values []string) {
	for i := 1; i < len(values); i++ {
		item := values[i]
		j := i - 1
		for j >= 0 && sourceEmbedStringAfter(values[j], item) {
			values[j+1] = values[j]
			j--
		}
		values[j+1] = item
	}
}

func sourceEmbedStringAfter(left string, right string) bool {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for i := 0; i < limit; i++ {
		if left[i] > right[i] {
			return true
		}
		if left[i] < right[i] {
			return false
		}
	}
	return len(left) > len(right)
}

func sourceEmbedInitializer(spec sourceEmbedSpec, files []sourceEmbedFile) ([]byte, bool) {
	if spec.kind == embedVarString || spec.kind == embedVarBytes {
		if len(spec.patterns) != 1 || len(files) != 1 {
			return nil, false
		}
		quoted := quoteSourceEmbedExpression(files[0].data)
		if spec.kind == embedVarString {
			return quoted, true
		}
		out := make([]byte, 0, len(quoted)+8)
		out = append(out, "[]byte("...)
		out = append(out, quoted...)
		out = append(out, ')')
		return out, true
	}
	if spec.kind != embedVarFS || spec.qualifier == "" {
		return nil, false
	}
	archive := buildSourceEmbedArchive(files)
	compressed := compressSourceEmbedArchive(archive)
	quoted := quoteSourceEmbedExpression(compressed)
	out := make([]byte, 0, len(spec.qualifier)+len(quoted)+24)
	out = append(out, spec.qualifier...)
	out = append(out, ".NewFS("...)
	out = append(out, quoted...)
	out = append(out, ',', ' ')
	out = appendSourceEmbedDecimal(out, len(archive))
	out = append(out, ')')
	return out, true
}

func quoteSourceEmbedExpression(data []byte) []byte {
	const chunkSize = 15000
	if len(data) <= chunkSize {
		return quoteSourceEmbedBytes(data)
	}
	out := make([]byte, 0, len(data)+len(data)/chunkSize*4)
	out = append(out, '(')
	for start := 0; start < len(data); start += chunkSize {
		end := start + chunkSize
		if end > len(data) {
			end = len(data)
		}
		if start > 0 {
			out = append(out, " + "...)
		}
		out = append(out, quoteSourceEmbedBytes(data[start:end])...)
	}
	out = append(out, ')')
	return out
}

func buildSourceEmbedArchive(files []sourceEmbedFile) []byte {
	size := 4
	for i := 0; i < len(files); i++ {
		size += 8 + len(files[i].name) + len(files[i].data)
	}
	out := make([]byte, 0, size)
	out = appendSourceEmbedUint32(out, len(files))
	for i := 0; i < len(files); i++ {
		out = appendSourceEmbedUint32(out, len(files[i].name))
		out = appendSourceEmbedUint32(out, len(files[i].data))
		out = append(out, files[i].name...)
		out = append(out, files[i].data...)
	}
	return out
}

func appendSourceEmbedUint32(out []byte, value int) []byte {
	return append(out, byte(value), byte(value>>8), byte(value>>16), byte(value>>24))
}

func appendSourceEmbedDecimal(out []byte, value int) []byte {
	if value == 0 {
		return append(out, '0')
	}
	var reversed []byte
	for value > 0 {
		reversed = append(reversed, byte('0'+value%10))
		value /= 10
	}
	for i := len(reversed) - 1; i >= 0; i-- {
		out = append(out, reversed[i])
	}
	return out
}

func compressSourceEmbedArchive(data []byte) []byte {
	const bucketCount = 65536
	buckets := make([]int32, bucketCount)
	previous := make([]int32, len(data))
	var out []byte
	for pos := 0; pos < len(data); {
		flagPos := len(out)
		out = append(out, 0)
		flags := byte(0)
		for bit := 0; bit < 8 && pos < len(data); bit++ {
			distance, length := sourceEmbedArchiveMatch(data, buckets, previous, pos)
			if length >= 3 {
				pair := (distance-1)<<4 | (length - 3)
				out = append(out, byte(pair>>8), byte(pair))
				for i := 0; i < length; i++ {
					sourceEmbedArchiveAddPosition(data, buckets, previous, pos+i)
				}
				pos += length
				continue
			}
			flags |= 1 << bit
			out = append(out, data[pos])
			sourceEmbedArchiveAddPosition(data, buckets, previous, pos)
			pos++
		}
		out[flagPos] = flags
	}
	return out
}

func sourceEmbedArchiveAddPosition(data []byte, buckets []int32, previous []int32, pos int) {
	if pos+2 >= len(data) {
		return
	}
	bucket := sourceEmbedArchiveBucket(data, pos, len(buckets))
	previous[pos] = buckets[bucket]
	buckets[bucket] = int32(pos + 1)
}

func sourceEmbedArchiveMatch(data []byte, buckets []int32, previous []int32, pos int) (int, int) {
	if pos+2 >= len(data) {
		return 0, 0
	}
	bestDistance := 0
	bestLength := 0
	checked := 0
	bucket := sourceEmbedArchiveBucket(data, pos, len(buckets))
	for candidate := int(buckets[bucket]) - 1; candidate >= 0 && checked < 128; candidate = int(previous[candidate]) - 1 {
		distance := pos - candidate
		if distance > 4096 {
			break
		}
		checked++
		if data[candidate] != data[pos] || data[candidate+1] != data[pos+1] || data[candidate+2] != data[pos+2] {
			continue
		}
		length := 0
		for length < 18 && pos+length < len(data) && data[candidate+length] == data[pos+length] {
			length++
		}
		if length > bestLength {
			bestDistance = distance
			bestLength = length
			if length == 18 {
				break
			}
		}
	}
	return bestDistance, bestLength
}

func sourceEmbedArchiveBucket(data []byte, pos int, bucketCount int) int {
	hash := (int(data[pos])*251+int(data[pos+1]))*251 + int(data[pos+2])
	return hash & (bucketCount - 1)
}

func quoteSourceEmbedBytes(data []byte) []byte {
	out := make([]byte, 0, len(data)+2)
	out = append(out, '"')
	const hex = "0123456789abcdef"
	for i := 0; i < len(data); i++ {
		c := data[i]
		if c == '\\' || c == '"' {
			out = append(out, '\\', c)
		} else if c >= 32 && c <= 126 {
			out = append(out, c)
		} else {
			out = append(out, '\\', 'x', hex[c>>4], hex[c&15])
		}
	}
	out = append(out, '"')
	return out
}

func sortSourceEmbedEdits(edits []sourceEmbedEdit) {
	for i := 1; i < len(edits); i++ {
		item := edits[i]
		j := i - 1
		for j >= 0 && edits[j].at > item.at {
			edits[j+1] = edits[j]
			j--
		}
		edits[j+1] = item
	}
}

func sourceEmbedEditSize(edits []sourceEmbedEdit) int {
	size := 0
	for i := 0; i < len(edits); i++ {
		size += len(edits[i].text) + 3
	}
	return size
}

func sourceEmbedStringsContain(values []string, value string) bool {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return true
		}
	}
	return false
}
