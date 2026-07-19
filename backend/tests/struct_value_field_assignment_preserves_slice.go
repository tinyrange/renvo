package main

type renvoStructFieldToken struct {
	Kind int
}

type renvoStructFieldImportDecl struct {
	NameTok  int
	PathTok  int
	StartTok int
	EndTok   int
}

type renvoStructFieldTopDecl struct {
	Kind     int
	NameTok  int
	StartTok int
	EndTok   int
}

type renvoStructFieldFuncDecl struct {
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
	renvoStructFieldTokenEOF = iota
	renvoStructFieldTokenIdent
	renvoStructFieldTokenPackage
)

type renvoStructFieldFile struct {
	Src         []byte
	Tokens      []renvoStructFieldToken
	PackageName int
	Imports     []renvoStructFieldImportDecl
	Decls       []renvoStructFieldTopDecl
	Funcs       []renvoStructFieldFuncDecl
	Ok          bool
	Error       int
	ErrorTok    int
}

func renvoStructFieldParseFail(file renvoStructFieldFile, tok int) renvoStructFieldFile {
	file.Ok = false
	file.ErrorTok = tok
	return file
}

func renvoStructFieldParseTokens(file renvoStructFieldFile) renvoStructFieldFile {
	if len(file.Tokens) < 3 || file.Tokens[0].Kind != renvoStructFieldTokenPackage || file.Tokens[1].Kind != renvoStructFieldTokenIdent {
		return renvoStructFieldParseFail(file, 0)
	}
	file.PackageName = 1
	return file
}

func renvoStructFieldParse(src []byte) renvoStructFieldFile {
	tokens := make([]renvoStructFieldToken, 0)
	tokens = append(tokens, renvoStructFieldToken{Kind: renvoStructFieldTokenPackage})
	tokens = append(tokens, renvoStructFieldToken{Kind: renvoStructFieldTokenIdent})
	tokens = append(tokens, renvoStructFieldToken{Kind: renvoStructFieldTokenEOF})
	file := renvoStructFieldFile{
		Src:         src,
		Tokens:      tokens,
		PackageName: -1,
		Ok:          true,
		Error:       0,
		ErrorTok:    -1,
	}
	return renvoStructFieldParseTokens(file)
}

func appMain(args []string, env []string) int {
	src := []byte("package syntax\n\nconst")
	file := renvoStructFieldParse(src)
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
