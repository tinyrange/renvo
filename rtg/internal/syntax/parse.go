package syntax

import "j5.nz/rtg/rtg/internal/arena"

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
	file := File{
		Src:         src,
		Tokens:      tokens,
		PackageName: -1,
		Ok:          true,
		Error:       ParseOK,
		ErrorTok:    -1,
	}
	// RTG represents these four integer fields in four eight-byte arena slots.
	// Scanning performs no other allocation, so the final backing array ends at
	// tokenArenaEnd even when append replaced one or more smaller arrays. Drop
	// those superseded arrays now so they do not contribute to peak RSS.
	const tokenArenaElementSize = 32
	finalTokenStart := tokenArenaEnd - cap(tokens)*tokenArenaElementSize
	if tokenArenaEnd > tokenArenaStart {
		arena.Discard(tokenArenaStart, finalTokenStart)
	}
	if !scanOK {
		return parseFail(file, ParseErrScan, len(tokens)-1)
	}
	return parseTokens(file)
}

func parseTokens(file File) File {
	if len(file.Tokens) < 3 || file.Tokens[0].Kind != TokenPackage || file.Tokens[1].Kind != TokenIdent {
		return parseFail(file, ParseErrPackage, 0)
	}
	file.PackageName = 1
	i := 2
	for i < len(file.Tokens) && file.Tokens[i].Kind != TokenEOF {
		i = skipTopSeparators(file, i)
		if i >= len(file.Tokens) || file.Tokens[i].Kind == TokenEOF {
			break
		}
		kind := file.Tokens[i].Kind
		if kind == TokenImport {
			next, ok := parseImportDecl(&file, i)
			if !ok {
				return parseFail(file, ParseErrImport, i)
			}
			i = next
			continue
		}
		if kind == TokenConst || kind == TokenVar || kind == TokenType {
			next, ok := parseTopDecl(&file, i)
			if !ok {
				return parseFail(file, ParseErrDecl, i)
			}
			i = next
			continue
		}
		if kind == TokenFunc {
			fn, ok := parseFuncDecl(file, i)
			if !ok {
				return parseFail(file, ParseErrFunc, i)
			}
			file.Funcs = append(file.Funcs, fn)
			i = fn.EndTok
			continue
		}
		return parseFail(file, ParseErrTopLevel, i)
	}
	return file
}

func parseFail(file File, err int, tok int) File {
	file.Ok = false
	file.Error = err
	file.ErrorTok = tok
	return file
}

func parseImportDecl(file *File, start int) (int, bool) {
	i := start + 1
	if tokCharIs(file.Src, file.Tokens, i, '(') {
		i++
		for i < len(file.Tokens) && file.Tokens[i].Kind != TokenEOF {
			i = skipImportSeparators(*file, i)
			if tokCharIs(file.Src, file.Tokens, i, ')') {
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
	if file.Tokens[pathTok].Kind != TokenString {
		if isImportName(file.Src, file.Tokens, pathTok) {
			nameTok = pathTok
			pathTok++
		}
	}
	if pathTok >= len(file.Tokens) || file.Tokens[pathTok].Kind != TokenString {
		return start, false
	}
	end := pathTok + 1
	next := end
	for next < len(file.Tokens) && file.Tokens[next].Kind != TokenEOF {
		if tokCharIs(file.Src, file.Tokens, next, ';') {
			next++
			break
		}
		if grouped && tokCharIs(file.Src, file.Tokens, next, ')') {
			break
		}
		if file.Tokens[next].Line != file.Tokens[pathTok].Line {
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
	kind := file.Tokens[start].Kind
	i := start + 1
	if tokCharIs(file.Src, file.Tokens, i, '(') {
		i++
		for i < len(file.Tokens) && file.Tokens[i].Kind != TokenEOF {
			i = skipDeclSeparators(*file, i)
			if tokCharIs(file.Src, file.Tokens, i, ')') {
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
	if start >= len(file.Tokens) || file.Tokens[start].Kind != TokenIdent {
		return start, false
	}
	end, next, ok := skipDeclSpec(*file, start, grouped)
	if !ok || end <= start {
		return start, false
	}
	if kind == TokenType {
		file.Decls = append(file.Decls, TopDecl{Kind: kind, NameTok: start, StartTok: start, EndTok: end})
		return next, true
	}
	file.Decls = append(file.Decls, TopDecl{Kind: kind, NameTok: start, StartTok: start, EndTok: end})
	i := start + 1
	for tokCharIs(file.Src, file.Tokens, i, ',') {
		i++
		if i >= len(file.Tokens) || file.Tokens[i].Kind != TokenIdent || i >= end {
			return start, false
		}
		file.Decls = append(file.Decls, TopDecl{Kind: kind, NameTok: i, StartTok: start, EndTok: end})
		i++
	}
	return next, true
}

func parseFuncDecl(file File, start int) (FuncDecl, bool) {
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
	if tokCharIs(file.Src, file.Tokens, i, '(') {
		receiverEnd := skipBalanced(file, i, '(', ')')
		if receiverEnd <= i {
			return fn, false
		}
		fn.ReceiverStart = i + 1
		fn.ReceiverEnd = receiverEnd - 1
		i = receiverEnd
	}
	if i >= len(file.Tokens) || file.Tokens[i].Kind != TokenIdent {
		return fn, false
	}
	fn.NameTok = i
	i++
	if !tokCharIs(file.Src, file.Tokens, i, '(') {
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
	bodyStart := findFuncBody(file, i)
	if bodyStart < 0 {
		return fn, false
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

func findFuncBody(file File, start int) int {
	i := start
	for i < len(file.Tokens) && file.Tokens[i].Kind != TokenEOF {
		if tokCharIs(file.Src, file.Tokens, i, '(') {
			next := skipBalanced(file, i, '(', ')')
			if next <= i {
				return -1
			}
			i = next
			continue
		}
		if tokCharIs(file.Src, file.Tokens, i, '[') {
			next := skipBalanced(file, i, '[', ']')
			if next <= i {
				return -1
			}
			i = next
			continue
		}
		if tokCharIs(file.Src, file.Tokens, i, '{') {
			if i > 0 && (file.Tokens[i-1].Kind == TokenStruct || file.Tokens[i-1].Kind == TokenInterface) {
				next := skipBalanced(file, i, '{', '}')
				if next <= i {
					return -1
				}
				i = next
				continue
			}
			return i
		}
		i++
	}
	return -1
}

func skipDeclSpec(file File, start int, grouped bool) (int, int, bool) {
	line := file.Tokens[start].Line
	i := start
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	for i < len(file.Tokens) && file.Tokens[i].Kind != TokenEOF {
		if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 {
			if grouped && tokCharIs(file.Src, file.Tokens, i, ')') {
				return i, i, true
			}
			if tokCharIs(file.Src, file.Tokens, i, ';') {
				return i, i + 1, true
			}
			if i > start && file.Tokens[i].Line != line {
				return i, i, true
			}
		}
		if tokCharIs(file.Src, file.Tokens, i, '(') {
			parenDepth++
		} else if tokCharIs(file.Src, file.Tokens, i, ')') {
			if parenDepth == 0 {
				if grouped && bracketDepth == 0 && braceDepth == 0 {
					return i, i, true
				}
				return start, start, false
			}
			parenDepth--
		} else if tokCharIs(file.Src, file.Tokens, i, '[') {
			bracketDepth++
		} else if tokCharIs(file.Src, file.Tokens, i, ']') {
			if bracketDepth == 0 {
				return start, start, false
			}
			bracketDepth--
		} else if tokCharIs(file.Src, file.Tokens, i, '{') {
			braceDepth++
		} else if tokCharIs(file.Src, file.Tokens, i, '}') {
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

func skipBalanced(file File, start int, open byte, close byte) int {
	if !tokCharIs(file.Src, file.Tokens, start, open) {
		return start
	}
	depth := 1
	i := start + 1
	for i < len(file.Tokens) && file.Tokens[i].Kind != TokenEOF {
		if tokCharIs(file.Src, file.Tokens, i, open) {
			depth++
		} else if tokCharIs(file.Src, file.Tokens, i, close) {
			depth--
			if depth == 0 {
				return i + 1
			}
		}
		i++
	}
	return start
}

func skipTopSeparators(file File, start int) int {
	for start < len(file.Tokens) && tokCharIs(file.Src, file.Tokens, start, ';') {
		start++
	}
	return start
}

func skipImportSeparators(file File, start int) int {
	for start < len(file.Tokens) && tokCharIs(file.Src, file.Tokens, start, ';') {
		start++
	}
	return start
}

func skipDeclSeparators(file File, start int) int {
	for start < len(file.Tokens) && tokCharIs(file.Src, file.Tokens, start, ';') {
		start++
	}
	return start
}

func isImportName(src []byte, toks []Token, i int) bool {
	if i < 0 || i >= len(toks) {
		return false
	}
	if toks[i].Kind == TokenIdent {
		return true
	}
	return tokCharIs(src, toks, i, '.')
}

func tokCharIs(src []byte, toks []Token, i int, c byte) bool {
	if i < 0 || i >= len(toks) {
		return false
	}
	if toks[i].Kind != TokenOperator {
		return false
	}
	if toks[i].End-toks[i].Start != 1 {
		return false
	}
	return src[toks[i].Start] == c
}
