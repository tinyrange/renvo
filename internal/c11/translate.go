package c11

const (
	TranslateOK = iota
	TranslateErrScan
	TranslateErrDeclaration
	TranslateErrStatement
	TranslateErrUnsupported
)

const (
	storageNone = iota
	storageOther
	storageExtern
	storageTypedef
)

// Result is the C11 source adapter result. Source is ordinary Renvo Go input,
// so every later frontend phase is shared with Go packages.
type Result struct {
	Source  []byte
	Ok      bool
	Error   int
	ErrorAt int
}

type cType struct {
	name    string
	pointer int
	arrays  [][]token
}

type declarator struct {
	name        token
	typeInfo    cType
	params      []parameter
	function    bool
	initializer []token
}

type parameter struct {
	name     token
	typeInfo cType
}

type translator struct {
	src         []byte
	tokens      []token
	pos         int
	packageName string
	out         []byte
	ok          bool
	err         int
	errorAt     int
}

// Translate lowers one preprocessed C translation unit into the shared Go
// frontend syntax. It deliberately performs no backend-specific lowering.
func Translate(packageName string, src []byte) Result {
	scanned := scan(src)
	if !scanned.ok {
		return Result{Ok: false, Error: TranslateErrScan, ErrorAt: scanned.errorAt}
	}
	t := translator{
		src: src, tokens: scanned.tokens, packageName: packageName,
		out: make([]byte, 0, len(src)+len(src)/4+32), ok: true, errorAt: -1,
	}
	t.out = append(t.out, "package "...)
	t.out = append(t.out, packageName...)
	t.out = append(t.out, '\n')
	for t.ok && t.kind() != tokenEOF {
		t.skipDirectives()
		if t.kind() == tokenEOF {
			break
		}
		t.externalDeclaration()
	}
	if !t.ok {
		return Result{Ok: false, Error: t.err, ErrorAt: t.errorAt}
	}
	return Result{Source: t.out, Ok: true, Error: TranslateOK, ErrorAt: -1}
}

func (t *translator) kind() int {
	if t.pos < 0 || t.pos >= len(t.tokens) {
		return tokenEOF
	}
	return tokenKind(t.tokens[t.pos])
}

func (t *translator) currentIs(text string) bool {
	return t.pos >= 0 && t.pos < len(t.tokens) && tokenIs(t.src, t.tokens[t.pos], text)
}

func (t *translator) take(text string) bool {
	if !t.currentIs(text) {
		return false
	}
	t.pos++
	return true
}

func (t *translator) fail(err int) {
	if !t.ok {
		return
	}
	t.ok = false
	t.err = err
	if t.pos >= 0 && t.pos < len(t.tokens) {
		t.errorAt = t.tokens[t.pos].start
	} else {
		t.errorAt = len(t.src)
	}
}

func (t *translator) skipDirectives() {
	for t.kind() != tokenEOF && t.currentIs("#") && (t.pos == 0 || tokenLine(t.tokens[t.pos-1]) < tokenLine(t.tokens[t.pos])) {
		line := tokenLine(t.tokens[t.pos])
		if t.pos+1 >= len(t.tokens) ||
			!tokenIs(t.src, t.tokens[t.pos+1], "include") && !tokenIs(t.src, t.tokens[t.pos+1], "pragma") {
			t.fail(TranslateErrUnsupported)
			return
		}
		for t.kind() != tokenEOF && tokenLine(t.tokens[t.pos]) == line {
			t.pos++
		}
	}
}

func (t *translator) externalDeclaration() {
	base, storage, ok := t.parseType()
	if !ok {
		t.fail(TranslateErrDeclaration)
		return
	}
	decl, ok := t.parseDeclarator(base, true)
	if !ok {
		t.fail(TranslateErrDeclaration)
		return
	}
	if decl.function {
		if t.take(";") {
			return // A prototype is satisfied by a C or Go definition in the package.
		}
		if !t.currentIs("{") {
			t.fail(TranslateErrDeclaration)
			return
		}
		t.emitFunction(decl)
		return
	}
	decl.initializer = t.takeInitializer()
	decls := []declarator{decl}
	for t.take(",") {
		next, valid := t.parseDeclarator(base, false)
		if !valid {
			t.fail(TranslateErrDeclaration)
			return
		}
		next.initializer = t.takeInitializer()
		decls = append(decls, next)
	}
	if !t.take(";") {
		t.fail(TranslateErrDeclaration)
		return
	}
	if storage == storageTypedef {
		t.fail(TranslateErrUnsupported)
		return
	}
	if storage == storageExtern {
		return
	}
	for i := 0; i < len(decls); i++ {
		t.emitVariable(decls[i])
	}
}

func (t *translator) parseType() (cType, int, bool) {
	storage := storageNone
	unsigned := false
	longCount := 0
	base := ""
	seen := false
	for t.kind() == tokenIdent {
		switch {
		case t.currentIs("static") || t.currentIs("register") || t.currentIs("auto") || t.currentIs("inline"):
			if storage == storageNone {
				storage = storageOther
			}
			t.pos++
		case t.currentIs("extern"):
			storage = storageExtern
			t.pos++
		case t.currentIs("typedef"):
			storage = storageTypedef
			t.pos++
		case t.currentIs("const") || t.currentIs("volatile") || t.currentIs("restrict") || t.currentIs("_Atomic"):
			t.pos++
		case t.currentIs("unsigned"):
			unsigned, seen = true, true
			t.pos++
		case t.currentIs("signed"):
			seen = true
			t.pos++
		case t.currentIs("long"):
			longCount++
			seen = true
			t.pos++
		case t.currentIs("short"):
			base, seen = "short", true
			t.pos++
		case t.currentIs("char"):
			base, seen = "char", true
			t.pos++
		case t.currentIs("int"):
			if base == "" {
				base = "int"
			}
			seen = true
			t.pos++
		case t.currentIs("void"):
			base, seen = "void", true
			t.pos++
		case t.currentIs("float"):
			base, seen = "float", true
			t.pos++
		case t.currentIs("double"):
			base, seen = "double", true
			t.pos++
		case t.currentIs("_Bool"):
			base, seen = "bool", true
			t.pos++
		default:
			if !seen {
				return cType{}, storage, false
			}
			goto done
		}
	}
done:
	if !seen {
		return cType{}, storage, false
	}
	if base == "" {
		base = "int"
	}
	name := "int"
	switch base {
	case "void":
		name = ""
	case "bool":
		name = "bool"
	case "char":
		if unsigned {
			name = "uint8"
		} else {
			name = "int8"
		}
	case "short":
		if unsigned {
			name = "uint16"
		} else {
			name = "int16"
		}
	case "float":
		name = "float32"
	case "double":
		name = "float64"
	default:
		if longCount > 0 {
			if unsigned {
				name = "uint64"
			} else {
				name = "int64"
			}
		} else if unsigned {
			name = "uint"
		}
	}
	return cType{name: name}, storage, true
}

func (t *translator) parseDeclarator(base cType, allowFunction bool) (declarator, bool) {
	result := declarator{typeInfo: base}
	for t.take("*") {
		result.typeInfo.pointer++
		for t.currentIs("const") || t.currentIs("volatile") || t.currentIs("restrict") {
			t.pos++
		}
	}
	if t.kind() != tokenIdent {
		return result, false
	}
	result.name = t.tokens[t.pos]
	t.pos++
	if allowFunction && t.take("(") {
		result.function = true
		if t.take(")") {
			return result, true
		}
		if t.currentIs("void") && t.pos+1 < len(t.tokens) && tokenIs(t.src, t.tokens[t.pos+1], ")") {
			t.pos += 2
			return result, true
		}
		for {
			paramType, _, ok := t.parseType()
			if !ok {
				return result, false
			}
			param, ok := t.parseDeclarator(paramType, false)
			if !ok || len(param.typeInfo.arrays) != 0 {
				return result, false
			}
			result.params = append(result.params, parameter{name: param.name, typeInfo: param.typeInfo})
			if t.take(")") {
				break
			}
			if !t.take(",") || t.currentIs("...") {
				return result, false
			}
		}
		return result, true
	}
	for t.take("[") {
		start := t.pos
		end := t.findClosing("]")
		if end <= start {
			return result, false
		}
		result.typeInfo.arrays = append(result.typeInfo.arrays, t.tokens[start:end])
		t.pos = end + 1
	}
	return result, true
}

func (t *translator) takeInitializer() []token {
	if !t.take("=") {
		return nil
	}
	start := t.pos
	paren, bracket, brace := 0, 0, 0
	for t.kind() != tokenEOF {
		if paren == 0 && bracket == 0 && brace == 0 && (t.currentIs(",") || t.currentIs(";")) {
			break
		}
		switch {
		case t.currentIs("("):
			paren++
		case t.currentIs(")"):
			paren--
		case t.currentIs("["):
			bracket++
		case t.currentIs("]"):
			bracket--
		case t.currentIs("{"):
			brace++
		case t.currentIs("}"):
			brace--
		}
		t.pos++
	}
	return t.tokens[start:t.pos]
}

func (t *translator) emitFunction(decl declarator) {
	if tokenIs(t.src, decl.name, "main") && t.packageName == "main" &&
		(len(decl.params) != 0 || decl.typeInfo.name != "int" || decl.typeInfo.pointer != 0 || len(decl.typeInfo.arrays) != 0) {
		t.fail(TranslateErrUnsupported)
		return
	}
	t.out = append(t.out, "func "...)
	name := tokenText(t.src, decl.name)
	if tokenIs(t.src, decl.name, "main") && t.packageName == "main" {
		name = []byte("appMain")
	}
	t.out = append(t.out, name...)
	t.out = append(t.out, '(')
	for i := 0; i < len(decl.params); i++ {
		if i > 0 {
			t.out = append(t.out, ',')
		}
		t.out = append(t.out, tokenText(t.src, decl.params[i].name)...)
		t.out = append(t.out, ' ')
		t.emitType(decl.params[i].typeInfo)
	}
	t.out = append(t.out, ')')
	if decl.typeInfo.name != "" || decl.typeInfo.pointer > 0 {
		t.out = append(t.out, ' ')
		t.emitType(decl.typeInfo)
	}
	t.out = append(t.out, ' ')
	t.block()
	t.out = append(t.out, '\n')
}

func (t *translator) emitVariable(decl declarator) {
	t.out = append(t.out, "var "...)
	t.out = append(t.out, tokenText(t.src, decl.name)...)
	t.out = append(t.out, ' ')
	t.emitType(decl.typeInfo)
	if len(decl.initializer) > 0 {
		t.out = append(t.out, '=')
		if len(decl.typeInfo.arrays) > 0 && tokenIs(t.src, decl.initializer[0], "{") {
			t.emitType(decl.typeInfo)
		}
		t.expression(decl.initializer)
	}
	t.out = append(t.out, ';', '\n')
}

func (t *translator) emitType(info cType) {
	for i := 0; i < len(info.arrays); i++ {
		t.out = append(t.out, '[')
		t.expression(info.arrays[i])
		t.out = append(t.out, ']')
	}
	for i := 0; i < info.pointer; i++ {
		t.out = append(t.out, '*')
	}
	if info.name == "" {
		t.out = append(t.out, "byte"...)
	} else {
		t.out = append(t.out, info.name...)
	}
}

func (t *translator) block() {
	if !t.take("{") {
		t.fail(TranslateErrStatement)
		return
	}
	t.out = append(t.out, '{')
	for t.ok && t.kind() != tokenEOF && !t.currentIs("}") {
		t.skipDirectives()
		if t.currentIs("}") {
			break
		}
		t.statement()
	}
	if !t.take("}") {
		t.fail(TranslateErrStatement)
		return
	}
	t.out = append(t.out, '}')
}

func (t *translator) statement() {
	if t.currentIs("{") {
		t.block()
		return
	}
	if t.isTypeStart() {
		t.localDeclaration()
		return
	}
	if t.take("if") {
		t.out = append(t.out, "if "...)
		t.parenthesizedCondition()
		t.out = append(t.out, ' ')
		t.statement()
		if t.take("else") {
			t.out = append(t.out, " else "...)
			t.statement()
		}
		return
	}
	if t.take("while") {
		t.out = append(t.out, "for "...)
		t.parenthesizedCondition()
		t.out = append(t.out, ' ')
		t.statement()
		return
	}
	if t.take("for") {
		t.forStatement()
		return
	}
	if t.currentIs("switch") || t.currentIs("case") || t.currentIs("default") ||
		t.currentIs("do") || t.currentIs("goto") || t.currentIs("asm") ||
		t.currentIs("__asm") || t.currentIs("__asm__") {
		t.fail(TranslateErrUnsupported)
		return
	}
	if t.take("return") {
		t.out = append(t.out, "return"...)
		start := t.pos
		end := t.findAtDepth(";")
		if end < 0 {
			t.fail(TranslateErrStatement)
			return
		}
		if end > start {
			t.out = append(t.out, ' ')
			t.expression(t.tokens[start:end])
		}
		t.out = append(t.out, ';')
		t.pos = end + 1
		return
	}
	if t.currentIs("break") || t.currentIs("continue") {
		start := t.pos
		end := t.findAtDepth(";")
		if end < 0 {
			t.fail(TranslateErrStatement)
			return
		}
		t.expression(t.tokens[start:end])
		t.out = append(t.out, ';')
		t.pos = end + 1
		return
	}
	if t.take(";") {
		t.out = append(t.out, ';')
		return
	}
	if t.kind() == tokenIdent && t.pos+1 < len(t.tokens) && tokenIs(t.src, t.tokens[t.pos+1], ":") {
		t.fail(TranslateErrUnsupported)
		return
	}
	start := t.pos
	end := t.findAtDepth(";")
	if end < 0 {
		t.fail(TranslateErrStatement)
		return
	}
	t.expression(t.tokens[start:end])
	t.out = append(t.out, ';')
	t.pos = end + 1
}

func (t *translator) localDeclaration() {
	base, storage, ok := t.parseType()
	if !ok || storage == storageTypedef || storage == storageExtern {
		t.fail(TranslateErrUnsupported)
		return
	}
	for {
		decl, valid := t.parseDeclarator(base, false)
		if !valid {
			t.fail(TranslateErrDeclaration)
			return
		}
		decl.initializer = t.takeInitializer()
		t.emitVariable(decl)
		if !t.take(",") {
			break
		}
	}
	if !t.take(";") {
		t.fail(TranslateErrDeclaration)
	}
}

func (t *translator) forStatement() {
	if !t.take("(") {
		t.fail(TranslateErrStatement)
		return
	}
	close := t.findClosing(")")
	if close < 0 {
		t.fail(TranslateErrStatement)
		return
	}
	parts := splitTopLevel(t.src, t.tokens[t.pos:close], ";")
	if len(parts) != 3 || len(parts[0]) > 0 && isTypeToken(t.src, parts[0][0]) {
		t.fail(TranslateErrUnsupported)
		return
	}
	t.out = append(t.out, "for "...)
	t.expression(parts[0])
	t.out = append(t.out, ';')
	t.condition(parts[1])
	t.out = append(t.out, ';')
	t.expression(parts[2])
	t.pos = close + 1
	t.out = append(t.out, ' ')
	t.statement()
}

func (t *translator) parenthesizedCondition() {
	if !t.take("(") {
		t.fail(TranslateErrStatement)
		return
	}
	end := t.findClosing(")")
	if end < 0 {
		t.fail(TranslateErrStatement)
		return
	}
	t.condition(t.tokens[t.pos:end])
	t.pos = end + 1
}

func (t *translator) condition(tokens []token) {
	boolean := false
	for i := 0; i < len(tokens); i++ {
		if tokenIs(t.src, tokens[i], "==") || tokenIs(t.src, tokens[i], "!=") ||
			tokenIs(t.src, tokens[i], "<") || tokenIs(t.src, tokens[i], ">") ||
			tokenIs(t.src, tokens[i], "<=") || tokenIs(t.src, tokens[i], ">=") ||
			tokenIs(t.src, tokens[i], "&&") || tokenIs(t.src, tokens[i], "||") ||
			tokenIs(t.src, tokens[i], "true") || tokenIs(t.src, tokens[i], "false") {
			boolean = true
			break
		}
	}
	if boolean {
		t.expression(tokens)
		return
	}
	t.out = append(t.out, '(')
	t.expression(tokens)
	t.out = append(t.out, ")!=0"...)
}

func (t *translator) expression(tokens []token) {
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		text := tokenText(t.src, tok)
		if tokenKind(tok) == tokenNumber {
			text = trimIntegerSuffix(text)
		} else if tokenIs(t.src, tok, "NULL") {
			text = []byte("nil")
		} else if tokenIs(t.src, tok, "->") {
			text = []byte(".")
		} else if tokenIs(t.src, tok, "~") {
			text = []byte("^")
		} else if (tokenKind(tok) == tokenString || tokenKind(tok) == tokenChar) && len(text) > 1 && text[0] != '"' && text[0] != '\'' {
			for len(text) > 0 && text[0] != '"' && text[0] != '\'' {
				text = text[1:]
			}
		}
		t.out = append(t.out, text...)
		if i+1 < len(tokens) && expressionNeedsSpace(text, tokenText(t.src, tokens[i+1])) {
			t.out = append(t.out, ' ')
		}
	}
}

func expressionNeedsSpace(left []byte, right []byte) bool {
	if len(left) == 0 || len(right) == 0 {
		return false
	}
	a, b := left[len(left)-1], right[0]
	return isIdentPart(a) && isIdentPart(b)
}

func trimIntegerSuffix(text []byte) []byte {
	end := len(text)
	for end > 0 {
		c := text[end-1]
		if c != 'u' && c != 'U' && c != 'l' && c != 'L' {
			break
		}
		end--
	}
	return text[:end]
}

func (t *translator) findClosing(close string) int {
	depth := 0
	open := "("
	if close == "]" {
		open = "["
	}
	for i := t.pos; i < len(t.tokens); i++ {
		if tokenIs(t.src, t.tokens[i], open) {
			depth++
		} else if tokenIs(t.src, t.tokens[i], close) {
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

func (t *translator) findAtDepth(text string) int {
	paren, bracket, brace := 0, 0, 0
	for i := t.pos; i < len(t.tokens); i++ {
		tok := t.tokens[i]
		if paren == 0 && bracket == 0 && brace == 0 && tokenIs(t.src, tok, text) {
			return i
		}
		switch {
		case tokenIs(t.src, tok, "("):
			paren++
		case tokenIs(t.src, tok, ")"):
			paren--
		case tokenIs(t.src, tok, "["):
			bracket++
		case tokenIs(t.src, tok, "]"):
			bracket--
		case tokenIs(t.src, tok, "{"):
			brace++
		case tokenIs(t.src, tok, "}"):
			if brace == 0 {
				return -1
			}
			brace--
		}
	}
	return -1
}

func (t *translator) isTypeStart() bool {
	return t.kind() == tokenIdent && isTypeToken(t.src, t.tokens[t.pos])
}

func isTypeToken(src []byte, tok token) bool {
	return tokenIs(src, tok, "auto") || tokenIs(src, tok, "register") || tokenIs(src, tok, "static") ||
		tokenIs(src, tok, "extern") || tokenIs(src, tok, "typedef") || tokenIs(src, tok, "const") ||
		tokenIs(src, tok, "volatile") || tokenIs(src, tok, "restrict") || tokenIs(src, tok, "inline") ||
		tokenIs(src, tok, "void") || tokenIs(src, tok, "char") || tokenIs(src, tok, "short") ||
		tokenIs(src, tok, "int") || tokenIs(src, tok, "long") || tokenIs(src, tok, "float") ||
		tokenIs(src, tok, "double") || tokenIs(src, tok, "signed") || tokenIs(src, tok, "unsigned") ||
		tokenIs(src, tok, "_Bool") || tokenIs(src, tok, "_Atomic")
}

func splitTopLevel(src []byte, tokens []token, separator string) [][]token {
	var out [][]token
	start := 0
	paren, bracket, brace := 0, 0, 0
	for i := 0; i < len(tokens); i++ {
		if paren == 0 && bracket == 0 && brace == 0 && tokenIs(src, tokens[i], separator) {
			out = append(out, tokens[start:i])
			start = i + 1
			continue
		}
		switch {
		case tokenIs(src, tokens[i], "("):
			paren++
		case tokenIs(src, tokens[i], ")"):
			paren--
		case tokenIs(src, tokens[i], "["):
			bracket++
		case tokenIs(src, tokens[i], "]"):
			bracket--
		case tokenIs(src, tokens[i], "{"):
			brace++
		case tokenIs(src, tokens[i], "}"):
			brace--
		}
	}
	out = append(out, tokens[start:])
	return out
}
