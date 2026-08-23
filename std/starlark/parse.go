package starlark

import (
	"strconv"
	"strings"
)

type tokenKind int

const (
	tEOF tokenKind = iota
	tNewline
	tIndent
	tDedent
	tName
	tInt
	tFloat
	tString
	tLParen
	tRParen
	tLBrack
	tRBrack
	tLBrace
	tRBrace
	tComma
	tColon
	tDot
	tSemi
	tPlus
	tMinus
	tStar
	tSlash
	tSlashSlash
	tPercent
	tPow
	tPipe
	tAmp
	tCaret
	tTilde
	tAssign
	tEq
	tNe
	tLT
	tLE
	tGT
	tGE
	tPlusEq
	tMinusEq
	tStarEq
	tSlashEq
	tPercentEq
)

type token struct {
	kind      tokenKind
	text      string
	line, col int
}

func lex(src string) ([]token, error) {
	var out []token
	indents := []int{0}
	pos, line := 0, 1
	lineStart := true
	nesting := 0
	emit := func(k tokenKind, s string, c int) { out = append(out, token{k, s, line, c}) }
	for pos < len(src) {
		if lineStart && nesting == 0 {
			start := pos
			width := 0
			for pos < len(src) && (src[pos] == ' ' || src[pos] == '\t') {
				if src[pos] == '\t' {
					width = (width/8 + 1) * 8
				} else {
					width++
				}
				pos++
			}
			if pos >= len(src) {
				break
			}
			if src[pos] == '\n' || src[pos] == '\r' || src[pos] == '#' {
				pos = start
			} else {
				if width > indents[len(indents)-1] {
					indents = append(indents, width)
					emit(tIndent, "", 1)
				} else if width < indents[len(indents)-1] {
					for len(indents) > 1 && width < indents[len(indents)-1] {
						indents = indents[:len(indents)-1]
						emit(tDedent, "", 1)
					}
					if width != indents[len(indents)-1] {
						return nil, errorf("line %d: inconsistent indentation", line)
					}
				}
				lineStart = false
			}
		}
		if pos >= len(src) {
			break
		}
		c := src[pos]
		col := pos
		if c == ' ' || c == '\t' || c == '\r' {
			pos++
			continue
		}
		if c == '#' {
			for pos < len(src) && src[pos] != '\n' {
				pos++
			}
			continue
		}
		if c == '\n' {
			pos++
			if nesting == 0 {
				if len(out) == 0 || out[len(out)-1].kind != tNewline {
					emit(tNewline, "", col)
				}
			}
			line++
			lineStart = true
			continue
		}
		rawString := (c == 'r' || c == 'R') && pos+1 < len(src) && (src[pos+1] == '\'' || src[pos+1] == '"')
		if isIdentStart(c) && !rawString {
			start := pos
			pos++
			for pos < len(src) && isIdentPart(src[pos]) {
				pos++
			}
			emit(tName, src[start:pos], col)
			continue
		}
		if c >= '0' && c <= '9' {
			start := pos
			kind := tInt
			pos++
			for pos < len(src) && ((src[pos] >= '0' && src[pos] <= '9') || src[pos] == '_') {
				pos++
			}
			if pos < len(src) && src[pos] == '.' {
				kind = tFloat
				pos++
				for pos < len(src) && src[pos] >= '0' && src[pos] <= '9' {
					pos++
				}
			}
			emit(kind, src[start:pos], col)
			continue
		}
		if c == '\'' || c == '"' || rawString {
			if rawString {
				pos++
				c = src[pos]
			}
			quote := c
			startLine := line
			triple := pos+2 < len(src) && src[pos+1] == quote && src[pos+2] == quote
			if triple {
				pos += 3
			} else {
				pos++
			}
			var b stringBuilder
			for {
				if pos >= len(src) {
					return nil, errorf("line %d: unterminated string", startLine)
				}
				if triple && pos+2 < len(src) && src[pos] == quote && src[pos+1] == quote && src[pos+2] == quote {
					pos += 3
					break
				}
				if !triple && src[pos] == quote {
					pos++
					break
				}
				if src[pos] == '\n' {
					if !triple {
						return nil, errorf("line %d: unterminated string", startLine)
					}
					b.WriteByte('\n')
					pos++
					line++
					continue
				}
				if rawString && src[pos] == '\\' && pos+1 < len(src) {
					if src[pos+1] == quote {
						b.WriteByte(quote)
						pos += 2
						continue
					}
					if src[pos+1] == '\n' {
						b.WriteByte('\\')
						b.WriteByte('\n')
						pos += 2
						line++
						continue
					}
				}
				if src[pos] == '\\' && !rawString {
					if pos+1 >= len(src) {
						return nil, errorf("line %d: bad escape", line)
					}
					esc := src[pos+1]
					switch esc {
					case 'n':
						b.WriteByte('\n')
					case 'r':
						b.WriteByte('\r')
					case 't':
						b.WriteByte('\t')
					case '\\', '\'', '"':
						b.WriteByte(esc)
					default:
						b.WriteByte(esc)
					}
					pos += 2
					continue
				}
				b.WriteByte(src[pos])
				pos++
			}
			emit(tString, b.String(), col)
			continue
		}
		two := ""
		if pos+1 < len(src) {
			two = src[pos : pos+2]
		}
		twoKinds := map[string]tokenKind{"==": tEq, "!=": tNe, "<=": tLE, ">=": tGE, "//": tSlashSlash, "**": tPow, "+=": tPlusEq, "-=": tMinusEq, "*=": tStarEq, "/=": tSlashEq, "%=": tPercentEq}
		if k, ok := twoKinds[two]; ok {
			emit(k, two, col)
			pos += 2
			continue
		}
		ones := map[byte]tokenKind{'(': tLParen, ')': tRParen, '[': tLBrack, ']': tRBrack, '{': tLBrace, '}': tRBrace, ',': tComma, ':': tColon, '.': tDot, ';': tSemi, '+': tPlus, '-': tMinus, '*': tStar, '/': tSlash, '%': tPercent, '|': tPipe, '&': tAmp, '^': tCaret, '~': tTilde, '=': tAssign, '<': tLT, '>': tGT}
		k, ok := ones[c]
		if !ok {
			character := byteString(c)
			return nil, errorf("line %d: unexpected character %q", line, character)
		}
		emit(k, byteString(c), col)
		pos++
		if k == tLParen || k == tLBrack || k == tLBrace {
			nesting++
		}
		if k == tRParen || k == tRBrack || k == tRBrace {
			nesting--
		}
	}
	if len(out) == 0 || out[len(out)-1].kind != tNewline {
		emit(tNewline, "", 0)
	}
	for len(indents) > 1 {
		indents = indents[:len(indents)-1]
		emit(tDedent, "", 0)
	}
	emit(tEOF, "", 0)
	return out, nil
}
func isIdentStart(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}
func isIdentPart(c byte) bool { return isIdentStart(c) || (c >= '0' && c <= '9') }

func byteString(c byte) string { return string([]byte{c}) }

type stmt interface{ stmtNode() }
type expr interface{ exprNode() }
type exprStmt struct{ x expr }

func (exprStmt) stmtNode() {}

type assignStmt struct {
	target expr
	op     tokenKind
	value  expr
}

func (assignStmt) stmtNode() {}

type defStmt struct {
	name   string
	params []paramExpr
	body   []stmt
	doc    string
}

func (defStmt) stmtNode() {}

type ifStmt struct {
	cond      expr
	then      []stmt
	otherwise []stmt
}

func (ifStmt) stmtNode() {}

type forStmt struct {
	target expr
	iter   expr
	body   []stmt
}

func (forStmt) stmtNode() {}

type whileStmt struct {
	cond expr
	body []stmt
}

func (whileStmt) stmtNode() {}

type returnStmt struct{ x expr }

func (returnStmt) stmtNode() {}

type flowStmt struct{ kind string }

func (flowStmt) stmtNode() {}

type paramExpr struct {
	name       string
	def        expr
	hasDefault bool
}
type literalExpr struct{ v Value }

func (literalExpr) exprNode() {}

type nameExpr struct{ name string }

func (nameExpr) exprNode() {}

type unaryExpr struct {
	op string
	x  expr
}

func (unaryExpr) exprNode() {}

type binaryExpr struct {
	op          string
	left, right expr
}

func (binaryExpr) exprNode() {}

type callExpr struct {
	fn     expr
	args   []expr
	kwargs []kwExpr
}

func (callExpr) exprNode() {}

type kwExpr struct {
	name string
	x    expr
}
type attrExpr struct {
	x    expr
	name string
}

func (attrExpr) exprNode() {}

type indexExpr struct {
	x, index expr
	end      expr
	isSlice  bool
}

func (indexExpr) exprNode() {}

type listExpr struct{ items []expr }

func (listExpr) exprNode() {}

type tupleExpr struct{ items []expr }

func (tupleExpr) exprNode() {}

type dictExpr struct{ keys, values []expr }

func (dictExpr) exprNode() {}

type comprehensionExpr struct {
	item, target, iter expr
	conds              []expr
}

func (comprehensionExpr) exprNode() {}

type dictComprehensionExpr struct {
	key, value, target, iter expr
	conds                    []expr
}

func (dictComprehensionExpr) exprNode() {}

type conditionalExpr struct{ yes, cond, no expr }

func (conditionalExpr) exprNode() {}

type parser struct {
	t        []token
	i        int
	filename string
}

func parse(filename string, src []byte) ([]stmt, error) {
	ts, err := lex(string(src))
	if err != nil {
		return nil, err
	}
	p := parser{t: ts, filename: filename}
	return p.file()
}
func (p *parser) cur() token          { return p.t[p.i] }
func (p *parser) at(k tokenKind) bool { return p.cur().kind == k }
func (p *parser) next() token {
	x := p.cur()
	if p.i < len(p.t)-1 {
		p.i++
	}
	return x
}
func (p *parser) accept(k tokenKind) bool {
	if p.at(k) {
		p.i++
		return true
	}
	return false
}
func (p *parser) err(msg string) error { return errorf("%s:%d: %s", p.filename, p.cur().line, msg) }
func (p *parser) need(k tokenKind, msg string) error {
	if !p.accept(k) {
		return p.err(msg)
	}
	return nil
}
func (p *parser) file() ([]stmt, error) {
	var out []stmt
	for !p.at(tEOF) {
		if p.accept(tNewline) {
			continue
		}
		s, err := p.statement()
		if err != nil {
			return nil, err
		}
		out = append(out, s...)
	}
	return out, nil
}
func (p *parser) statement() ([]stmt, error) {
	if p.at(tName) {
		switch p.cur().text {
		case "def":
			s, e := p.parseDef()
			return one(s), e
		case "if":
			s, e := p.parseIf()
			return one(s), e
		case "for":
			s, e := p.parseFor()
			return one(s), e
		case "while":
			s, e := p.parseWhile()
			return one(s), e
		}
	}
	var out []stmt
	for {
		s, err := p.simple()
		if err != nil {
			return nil, err
		}
		out = append(out, s)
		if !p.accept(tSemi) {
			break
		}
	}
	if err := p.need(tNewline, "expected end of line"); err != nil {
		return nil, err
	}
	return out, nil
}
func one(s stmt) []stmt {
	if s == nil {
		return nil
	}
	return []stmt{s}
}
func (p *parser) simple() (stmt, error) {
	if p.at(tName) {
		word := p.cur().text
		switch word {
		case "return":
			p.next()
			if p.at(tNewline) || p.at(tSemi) {
				return returnStmt{}, nil
			}
			x, e := p.expressionList()
			return returnStmt{x}, e
		case "pass", "break", "continue":
			p.next()
			return flowStmt{word}, nil
		}
	}
	x, err := p.expressionList()
	if err != nil {
		return nil, err
	}
	op := p.cur().kind
	if op == tAssign || op == tPlusEq || op == tMinusEq || op == tStarEq || op == tSlashEq || op == tPercentEq {
		p.next()
		v, e := p.expressionList()
		return assignStmt{x, op, v}, e
	}
	return exprStmt{x}, nil
}

func (p *parser) expressionList() (expr, error) {
	first, err := p.expression()
	if err != nil || !p.accept(tComma) {
		return first, err
	}
	items := []expr{first}
	for !p.at(tNewline) && !p.at(tSemi) && !p.at(tAssign) && !p.at(tPlusEq) && !p.at(tMinusEq) && !p.at(tStarEq) && !p.at(tSlashEq) && !p.at(tPercentEq) {
		item, err := p.expression()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
		if !p.accept(tComma) {
			break
		}
	}
	return tupleExpr{items}, nil
}
func (p *parser) suite() ([]stmt, error) {
	if err := p.need(tColon, "expected ':'"); err != nil {
		return nil, err
	}
	if !p.accept(tNewline) {
		var out []stmt
		for {
			s, e := p.simple()
			if e != nil {
				return nil, e
			}
			out = append(out, s)
			if !p.accept(tSemi) {
				break
			}
		}
		return out, p.need(tNewline, "expected end of line")
	}
	if err := p.need(tIndent, "expected indented block"); err != nil {
		return nil, err
	}
	var out []stmt
	for !p.at(tDedent) && !p.at(tEOF) {
		if p.accept(tNewline) {
			continue
		}
		ss, e := p.statement()
		if e != nil {
			return nil, e
		}
		out = append(out, ss...)
	}
	if err := p.need(tDedent, "expected dedent"); err != nil {
		return nil, err
	}
	return out, nil
}
func (p *parser) parseDef() (stmt, error) {
	p.next()
	if !p.at(tName) {
		return nil, p.err("expected function name")
	}
	name := p.next().text
	if e := p.need(tLParen, "expected '('"); e != nil {
		return nil, e
	}
	var params []paramExpr
	seenDefault := false
	seenNames := map[string]bool{}
	if !p.at(tRParen) {
		for {
			if !p.at(tName) {
				return nil, p.err("expected parameter")
			}
			q := paramExpr{name: p.next().text}
			if seenNames[q.name] {
				return nil, p.err("duplicate parameter " + q.name)
			}
			seenNames[q.name] = true
			if p.accept(tAssign) {
				q.def, q.hasDefault = mustExpr(p)
				if q.def == nil {
					return nil, p.err("expected default value")
				}
				seenDefault = true
			} else if seenDefault {
				return nil, p.err("required parameter follows default parameter")
			}
			params = append(params, q)
			if !p.accept(tComma) {
				break
			}
			if p.at(tRParen) {
				break
			}
		}
	}
	if e := p.need(tRParen, "expected ')'"); e != nil {
		return nil, e
	}
	body, e := p.suite()
	if e != nil {
		return nil, e
	}
	doc := ""
	if len(body) > 0 {
		if es, ok := body[0].(exprStmt); ok {
			if lit, ok := es.x.(literalExpr); ok {
				if s, ok := AsString(lit.v); ok {
					doc = s
					body = body[1:]
				}
			}
		}
	}
	return defStmt{name, params, body, doc}, nil
}
func mustExpr(p *parser) (expr, bool) { x, e := p.expression(); return x, e == nil }
func (p *parser) parseIf() (stmt, error) {
	p.next()
	cond, e := p.expression()
	if e != nil {
		return nil, e
	}
	then, e := p.suite()
	if e != nil {
		return nil, e
	}
	var other []stmt
	if p.at(tName) && p.cur().text == "elif" {
		p.t[p.i].text = "if"
		s, e := p.parseIf()
		if e != nil {
			return nil, e
		}
		other = []stmt{s}
	} else if p.at(tName) && p.cur().text == "else" {
		p.next()
		other, e = p.suite()
		if e != nil {
			return nil, e
		}
	}
	return ifStmt{cond, then, other}, nil
}
func (p *parser) parseFor() (stmt, error) {
	p.next()
	if !p.at(tName) {
		return nil, p.err("expected loop variable")
	}
	var targets []expr
	for {
		targets = append(targets, nameExpr{p.next().text})
		if !p.accept(tComma) {
			break
		}
		if !p.at(tName) {
			return nil, p.err("expected loop variable")
		}
	}
	var target expr = targets[0]
	if len(targets) > 1 {
		target = tupleExpr{targets}
	}
	if !p.at(tName) || p.next().text != "in" {
		return nil, p.err("expected 'in'")
	}
	it, e := p.expression()
	if e != nil {
		return nil, e
	}
	body, e := p.suite()
	return forStmt{target, it, body}, e
}
func (p *parser) parseWhile() (stmt, error) {
	p.next()
	cond, e := p.expression()
	if e != nil {
		return nil, e
	}
	body, e := p.suite()
	return whileStmt{cond, body}, e
}

func (p *parser) expression() (expr, error) {
	x, e := p.binary(0)
	if e != nil {
		return nil, e
	}
	if p.at(tName) && p.cur().text == "if" {
		p.next()
		c, e := p.binary(0)
		if e != nil {
			return nil, e
		}
		if !p.at(tName) || p.next().text != "else" {
			return nil, p.err("expected else")
		}
		n, e := p.expression()
		return conditionalExpr{x, c, n}, e
	}
	return x, nil
}
func precedence(t token) (int, string, bool) {
	if t.kind == tName {
		switch t.text {
		case "or":
			return 1, "or", true
		case "and":
			return 2, "and", true
		case "in":
			return 3, "in", true
		}
	}
	switch t.kind {
	case tEq:
		return 3, "==", true
	case tNe:
		return 3, "!=", true
	case tLT:
		return 3, "<", true
	case tLE:
		return 3, "<=", true
	case tGT:
		return 3, ">", true
	case tGE:
		return 3, ">=", true
	case tPipe:
		return 4, "|", true
	case tCaret:
		return 5, "^", true
	case tAmp:
		return 6, "&", true
	case tPlus:
		return 7, "+", true
	case tMinus:
		return 7, "-", true
	case tStar:
		return 8, "*", true
	case tSlash:
		return 8, "/", true
	case tSlashSlash:
		return 8, "//", true
	case tPercent:
		return 8, "%", true
	}
	return 0, "", false
}
func (p *parser) binary(min int) (expr, error) {
	left, e := p.unary()
	if e != nil {
		return nil, e
	}
	for {
		if p.at(tName) && p.cur().text == "not" && p.i+1 < len(p.t) && p.t[p.i+1].kind == tName && p.t[p.i+1].text == "in" {
			if 3 < min {
				break
			}
			p.i += 2
			right, e := p.binary(4)
			if e != nil {
				return nil, e
			}
			left = unaryExpr{"not", binaryExpr{"in", left, right}}
			continue
		}
		prec, op, ok := precedence(p.cur())
		if !ok || prec < min {
			break
		}
		p.next()
		right, e := p.binary(prec + 1)
		if e != nil {
			return nil, e
		}
		left = binaryExpr{op, left, right}
	}
	return left, nil
}
func (p *parser) unary() (expr, error) {
	if p.at(tName) && p.cur().text == "not" {
		op := p.next().text
		x, e := p.binary(3)
		return unaryExpr{op, x}, e
	}
	if p.at(tPlus) || p.at(tMinus) || p.at(tTilde) {
		op := p.next().text
		x, e := p.unary()
		return unaryExpr{op, x}, e
	}
	return p.power()
}
func (p *parser) power() (expr, error) {
	left, err := p.postfix()
	if err != nil || !p.accept(tPow) {
		return left, err
	}
	right, err := p.unary()
	if err != nil {
		return nil, err
	}
	return binaryExpr{"**", left, right}, nil
}
func (p *parser) postfix() (expr, error) {
	x, e := p.primary()
	if e != nil {
		return nil, e
	}
	for {
		switch {
		case p.accept(tDot):
			if !p.at(tName) {
				return nil, p.err("expected attribute")
			}
			x = attrExpr{x, p.next().text}
		case p.accept(tLParen):
			c := callExpr{fn: x}
			seenKeyword := false
			if !p.at(tRParen) {
				for {
					if p.at(tName) && p.t[p.i+1].kind == tAssign {
						seenKeyword = true
						name := p.next().text
						p.next()
						v, e := p.expression()
						if e != nil {
							return nil, e
						}
						c.kwargs = append(c.kwargs, kwExpr{name, v})
					} else {
						if seenKeyword {
							return nil, p.err("positional argument follows keyword argument")
						}
						v, e := p.expression()
						if e != nil {
							return nil, e
						}
						c.args = append(c.args, v)
					}
					if !p.accept(tComma) {
						break
					}
					if p.at(tRParen) {
						break
					}
				}
			}
			if e := p.need(tRParen, "expected ')'"); e != nil {
				return nil, e
			}
			x = c
		case p.accept(tLBrack):
			var first, end expr
			isSlice := false
			if !p.at(tColon) && !p.at(tRBrack) {
				first, e = p.expression()
				if e != nil {
					return nil, e
				}
			}
			if p.accept(tColon) {
				isSlice = true
				if !p.at(tRBrack) {
					end, e = p.expression()
					if e != nil {
						return nil, e
					}
				}
			}
			if e := p.need(tRBrack, "expected ']'"); e != nil {
				return nil, e
			}
			x = indexExpr{x, first, end, isSlice}
		default:
			return x, nil
		}
	}
}
func (p *parser) primary() (expr, error) {
	tok := p.next()
	switch tok.kind {
	case tInt:
		n, e := strconv.ParseInt(strings.ReplaceAll(tok.text, "_", ""), 10, 64)
		return literalExpr{Int(n)}, e
	case tFloat:
		n, e := parseFloat(tok.text)
		return literalExpr{Float(n)}, e
	case tString:
		return literalExpr{String(tok.text)}, nil
	case tName:
		switch tok.text {
		case "True":
			return literalExpr{True}, nil
		case "False":
			return literalExpr{False}, nil
		case "None":
			return literalExpr{None}, nil
		}
		return nameExpr{tok.text}, nil
	case tLParen:
		if p.accept(tRParen) {
			return tupleExpr{}, nil
		}
		first, e := p.expression()
		if e != nil {
			return nil, e
		}
		if !p.accept(tComma) {
			if e := p.need(tRParen, "expected ')'"); e != nil {
				return nil, e
			}
			return first, nil
		}
		items := []expr{first}
		for !p.at(tRParen) {
			v, e := p.expression()
			if e != nil {
				return nil, e
			}
			items = append(items, v)
			if !p.accept(tComma) {
				break
			}
		}
		if e := p.need(tRParen, "expected ')'"); e != nil {
			return nil, e
		}
		return tupleExpr{items}, nil
	case tLBrack:
		var items []expr
		for !p.at(tRBrack) {
			v, e := p.expression()
			if e != nil {
				return nil, e
			}
			if len(items) == 0 && p.at(tName) && p.cur().text == "for" {
				p.next()
				target, e := p.comprehensionTarget()
				if e != nil {
					return nil, e
				}
				if !p.at(tName) || p.next().text != "in" {
					return nil, p.err("expected 'in' in comprehension")
				}
				iter, e := p.binary(0)
				if e != nil {
					return nil, e
				}
				var conds []expr
				for p.at(tName) && p.cur().text == "if" {
					p.next()
					c, e := p.binary(0)
					if e != nil {
						return nil, e
					}
					conds = append(conds, c)
				}
				if e := p.need(tRBrack, "expected ']'"); e != nil {
					return nil, e
				}
				return comprehensionExpr{v, target, iter, conds}, nil
			}
			items = append(items, v)
			if !p.accept(tComma) {
				break
			}
		}
		if e := p.need(tRBrack, "expected ']'"); e != nil {
			return nil, e
		}
		return listExpr{items}, nil
	case tLBrace:
		d := dictExpr{}
		for !p.at(tRBrace) {
			k, e := p.expression()
			if e != nil {
				return nil, e
			}
			if e := p.need(tColon, "expected ':' in dict"); e != nil {
				return nil, e
			}
			v, e := p.expression()
			if e != nil {
				return nil, e
			}
			if len(d.keys) == 0 && p.at(tName) && p.cur().text == "for" {
				p.next()
				target, e := p.comprehensionTarget()
				if e != nil {
					return nil, e
				}
				if !p.at(tName) || p.next().text != "in" {
					return nil, p.err("expected 'in' in comprehension")
				}
				iter, e := p.binary(0)
				if e != nil {
					return nil, e
				}
				var conds []expr
				for p.at(tName) && p.cur().text == "if" {
					p.next()
					condition, e := p.binary(0)
					if e != nil {
						return nil, e
					}
					conds = append(conds, condition)
				}
				if e := p.need(tRBrace, "expected '}'"); e != nil {
					return nil, e
				}
				return dictComprehensionExpr{k, v, target, iter, conds}, nil
			}
			d.keys = append(d.keys, k)
			d.values = append(d.values, v)
			if !p.accept(tComma) {
				break
			}
		}
		if e := p.need(tRBrace, "expected '}'"); e != nil {
			return nil, e
		}
		return d, nil
	}
	return nil, p.err("expected expression")
}

func (p *parser) comprehensionTarget() (expr, error) {
	if !p.at(tName) {
		return nil, p.err("expected comprehension variable")
	}
	var items []expr
	for {
		items = append(items, nameExpr{p.next().text})
		if !p.accept(tComma) {
			break
		}
		if !p.at(tName) {
			return nil, p.err("expected comprehension variable")
		}
	}
	if len(items) == 1 {
		return items[0], nil
	}
	return tupleExpr{items}, nil
}
