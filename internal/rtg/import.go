package rtg

// ParseImports expands top-level @import declarations and parses the resulting
// closed definition. Import paths are intentionally just strings here: the
// caller owns filesystem and path policy through ImportLoader.
func ParseImports(source []byte, filename string, loader ImportLoader) Document {
	builder := importBuilder{loader: loader}
	if !builder.expand(source, filename) {
		return Document{
			Filename:    filename,
			Source:      source,
			Diagnostics: builder.diagnostics,
		}
	}
	return parseDocument(builder.source, filename, builder.segments)
}

type importBuilder struct {
	loader      ImportLoader
	source      []byte
	segments    []sourceSegment
	loading     []string
	diagnostics []Diagnostic
}

func (b *importBuilder) expand(source []byte, filename string) bool {
	if len(source) > maxDefinitionBytes {
		b.fail(filename, source, 0, 0, "RTG-LIMIT-001",
			"machine definition exceeds the 16 MiB source limit")
		return false
	}
	if invalid := invalidUTF8Offset(source); invalid >= 0 {
		b.fail(filename, source, invalid, invalid+1, "RTG-SCAN-003",
			"machine definition is not valid UTF-8")
		return false
	}
	tokens, diagnostics := scan(source, filename)
	if len(diagnostics) != 0 {
		b.diagnostics = append(b.diagnostics, diagnostics...)
		return false
	}
	b.loading = append(b.loading, filename)
	defer func() { b.loading = b.loading[:len(b.loading)-1] }()

	start := 0
	depth := 0
	for at := 0; at < len(tokens) && tokens[at].Kind != TokenEOF; at++ {
		text := tokenText(source, tokens[at])
		if text == "{" {
			depth++
			continue
		}
		if text == "}" {
			if depth > 0 {
				depth--
			}
			continue
		}
		if text != "@" {
			continue
		}
		if depth != 0 {
			continue
		}
		if at+2 >= len(tokens) || tokens[at+1].Kind != TokenIdent ||
			tokenText(source, tokens[at+1]) != "import" ||
			tokens[at+2].Kind != TokenString {
			b.fail(filename, source, tokens[at].Start, tokens[at].End,
				"RTG-IMPORT-002", `expected @import "relative/path.rtg"`)
			return false
		}
		path := unquoteSimple(tokenText(source, tokens[at+2]))
		if !validImportPath(path) {
			b.fail(filename, source, tokens[at+2].Start, tokens[at+2].End,
				"RTG-IMPORT-003", "import path must be a portable relative path")
			return false
		}
		if b.loader == nil {
			b.fail(filename, source, tokens[at].Start, tokens[at+2].End,
				"RTG-IMPORT-004", "definition imports require an import loader")
			return false
		}
		imported := b.loader.LoadImport(filename, path)
		if !imported.Ok || imported.Filename == "" {
			b.fail(filename, source, tokens[at].Start, tokens[at+2].End,
				"RTG-IMPORT-005", "could not import "+path)
			return false
		}
		for i := 0; i < len(b.loading); i++ {
			if b.loading[i] == imported.Filename {
				b.fail(filename, source, tokens[at].Start, tokens[at+2].End,
					"RTG-IMPORT-006", "import cycle includes "+imported.Filename)
				return false
			}
		}

		if !b.appendSource(source[start:tokens[at].Start], filename, source, start) ||
			!b.appendSource([]byte("\n"), filename, source, tokens[at].Start) {
			return false
		}
		if !b.expand(imported.Source, imported.Filename) {
			return false
		}
		if !b.appendSource([]byte("\n"), filename, source, tokens[at+2].End) {
			return false
		}
		start = tokens[at+2].End
		at += 2
	}
	return b.appendSource(source[start:], filename, source, start)
}

func validImportPath(path string) bool {
	if path == "" || path[0] == '/' || path[0] == '\\' ||
		len(path) >= 2 && path[1] == ':' {
		return false
	}
	for i := 0; i < len(path); i++ {
		if path[i] == '\\' || path[i] == 0 || path[i] == '\n' || path[i] == '\r' {
			return false
		}
	}
	return true
}

func (b *importBuilder) appendSource(
	data []byte, filename string, source []byte, sourceStart int,
) bool {
	if len(data) == 0 {
		return true
	}
	if len(data) > maxDefinitionBytes-len(b.source) {
		b.fail(filename, source, sourceStart, sourceStart, "RTG-LIMIT-001",
			"machine definition exceeds the 16 MiB source limit")
		return false
	}
	logicalStart := len(b.source)
	b.source = append(b.source, data...)
	b.segments = append(b.segments, sourceSegment{
		logicalStart: logicalStart,
		logicalEnd:   len(b.source),
		sourceStart:  sourceStart,
		filename:     filename,
		source:       source,
	})
	return true
}

func (b *importBuilder) fail(
	filename string, source []byte, start int, end int, code string, message string,
) {
	b.diagnostics = append(b.diagnostics, Diagnostic{
		Filename: filename,
		Span:     sourceSpan(source, start, end),
		Code:     code,
		Message:  message,
	})
}
