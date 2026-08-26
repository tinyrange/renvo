package syntax

import (
	"unsafe"

	"renvo.dev/internal/arena"
)

const (
	ParseOK = iota
	ParseErrScan
	ParseErrPackage
	ParseErrImport
	ParseErrDecl
	ParseErrFunc
	ParseErrTopLevel
)

type File struct {
	Src         []byte
	Tokens      []Token
	PackageName int
	Imports     []ImportDecl
	Decls       []TopDecl
	Funcs       []FuncDecl
	Ok          bool
	Error       int
	ErrorTok    int
}

type ImportDecl struct {
	NameTok  int
	PathTok  int
	StartTok int
	EndTok   int
}

type TopDecl struct {
	Kind     int
	NameTok  int
	StartTok int
	EndTok   int
}

type FuncDecl struct {
	NameTok       int
	StartTok      int
	EndTok        int
	ReceiverStart int
	ReceiverEnd   int
	ParamsStart   int
	ParamsEnd     int
	ResultStart   int
	ResultEnd     int
	BodyStart     int
	BodyEnd       int
}

func ParseFile(src []byte) File {
	tokenArenaStart := arena.Mark()
	tokens, scanOK := parseScanTokens(src)
	tokenArenaEnd := arena.Mark()
	tokenCapacity := cap(tokens)
	file := File{
		Src:         src,
		Tokens:      tokens,
		PackageName: -1,
		Ok:          true,
		Error:       ParseOK,
		ErrorTok:    -1,
	}
	// Scanning performs no other allocation, so the final backing array ends at
	// tokenArenaEnd even when append replaced one or more smaller arrays. Drop
	// those superseded arrays now so they do not contribute to peak RSS.
	tokenArenaElementSize := int(unsafe.Sizeof(Token{}))
	finalTokenStart := tokenArenaEnd - tokenCapacity*tokenArenaElementSize
	if tokenArenaEnd > tokenArenaStart {
		arena.Discard(tokenArenaStart, finalTokenStart)
		unusedTokenStart := finalTokenStart + len(tokens)*tokenArenaElementSize
		arena.Discard(unusedTokenStart, tokenArenaEnd)
		// Scanning is the only allocation between the marks. Move the arena
		// cursor back over the unused tail so subsequent persistent allocations
		// reuse even its partial final page.
		arena.Rewind(unusedTokenStart)
	}
	// The discarded tail is no longer part of the live token allocation.
	// Restrict capacity so a caller cannot append into pages that were released.
	tokens = tokens[:len(tokens):len(tokens)]
	file.Tokens = tokens
	if !scanOK {
		return parseFail(file, ParseErrScan, len(tokens)-1)
	}
	parseTokens(&file)
	return file
}

func parseTokens(file *File) {
	if len(file.Tokens) < 3 || file.Tokens[0].KindLine&255 != TokenPackage || file.Tokens[1].KindLine&255 != TokenIdent {
		parseFailInPlace(file, ParseErrPackage, 0)
		return
	}
	file.PackageName = 1
	i := 2
	for i < len(file.Tokens) && file.Tokens[i].KindLine&255 != TokenEOF {
		i = skipTopSeparators(file, i)
		if i >= len(file.Tokens) || file.Tokens[i].KindLine&255 == TokenEOF {
			break
		}
		kind := file.Tokens[i].KindLine & 255
		if kind == TokenImport {
			next, ok := parseImportDecl(file, i)
			if !ok {
				parseFailInPlace(file, ParseErrImport, i)
				return
			}
			i = next
			continue
		}
		if kind == TokenConst || kind == TokenVar || kind == TokenType {
			next, ok := parseTopDecl(file, i)
			if !ok {
				parseFailInPlace(file, ParseErrDecl, i)
				return
			}
			i = next
			continue
		}
		if kind == TokenFunc {
			fn, ok := parseFuncDecl(file, i)
			if !ok {
				parseFailInPlace(file, ParseErrFunc, i)
				return
			}
			file.Funcs = append(file.Funcs, fn)
			i = fn.EndTok
			continue
		}
		parseFailInPlace(file, ParseErrTopLevel, i)
		return
	}
}

func parseFailInPlace(file *File, err int, tok int) {
	file.Ok = false
	file.Error = err
	file.ErrorTok = tok
}

func parseFail(file File, err int, tok int) File {
	file.Ok = false
	file.Error = err
	file.ErrorTok = tok
	return file
}

func parseImportDecl(file *File, start int) (int, bool) {
	i := start + 1
	if tokCharIs(file.Tokens, i, '(') {
		i++
		for i < len(file.Tokens) && file.Tokens[i].KindLine&255 != TokenEOF {
			i = skipImportSeparators(file, i)
			if tokCharIs(file.Tokens, i, ')') {
				return i + 1, true
			}
			next, ok := parseImportSpec(file, i, true)
			if !ok {
				return start, false
			}
			i = next
		}
		return start, false
	}
	next, ok := parseImportSpec(file, i, false)
	if !ok {
		return start, false
	}
	return next, true
}

func parseImportSpec(file *File, start int, grouped bool) (int, bool) {
	nameTok := -1
	pathTok := start
	if pathTok >= len(file.Tokens) {
		return start, false
	}
	if file.Tokens[pathTok].KindLine&255 != TokenString {
		if isImportName(file.Tokens, pathTok) {
			nameTok = pathTok
			pathTok++
		}
	}
	if pathTok >= len(file.Tokens) || file.Tokens[pathTok].KindLine&255 != TokenString {
		return start, false
	}
	end := pathTok + 1
	next := end
	for next < len(file.Tokens) && file.Tokens[next].KindLine&255 != TokenEOF {
		if tokCharIs(file.Tokens, next, ';') {
			next++
			break
		}
		if grouped && tokCharIs(file.Tokens, next, ')') {
			break
		}
		if TokenLine(file.Tokens[next]) != TokenLine(file.Tokens[pathTok]) {
			break
		}
		return start, false
	}
	file.Imports = append(file.Imports, ImportDecl{
		NameTok:  nameTok,
		PathTok:  pathTok,
		StartTok: start,
		EndTok:   end,
	})
	return next, true
}

func parseTopDecl(file *File, start int) (int, bool) {
	kind := file.Tokens[start].KindLine & 255
	i := start + 1
	if tokCharIs(file.Tokens, i, '(') {
		i++
		for i < len(file.Tokens) && file.Tokens[i].KindLine&255 != TokenEOF {
			i = skipDeclSeparators(file, i)
			if tokCharIs(file.Tokens, i, ')') {
				return i + 1, true
			}
			next, ok := parseDeclSpec(file, kind, i, true)
			if !ok {
				return start, false
			}
			i = next
		}
		return start, false
	}
	next, ok := parseDeclSpec(file, kind, i, false)
	if !ok {
		return start, false
	}
	return next, true
}

func parseDeclSpec(file *File, kind int, start int, grouped bool) (int, bool) {
	if start >= len(file.Tokens) || file.Tokens[start].KindLine&255 != TokenIdent {
		return start, false
	}
	end, next, ok := skipDeclSpec(file, start, grouped)
	if !ok || end <= start {
		return start, false
	}
	if kind == TokenType {
		file.Decls = append(file.Decls, TopDecl{Kind: kind, NameTok: start, StartTok: start, EndTok: end})
		return next, true
	}
	file.Decls = append(file.Decls, TopDecl{Kind: kind, NameTok: start, StartTok: start, EndTok: end})
	i := start + 1
	for tokCharIs(file.Tokens, i, ',') {
		i++
		if i >= len(file.Tokens) || file.Tokens[i].KindLine&255 != TokenIdent || i >= end {
			return start, false
		}
		file.Decls = append(file.Decls, TopDecl{Kind: kind, NameTok: i, StartTok: start, EndTok: end})
		i++
	}
	return next, true
}

func parseFuncDecl(file *File, start int) (FuncDecl, bool) {
	fn := FuncDecl{
		StartTok:      start,
		EndTok:        start,
		NameTok:       -1,
		ReceiverStart: -1,
		ReceiverEnd:   -1,
		ParamsStart:   -1,
		ParamsEnd:     -1,
		ResultStart:   -1,
		ResultEnd:     -1,
		BodyStart:     -1,
		BodyEnd:       -1,
	}
	i := start + 1
	if tokCharIs(file.Tokens, i, '(') {
		receiverEnd := skipBalanced(file, i, '(', ')')
		if receiverEnd <= i {
			return fn, false
		}
		fn.ReceiverStart = i + 1
		fn.ReceiverEnd = receiverEnd - 1
		i = receiverEnd
	}
	if i >= len(file.Tokens) || file.Tokens[i].KindLine&255 != TokenIdent {
		return fn, false
	}
	fn.NameTok = i
	i++
	if !tokCharIs(file.Tokens, i, '(') {
		return fn, false
	}
	fn.ParamsStart = i
	paramsEnd := skipBalanced(file, i, '(', ')')
	if paramsEnd <= i {
		return fn, false
	}
	fn.ParamsEnd = paramsEnd
	i = paramsEnd
	fn.ResultStart = i
	bodyStart, declarationEnd := findFuncBody(file, i)
	if bodyStart < 0 {
		// Bodyless declarations are retained for project assembly binding.  The
		// checker/lowerer reject them later unless a selected .rtgasm entry owns
		// the implementation.
		fn.ResultEnd = declarationEnd
		fn.EndTok = declarationEnd
		return fn, declarationEnd >= i
	}
	fn.ResultEnd = bodyStart
	bodyEnd := skipBalanced(file, bodyStart, '{', '}')
	if bodyEnd <= bodyStart {
		return fn, false
	}
	fn.BodyStart = bodyStart
	fn.BodyEnd = bodyEnd
	fn.EndTok = bodyEnd
	return fn, true
}

func findFuncBody(file *File, start int) (int, int) {
	i := start
	for i < len(file.Tokens) && file.Tokens[i].KindLine&255 != TokenEOF {
		if tokCharIs(file.Tokens, i, ';') {
			return -1, i + 1
		}
		kind := file.Tokens[i].KindLine & 255
		previousLine := TokenLine(file.Tokens[start-1])
		if i > start {
			previousLine = TokenLine(file.Tokens[i-1])
		}
		functionDecl := kind == TokenFunc && i+1 < len(file.Tokens) &&
			file.Tokens[i+1].KindLine&255 == TokenIdent
		if TokenLine(file.Tokens[i]) > previousLine &&
			(functionDecl || kind == TokenConst || kind == TokenVar ||
				kind == TokenType || kind == TokenImport) {
			return -1, i
		}
		if tokCharIs(file.Tokens, i, '(') {
			next := skipBalanced(file, i, '(', ')')
			if next <= i {
				return -1, -1
			}
			i = next
			continue
		}
		if tokCharIs(file.Tokens, i, '[') {
			next := skipBalanced(file, i, '[', ']')
			if next <= i {
				return -1, -1
			}
			i = next
			continue
		}
		if tokCharIs(file.Tokens, i, '{') {
			if i > 0 && (file.Tokens[i-1].KindLine&255 == TokenStruct || file.Tokens[i-1].KindLine&255 == TokenInterface) {
				next := skipBalanced(file, i, '{', '}')
				if next <= i {
					return -1, -1
				}
				i = next
				continue
			}
			return i, i
		}
		i++
	}
	if i < len(file.Tokens) && file.Tokens[i].KindLine&255 == TokenEOF {
		return -1, i
	}
	return -1, -1
}

func skipDeclSpec(file *File, start int, grouped bool) (int, int, bool) {
	line := TokenLine(file.Tokens[start])
	i := start
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	for i < len(file.Tokens) && file.Tokens[i].KindLine&255 != TokenEOF {
		c := byte(file.Tokens[i].KindLine >> TokenOperatorCharShift & TokenOperatorCharMask)
		if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 {
			if grouped && c == ')' {
				return i, i, true
			}
			if c == ';' {
				return i, i + 1, true
			}
			if i > start && TokenLine(file.Tokens[i]) != line {
				return i, i, true
			}
		}
		if c == '(' {
			parenDepth++
		} else if c == ')' {
			if parenDepth == 0 {
				if grouped && bracketDepth == 0 && braceDepth == 0 {
					return i, i, true
				}
				return start, start, false
			}
			parenDepth--
		} else if c == '[' {
			bracketDepth++
		} else if c == ']' {
			if bracketDepth == 0 {
				return start, start, false
			}
			bracketDepth--
		} else if c == '{' {
			braceDepth++
		} else if c == '}' {
			if braceDepth == 0 {
				return start, start, false
			}
			braceDepth--
		}
		i++
	}
	if parenDepth != 0 || bracketDepth != 0 || braceDepth != 0 {
		return start, start, false
	}
	return i, i, true
}

func skipBalanced(file *File, start int, open byte, close byte) int {
	if !tokCharIs(file.Tokens, start, open) {
		return start
	}
	depth := 1
	i := start + 1
	for i < len(file.Tokens) {
		tok := file.Tokens[i]
		if tok.KindLine&255 == TokenEOF {
			break
		}
		c := byte(tok.KindLine >> TokenOperatorCharShift & TokenOperatorCharMask)
		if c == open {
			depth++
		} else if c == close {
			depth--
			if depth == 0 {
				return i + 1
			}
		}
		i++
	}
	return start
}

func skipTopSeparators(file *File, start int) int {
	for start < len(file.Tokens) && tokCharIs(file.Tokens, start, ';') {
		start++
	}
	return start
}

func skipImportSeparators(file *File, start int) int {
	for start < len(file.Tokens) && tokCharIs(file.Tokens, start, ';') {
		start++
	}
	return start
}

func skipDeclSeparators(file *File, start int) int {
	for start < len(file.Tokens) && tokCharIs(file.Tokens, start, ';') {
		start++
	}
	return start
}

func isImportName(toks []Token, i int) bool {
	if i < 0 || i >= len(toks) {
		return false
	}
	if toks[i].KindLine&255 == TokenIdent {
		return true
	}
	return tokCharIs(toks, i, '.')
}

func tokCharIs(toks []Token, i int, c byte) bool {
	if uint(i) >= uint(len(toks)) {
		return false
	}
	packed := toks[i].KindLine
	return packed>>TokenOperatorCharShift&TokenOperatorCharMask == int(c)
}
