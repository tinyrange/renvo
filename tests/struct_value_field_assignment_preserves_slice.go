package main

type rtgStructFieldToken struct {
	Kind int
}

type rtgStructFieldImportDecl struct {
	NameTok  int
	PathTok  int
	StartTok int
	EndTok   int
}

type rtgStructFieldTopDecl struct {
	Kind     int
	NameTok  int
	StartTok int
	EndTok   int
}

type rtgStructFieldFuncDecl struct {
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

const (
	rtgStructFieldTokenEOF = iota
	rtgStructFieldTokenIdent
	rtgStructFieldTokenPackage
)

type rtgStructFieldFile struct {
	Src         []byte
	Tokens      []rtgStructFieldToken
	PackageName int
	Imports     []rtgStructFieldImportDecl
	Decls       []rtgStructFieldTopDecl
	Funcs       []rtgStructFieldFuncDecl
	Ok          bool
	Error       int
	ErrorTok    int
}

func rtgStructFieldParseFail(file rtgStructFieldFile, tok int) rtgStructFieldFile {
	file.Ok = false
	file.ErrorTok = tok
	return file
}

func rtgStructFieldParseTokens(file rtgStructFieldFile) rtgStructFieldFile {
	if len(file.Tokens) < 3 || file.Tokens[0].Kind != rtgStructFieldTokenPackage || file.Tokens[1].Kind != rtgStructFieldTokenIdent {
		return rtgStructFieldParseFail(file, 0)
	}
	file.PackageName = 1
	return file
}

func rtgStructFieldParse(src []byte) rtgStructFieldFile {
	tokens := make([]rtgStructFieldToken, 0)
	tokens = append(tokens, rtgStructFieldToken{Kind: rtgStructFieldTokenPackage})
	tokens = append(tokens, rtgStructFieldToken{Kind: rtgStructFieldTokenIdent})
	tokens = append(tokens, rtgStructFieldToken{Kind: rtgStructFieldTokenEOF})
	file := rtgStructFieldFile{
		Src:         src,
		Tokens:      tokens,
		PackageName: -1,
		Ok:          true,
		Error:       0,
		ErrorTok:    -1,
	}
	return rtgStructFieldParseTokens(file)
}

func appMain(args []string, env []string) int {
	src := []byte("package syntax\n\nconst")
	file := rtgStructFieldParse(src)
	if file.PackageName != 1 {
		return 1
	}
	if len(src) != 21 {
		return 1
	}
	if src[0] != 'p' || src[1] != 'a' || src[2] != 'c' || src[3] != 'k' {
		return 1
	}
	if src[8] != 's' || src[14] != '\n' || src[15] != '\n' || src[16] != 'c' {
		return 1
	}
	print("PASS\n")
	return 0
}
