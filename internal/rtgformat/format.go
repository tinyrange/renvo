// Package rtgformat formats and validates Renvo target-generation and backend
// enablement source files.
package rtgformat

import (
	"bytes"
	"fmt"
	"go/format"
	"path/filepath"
	"strings"

	"renvo.dev/internal/rbe"
	"renvo.dev/internal/rtg"
)

const goPackagePrefix = "package backend\n"

// Source validates and formats one RTG or RBE source file. Imports are resolved
// through loader so the same parser and embedded-Go checks used by generation
// remain the syntax authority.
func Source(source []byte, filename string, loader rtg.ImportLoader) ([]byte, error) {
	bundle := rbe.Parse(source)
	if !bundle.Ok {
		line, column := sourcePosition(source, bundle.Offset)
		return nil, fmt.Errorf("%s:%d:%d: %s", filename, line, column, bundle.Message)
	}
	_, err := validateDefinition(bundle.Definition, filename, loader)
	if err != nil {
		return nil, err
	}

	definition, err := formatDefinition(bundle.Definition, filename)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.Write(definition)
	if len(bundle.Files) != 0 {
		ensureNewline(&out)
		if out.Len() != 0 && !bytes.HasSuffix(out.Bytes(), []byte("\n\n")) {
			out.WriteByte('\n')
		}
	}
	for i := range bundle.Files {
		file := bundle.Files[i]
		formatted, formatErr := format.Source(file.Source)
		if formatErr != nil {
			return nil, fmt.Errorf("%s: @stdlib %q: %w", filename, file.Path, formatErr)
		}
		fmt.Fprintf(&out, "@stdlib %q\n", file.Path)
		out.Write(formatted)
		ensureNewline(&out)
		out.WriteString("@endstdlib\n")
		if i+1 < len(bundle.Files) {
			out.WriteByte('\n')
		}
	}

	formatted := out.Bytes()
	formattedBundle := rbe.Parse(formatted)
	if !formattedBundle.Ok {
		return nil, fmt.Errorf("%s: formatter produced invalid RBE: %s", filename, formattedBundle.Message)
	}
	_, err = validateDefinition(formattedBundle.Definition, filename, loader)
	if err != nil {
		return nil, fmt.Errorf("formatter produced invalid RTG: %w", err)
	}
	return append([]byte(nil), formatted...), nil
}

func validateDefinition(source []byte, filename string, loader rtg.ImportLoader) (rtg.Document, error) {
	document := rtg.ParseImports(source, filename, loader)
	for _, diagnostic := range document.Diagnostics {
		// Imported fragments deliberately omit the closed-definition header. They
		// are independently formattable, while their importing root supplies and
		// validates these fields.
		if diagnostic.Code == "RTG-VALIDATE-001" ||
			diagnostic.Code == "RTG-VALIDATE-002" ||
			diagnostic.Code == "RTG-VALIDATE-003" {
			continue
		}
		return document, diagnosticError(diagnostic)
	}
	return document, nil
}

func diagnosticError(diagnostic rtg.Diagnostic) error {
	return fmt.Errorf("%s:%d:%d: %s: %s", diagnostic.Filename,
		diagnostic.Span.Start.Line, diagnostic.Span.Start.Column,
		diagnostic.Code, diagnostic.Message)
}

func formatDefinition(source []byte, filename string) ([]byte, error) {
	// Imports are validated above. Blank them with equal-width spaces so the
	// local parser can expose Go declaration byte spans without expanding files
	// or changing any source offsets.
	masked := maskImports(source)
	document := rtg.Parse(masked, filename)
	var goDeclarations []rtg.Declaration
	for _, declaration := range document.Declarations {
		if declaration.Kind == rtg.DeclGo {
			goDeclarations = append(goDeclarations, declaration)
		}
	}

	var out bytes.Buffer
	depth := 0
	start := 0
	for _, declaration := range goDeclarations {
		if declaration.Start < start || declaration.End > len(source) {
			return nil, fmt.Errorf("%s: invalid embedded Go source span", filename)
		}
		formatMetadata(&out, source[start:declaration.Start], &depth)
		body, err := formatGoBody(source[declaration.BodyStart:declaration.BodyEnd])
		if err != nil {
			line, column := sourcePosition(source, declaration.BodyStart)
			return nil, fmt.Errorf("%s:%d:%d: embedded Go: %w", filename, line, column, err)
		}
		out.WriteString("go ")
		out.WriteString(declaration.Name)
		out.WriteString(" {\n")
		writeIndentedGo(&out, body)
		out.WriteString("}\n")
		start = declaration.End
	}
	formatMetadata(&out, source[start:], &depth)
	result := trimBlankLines(out.Bytes())
	if len(result) != 0 {
		result = append(result, '\n')
	}
	return result, nil
}

func formatGoBody(body []byte) ([]byte, error) {
	wrapper := make([]byte, 0, len(goPackagePrefix)+len(body))
	wrapper = append(wrapper, goPackagePrefix...)
	wrapper = append(wrapper, body...)
	formatted, err := format.Source(wrapper)
	if err != nil {
		return nil, err
	}
	formatted = bytes.TrimPrefix(formatted, []byte(goPackagePrefix))
	formatted = bytes.TrimPrefix(formatted, []byte("\n"))
	return bytes.TrimRight(formatted, "\n"), nil
}

func writeIndentedGo(out *bytes.Buffer, body []byte) {
	if len(body) == 0 {
		return
	}
	for _, line := range bytes.Split(body, []byte("\n")) {
		if len(line) != 0 {
			out.WriteByte('\t')
			out.Write(line)
		}
		out.WriteByte('\n')
	}
}

func maskImports(source []byte) []byte {
	masked := append([]byte(nil), source...)
	at := 0
	for at < len(masked) {
		end := bytes.IndexByte(masked[at:], '\n')
		if end < 0 {
			end = len(masked)
		} else {
			end += at
		}
		line := masked[at:end]
		if bytes.HasPrefix(bytes.TrimSpace(line), []byte("@import")) {
			for i := at; i < end; i++ {
				if masked[i] != '\r' {
					masked[i] = ' '
				}
			}
		}
		if end == len(masked) {
			break
		}
		at = end + 1
	}
	return masked
}

func formatMetadata(out *bytes.Buffer, source []byte, depth *int) {
	lines := bytes.Split(source, []byte("\n"))
	inBlockComment := false
	for i, raw := range lines {
		if i == len(lines)-1 && len(raw) == 0 {
			continue
		}
		line := strings.TrimSpace(strings.TrimSuffix(string(raw), "\r"))
		if line == "" {
			writeBlankLine(out)
			continue
		}
		if inBlockComment || strings.HasPrefix(line, "/*") {
			out.WriteString(strings.Repeat("\t", *depth))
			out.WriteString(line)
			out.WriteByte('\n')
			if inBlockComment {
				inBlockComment = !strings.Contains(line, "*/")
			} else {
				inBlockComment = !strings.Contains(line[2:], "*/")
			}
			continue
		}
		tokens := metadataTokens(line)
		if formatInlineRelocation(out, tokens, *depth) {
			continue
		}
		leadingClose := 0
		for leadingClose < len(tokens) && (tokens[leadingClose] == "}" || tokens[leadingClose] == "]") {
			leadingClose++
		}
		indent := *depth - leadingClose
		if indent < 0 {
			indent = 0
		}
		out.WriteString(strings.Repeat("\t", indent))
		out.WriteString(joinMetadataTokens(tokens))
		out.WriteByte('\n')
		for at, token := range tokens {
			switch token {
			case "{":
				*depth++
			case "[":
				if metadataListOpener(tokens, at) {
					*depth++
				}
			case "}":
				if *depth > 0 {
					*depth--
				}
			case "]":
				if !metadataTokenBefore(tokens, at, "[") && *depth > 0 {
					*depth--
				}
			}
		}
	}
}

func metadataTokenBefore(tokens []string, at int, want string) bool {
	for i := 0; i < at; i++ {
		if tokens[i] == want {
			return true
		}
	}
	return false
}

func metadataListOpener(tokens []string, at int) bool {
	if at < 0 || at >= len(tokens) || tokens[at] != "[" {
		return false
	}
	for i := at + 1; i < len(tokens); i++ {
		if tokens[i] == "]" || tokens[i] == ")" {
			return false
		}
	}
	return true
}

func formatInlineRelocation(out *bytes.Buffer, tokens []string, depth int) bool {
	if len(tokens) < 7 || tokens[0] != "relocation" || tokens[len(tokens)-1] != "}" {
		return false
	}
	open := -1
	for i := 1; i < len(tokens)-1; i++ {
		if tokens[i] == "{" {
			open = i
			break
		}
	}
	if open < 2 {
		return false
	}
	var fields [][]string
	for at := open + 1; at < len(tokens)-1; {
		if at+2 >= len(tokens)-1 || !metadataIdentifier(tokens[at]) || tokens[at+1] != "=" {
			return false
		}
		end := at + 3
		for end < len(tokens)-1 {
			if end+1 < len(tokens)-1 && metadataIdentifier(tokens[end]) && tokens[end+1] == "=" {
				break
			}
			end++
		}
		fields = append(fields, tokens[at:end])
		at = end
	}
	if len(fields) == 0 {
		return false
	}
	out.WriteString(strings.Repeat("\t", depth))
	out.WriteString(joinMetadataTokens(tokens[:open+1]))
	out.WriteByte('\n')
	for _, field := range fields {
		out.WriteString(strings.Repeat("\t", depth+1))
		out.WriteString(joinMetadataTokens(field))
		out.WriteByte('\n')
	}
	out.WriteString(strings.Repeat("\t", depth))
	out.WriteString("}\n")
	return true
}

func metadataIdentifier(token string) bool {
	if token == "" || !isIdentStart(token[0]) {
		return false
	}
	for i := 1; i < len(token); i++ {
		if !isIdentPart(token[i]) {
			return false
		}
	}
	return true
}

func metadataTokens(line string) []string {
	var tokens []string
	for at := 0; at < len(line); {
		if line[at] == ' ' || line[at] == '\t' || line[at] == '\r' {
			at++
			continue
		}
		if line[at] == '#' || line[at] == '/' && at+1 < len(line) && line[at+1] == '/' {
			tokens = append(tokens, strings.TrimRight(line[at:], " \t\r"))
			break
		}
		if line[at] == '/' && at+1 < len(line) && line[at+1] == '*' {
			end := strings.Index(line[at+2:], "*/")
			if end < 0 {
				tokens = append(tokens, strings.TrimRight(line[at:], " \t\r"))
				break
			}
			end += at + 4
			tokens = append(tokens, line[at:end])
			at = end
			continue
		}
		start := at
		if isIdentStart(line[at]) {
			at++
			for at < len(line) && isIdentPart(line[at]) {
				at++
			}
		} else if line[at] >= '0' && line[at] <= '9' {
			at++
			for at < len(line) && (isIdentPart(line[at]) || line[at] == '.') {
				at++
			}
		} else if line[at] == '"' || line[at] == '\'' || line[at] == '`' {
			quote := line[at]
			at++
			for at < len(line) {
				if quote != '`' && line[at] == '\\' && at+1 < len(line) {
					at += 2
					continue
				}
				at++
				if line[at-1] == quote {
					break
				}
			}
		} else {
			at++
			if at < len(line) && metadataPair(line[start], line[at]) {
				at++
			}
		}
		tokens = append(tokens, line[start:at])
	}
	return tokens
}

func metadataPair(first, second byte) bool {
	return first == '=' && second == '>' || first == '<' && second == '<' ||
		first == '>' && second == '>' || first == '&' && second == '&' ||
		first == '|' && second == '|' || first == '!' && second == '=' ||
		first == '=' && second == '=' || first == '<' && second == '=' ||
		first == '>' && second == '='
}

func joinMetadataTokens(tokens []string) string {
	var out strings.Builder
	for i, token := range tokens {
		if i != 0 && metadataSpace(tokens, i) {
			out.WriteByte(' ')
		}
		out.WriteString(token)
	}
	return out.String()
}

func metadataSpace(tokens []string, at int) bool {
	previous := tokens[at-1]
	current := tokens[at]
	if strings.HasPrefix(current, "#") || strings.HasPrefix(current, "//") {
		return true
	}
	if typedWordFormTokens(tokens) &&
		(previous == "@" || current == "@" || previous == "+" || current == "+") {
		return false
	}
	if previous == "!" {
		return false
	}
	if previous == "-" && current == ">" || previous == ">" && current == "-" {
		return false
	}
	if targetNameToken(tokens, at-1) || targetNameToken(tokens, at) {
		if previous == "-" || current == "-" {
			return false
		}
	}
	if previous == "-" && at >= 2 && unaryOperatorPrecedes(tokens[at-2]) {
		return false
	}
	if current == "." || previous == "." || current == "/" || previous == "/" ||
		previous == "@" ||
		current == "," || current == ")" || current == "]" {
		return false
	}
	if current == ":" || previous == ":" {
		return ternaryColon(tokens, at)
	}
	if previous == "(" || previous == "[" {
		return false
	}
	if current == "(" {
		return false
	}
	return true
}

func ternaryColon(tokens []string, at int) bool {
	colon := at
	if tokens[at-1] == ":" {
		colon = at - 1
	}
	for i := 0; i < colon; i++ {
		if tokens[i] == "?" {
			return true
		}
	}
	return false
}

func typedWordFormTokens(tokens []string) bool {
	for i := 0; i+1 < len(tokens); i++ {
		if (tokens[i] == "word16" || tokens[i] == "word32") && tokens[i+1] == "(" {
			return true
		}
	}
	return false
}

func targetNameToken(tokens []string, at int) bool {
	if len(tokens) == 0 || tokens[0] != "target" || at <= 0 || at >= len(tokens) {
		return false
	}
	for i := 1; i <= at; i++ {
		if tokens[i] == "{" {
			return false
		}
	}
	return true
}

func unaryOperatorPrecedes(token string) bool {
	return token == "=" || token == "[" || token == "(" || token == "{" ||
		token == "," || token == ":" || token == "=>"
}

func writeBlankLine(out *bytes.Buffer) {
	if out.Len() != 0 && !bytes.HasSuffix(out.Bytes(), []byte("\n\n")) {
		out.WriteByte('\n')
	}
}

func trimBlankLines(source []byte) []byte {
	return bytes.Trim(source, "\n")
}

func ensureNewline(out *bytes.Buffer) {
	if out.Len() != 0 && out.Bytes()[out.Len()-1] != '\n' {
		out.WriteByte('\n')
	}
}

func sourcePosition(source []byte, offset int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(source) {
		offset = len(source)
	}
	line, column := 1, 1
	for _, ch := range source[:offset] {
		if ch == '\n' {
			line, column = line+1, 1
		} else {
			column++
		}
	}
	return line, column
}

func isIdentStart(ch byte) bool {
	return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch == '_'
}

func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || ch >= '0' && ch <= '9'
}

// Extension reports whether path is a source format handled by this package.
func Extension(path string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	return extension == ".rtg" || extension == ".rbe"
}
