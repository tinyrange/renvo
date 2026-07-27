package rtg

import (
	"strings"

	"renvo.dev/internal/syntax"
)

func Parse(source []byte, filename string) Document {
	tokens, diagnostics := scan(source, filename)
	document := Document{
		Filename:    filename,
		Source:      source,
		Tokens:      tokens,
		Diagnostics: diagnostics,
		Ok:          len(diagnostics) == 0,
	}
	if !document.Ok {
		return document
	}
	parser := documentParser{document: &document}
	parser.parse()
	if len(document.Diagnostics) == 0 {
		if diagnostic, ok := validateEmbeddedGoTypes(document); !ok {
			document.Diagnostics = append(document.Diagnostics, diagnostic)
		}
	}
	if len(document.Diagnostics) == 0 {
		canonical := Canonical(document)
		document.Hash = sha256Bytes(canonical)
		document.Ok = true
	} else {
		document.Ok = false
	}
	return document
}

type documentParser struct {
	document *Document
	at       int
	goNames  []string
}

func (p *documentParser) parse() {
	for p.kind() != TokenEOF {
		if p.kind() != TokenIdent {
			p.failAt(p.at, "RTG-PARSE-001", "expected a top-level declaration")
			return
		}
		keyword := p.text(p.at)
		if keyword == "definition" {
			p.parseVersion()
		} else if keyword == "unit" {
			p.parseUnit()
		} else if keyword == "implements" {
			p.parseImplements()
		} else if keyword == DeclSystem || keyword == DeclArch || keyword == DeclABI ||
			keyword == DeclRuntime || keyword == DeclFormat || keyword == DeclTarget || keyword == DeclIR {
			p.parseDeclaration(keyword)
		} else if keyword == DeclGo {
			p.parseGo()
		} else {
			p.failAt(p.at, "RTG-PARSE-002", "unknown top-level declaration "+keyword)
			return
		}
		if len(p.document.Diagnostics) > 0 {
			return
		}
	}
	hasMachine := false
	for i := 0; i < len(p.document.Declarations); i++ {
		if p.document.Declarations[i].Kind != DeclSystem {
			hasMachine = true
			break
		}
	}
	if hasMachine {
		if p.document.Version == 0 {
			p.failOffset(0, "RTG-VALIDATE-001", "machine definition is missing definition version")
		}
		if p.document.Unit == "" {
			p.failOffset(0, "RTG-VALIDATE-002", "machine definition is missing unit name")
		}
		if len(p.document.Implements) == 0 {
			p.failOffset(0, "RTG-VALIDATE-003", "machine definition is missing implements contract")
		}
	}
}

func (p *documentParser) parseVersion() {
	start := p.at
	p.at++
	if p.document.Version != 0 {
		p.failAt(start, "RTG-PARSE-003", "duplicate definition version")
		return
	}
	if p.kind() != TokenNumber {
		p.failAt(p.at, "RTG-PARSE-004", "expected numeric definition version")
		return
	}
	version, ok := parsePositiveDecimal(p.text(p.at))
	if !ok || version != 1 {
		p.failAt(p.at, "RTG-PARSE-005", "unsupported definition version "+p.text(p.at))
		return
	}
	p.document.Version = version
	p.at++
}

func (p *documentParser) parseUnit() {
	start := p.at
	p.at++
	if p.document.Unit != "" {
		p.failAt(start, "RTG-PARSE-006", "duplicate unit declaration")
		return
	}
	if p.kind() != TokenIdent {
		p.failAt(p.at, "RTG-PARSE-007", "expected unit identifier")
		return
	}
	p.document.Unit = p.text(p.at)
	p.at++
}

func (p *documentParser) parseImplements() {
	p.at++
	if p.kind() != TokenIdent {
		p.failAt(p.at, "RTG-PARSE-008", "expected contract identifier")
		return
	}
	name := p.text(p.at)
	for i := 0; i < len(p.document.Implements); i++ {
		if p.document.Implements[i] == name {
			p.failAt(p.at, "RTG-PARSE-009", "duplicate implements contract "+name)
			return
		}
	}
	p.document.Implements = append(p.document.Implements, name)
	p.at++
}

func (p *documentParser) parseDeclaration(kind string) {
	startToken := p.at
	p.at++
	if p.kind() == TokenEOF {
		p.failAt(p.at, "RTG-PARSE-010", "missing declaration name")
		return
	}
	nameStart := p.at
	for p.kind() != TokenEOF && !p.operator(p.at, "{") {
		p.at++
	}
	if p.kind() == TokenEOF || p.at == nameStart {
		p.failAt(p.at, "RTG-PARSE-011", "expected declaration name and body")
		return
	}
	name := declarationName(p.document.Source, p.document.Tokens[nameStart:p.at])
	if name == "" {
		p.failAt(nameStart, "RTG-PARSE-012", "invalid declaration name")
		return
	}
	for i := 0; i < len(p.document.Declarations); i++ {
		if p.document.Declarations[i].Kind == kind && p.document.Declarations[i].Name == name {
			p.failAt(nameStart, "RTG-PARSE-013", "duplicate "+kind+" declaration "+name)
			return
		}
	}
	open := p.at
	close, ok := p.matchBlock(open)
	if !ok {
		p.failAt(open, "RTG-PARSE-014", "unterminated "+kind+" declaration")
		return
	}
	declaration := Declaration{
		Kind:      kind,
		Name:      name,
		Start:     p.document.Tokens[startToken].Start,
		End:       p.document.Tokens[close].End,
		BodyStart: p.document.Tokens[open].End,
		BodyEnd:   p.document.Tokens[close].Start,
		Span:      sourceSpan(p.document.Source, p.document.Tokens[startToken].Start, p.document.Tokens[close].End),
	}
	declaration.Fields = p.parseFields(open+1, close)
	declaration.Statements = p.parseStatements(open+1, close)
	p.document.Declarations = append(p.document.Declarations, declaration)
	p.at = close + 1
}

func (p *documentParser) parseGo() {
	startToken := p.at
	p.at++
	if p.kind() != TokenIdent || p.text(p.at) != "backend" {
		p.failAt(p.at, "RTG-GO-001", "expected go backend block")
		return
	}
	p.at++
	if !p.operator(p.at, "{") {
		p.failAt(p.at, "RTG-GO-002", "expected { after go backend")
		return
	}
	open := p.at
	close, ok := p.matchBlock(open)
	if !ok {
		p.failAt(open, "RTG-GO-003", "unterminated go backend block")
		return
	}
	bodyStart := p.document.Tokens[open].End
	bodyEnd := p.document.Tokens[close].Start
	body := make([]byte, bodyEnd-bodyStart)
	copy(body, p.document.Source[bodyStart:bodyEnd])
	declaration := Declaration{
		Kind:      DeclGo,
		Name:      "backend",
		Start:     p.document.Tokens[startToken].Start,
		End:       p.document.Tokens[close].End,
		BodyStart: bodyStart,
		BodyEnd:   bodyEnd,
		GoSource:  body,
		Span:      sourceSpan(p.document.Source, p.document.Tokens[startToken].Start, p.document.Tokens[close].End),
	}
	p.validateGo(declaration)
	p.document.Declarations = append(p.document.Declarations, declaration)
	p.at = close + 1
}

func (p *documentParser) validateGo(declaration Declaration) {
	text := string(declaration.GoSource)
	if strings.Contains(text, "//go:") || strings.Contains(text, "//line ") ||
		strings.Contains(text, "/*line ") {
		p.failOffset(declaration.BodyStart, "RTG-GO-004", "embedded Go compiler and line directives are not allowed")
		return
	}
	wrapped := make([]byte, 0, len(declaration.GoSource)+17)
	wrapped = append(wrapped, "package backend\n"...)
	wrapped = append(wrapped, declaration.GoSource...)
	file := syntax.ParseFile(wrapped)
	if !file.Ok {
		offset := declaration.BodyStart
		if file.ErrorTok >= 0 && file.ErrorTok < len(file.Tokens) {
			offset += file.Tokens[file.ErrorTok].Start - len("package backend\n")
		}
		p.failOffset(offset, "RTG-GO-005", "embedded Go does not parse in the Renvo frontend")
		return
	}
	if len(file.Imports) != 0 {
		p.failOffset(declaration.BodyStart, "RTG-GO-006", "embedded Go imports are not allowed")
		return
	}
	for i := 0; i < len(file.Decls); i++ {
		p.validateGoName(wrapped, file.Tokens[file.Decls[i].NameTok], declaration.BodyStart)
	}
	for i := 0; i < len(file.Funcs); i++ {
		p.validateGoName(wrapped, file.Tokens[file.Funcs[i].NameTok], declaration.BodyStart)
	}
}

func (p *documentParser) validateGoName(wrapped []byte, token syntax.Token, bodyStart int) {
	name := string(syntax.TokenText(wrapped, token))
	offset := bodyStart + token.Start - len("package backend\n")
	if strings.HasPrefix(name, "RTG") {
		p.failOffset(offset, "RTG-GO-007", "embedded declaration uses reserved RTG prefix: "+name)
		return
	}
	for i := 0; i < len(p.goNames); i++ {
		if p.goNames[i] == name {
			p.failOffset(offset, "RTG-GO-008", "duplicate embedded Go declaration "+name)
			return
		}
	}
	p.goNames = append(p.goNames, name)
}

func (p *documentParser) parseFields(start int, end int) []Field {
	var fields []Field
	at := start
	for at < end {
		if p.document.Tokens[at].Kind != TokenIdent {
			at++
			continue
		}
		nameStart := at
		at++
		for at+1 < end && p.operator(at, ".") && p.document.Tokens[at+1].Kind == TokenIdent {
			at += 2
		}
		if at >= end || !p.operator(at, "=") {
			if at < end && p.operator(at, "{") {
				close, ok := p.matchBlock(at)
				if ok {
					at = close + 1
				}
			}
			continue
		}
		name := compactTokens(p.document.Source, p.document.Tokens[nameStart:at])
		valueStart := at + 1
		at = valueStart
		depth := 0
		for at < end {
			if p.operator(at, "{") || p.operator(at, "[") || p.operator(at, "(") {
				depth++
			} else if p.operator(at, "}") || p.operator(at, "]") || p.operator(at, ")") {
				if depth == 0 {
					break
				}
				depth--
			}
			if depth == 0 && p.document.Tokens[at].Kind == TokenIdent && at+1 < end && p.operator(at+1, "=") {
				break
			}
			at++
		}
		if valueStart == at {
			continue
		}
		fields = append(fields, Field{
			Name:       name,
			ValueStart: p.document.Tokens[valueStart].Start,
			ValueEnd:   p.document.Tokens[at-1].End,
			Span:       sourceSpan(p.document.Source, p.document.Tokens[nameStart].Start, p.document.Tokens[at-1].End),
		})
	}
	return fields
}

func (p *documentParser) parseStatements(start int, end int) []Statement {
	var statements []Statement
	at := start
	for at < end {
		statementStart := at
		line := p.document.Tokens[at].Line
		roundDepth := 0
		squareDepth := 0
		for at < end {
			if roundDepth == 0 && squareDepth == 0 && p.operator(at, "{") {
				close, ok := p.matchBlock(at)
				if !ok || close > end {
					return statements
				}
				statement := Statement{
					Tokens:   p.statementTokens(statementStart, at),
					Children: p.parseStatements(at+1, close),
					Span: sourceSpan(p.document.Source,
						p.document.Tokens[statementStart].Start,
						p.document.Tokens[close].End),
				}
				if len(statement.Tokens) != 0 {
					statements = append(statements, statement)
				}
				at = close + 1
				break
			}
			if p.operator(at, "(") {
				roundDepth++
			} else if p.operator(at, ")") && roundDepth > 0 {
				roundDepth--
			} else if p.operator(at, "[") {
				squareDepth++
			} else if p.operator(at, "]") && squareDepth > 0 {
				squareDepth--
			}
			at++
			if at == end || roundDepth == 0 && squareDepth == 0 &&
				p.document.Tokens[at].Line > line {
				statement := Statement{
					Tokens: p.statementTokens(statementStart, at),
					Span: sourceSpan(p.document.Source,
						p.document.Tokens[statementStart].Start,
						p.document.Tokens[at-1].End),
				}
				if len(statement.Tokens) != 0 {
					statements = append(statements, statement)
				}
				break
			}
		}
	}
	return statements
}

func (p *documentParser) statementTokens(start int, end int) []string {
	tokens := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		tokens = append(tokens, p.text(i))
	}
	return tokens
}

func (p *documentParser) matchBlock(open int) (int, bool) {
	if !p.operator(open, "{") {
		return open, false
	}
	depth := 0
	for i := open; i < len(p.document.Tokens); i++ {
		if p.operator(i, "{") {
			depth++
		} else if p.operator(i, "}") {
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return open, false
}

func (p *documentParser) kind() int {
	if p.at < 0 || p.at >= len(p.document.Tokens) {
		return TokenEOF
	}
	return p.document.Tokens[p.at].Kind
}

func (p *documentParser) text(at int) string {
	if at < 0 || at >= len(p.document.Tokens) {
		return ""
	}
	return tokenText(p.document.Source, p.document.Tokens[at])
}

func (p *documentParser) operator(at int, text string) bool {
	return at >= 0 && at < len(p.document.Tokens) &&
		p.document.Tokens[at].Kind == TokenOperator && p.text(at) == text
}

func (p *documentParser) failAt(at int, code string, message string) {
	if at < 0 {
		at = 0
	}
	if at >= len(p.document.Tokens) {
		at = len(p.document.Tokens) - 1
	}
	token := p.document.Tokens[at]
	p.document.Diagnostics = append(p.document.Diagnostics, Diagnostic{
		Filename: p.document.Filename,
		Span:     sourceSpan(p.document.Source, token.Start, token.End),
		Code:     code,
		Message:  message,
	})
}

func (p *documentParser) failOffset(offset int, code string, message string) {
	p.document.Diagnostics = append(p.document.Diagnostics, Diagnostic{
		Filename: p.document.Filename,
		Span:     sourceSpan(p.document.Source, offset, offset),
		Code:     code,
		Message:  message,
	})
}

func declarationName(source []byte, tokens []Token) string {
	if len(tokens) == 1 && tokens[0].Kind == TokenString {
		return unquoteSimple(tokenText(source, tokens[0]))
	}
	return compactTokens(source, tokens)
}

func compactTokens(source []byte, tokens []Token) string {
	var out []byte
	for i := 0; i < len(tokens); i++ {
		if tokens[i].Kind == TokenEOF {
			continue
		}
		out = append(out, source[tokens[i].Start:tokens[i].End]...)
	}
	return string(out)
}

func unquoteSimple(text string) string {
	if len(text) < 2 {
		return ""
	}
	quote := text[0]
	if text[len(text)-1] != quote || quote != '"' && quote != '\'' && quote != '`' {
		return ""
	}
	if quote == '`' {
		return text[1 : len(text)-1]
	}
	var out []byte
	for i := 1; i < len(text)-1; i++ {
		if text[i] != '\\' {
			out = append(out, text[i])
			continue
		}
		i++
		if i >= len(text)-1 {
			return ""
		}
		if text[i] == 'n' {
			out = append(out, '\n')
		} else if text[i] == 'r' {
			out = append(out, '\r')
		} else if text[i] == 't' {
			out = append(out, '\t')
		} else if text[i] == '\\' || text[i] == '"' || text[i] == '\'' {
			out = append(out, text[i])
		} else {
			return ""
		}
	}
	return string(out)
}

func parsePositiveDecimal(text string) (int, bool) {
	if text == "" {
		return 0, false
	}
	value := 0
	for i := 0; i < len(text); i++ {
		if text[i] < '0' || text[i] > '9' {
			return 0, false
		}
		digit := int(text[i] - '0')
		if value > (1073741824-digit)/10 {
			return 0, false
		}
		value = value*10 + digit
	}
	return value, true
}
