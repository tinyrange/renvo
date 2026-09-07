package syntax

// ReflectDirective reports an explicit //renvo:reflect line immediately above
// a type declaration (or an individual specification in a grouped declaration).
// It does not opt in sibling or transitively referenced types.
func ReflectDirective(file File, decl TopDecl) bool {
	if decl.Kind != TokenType || decl.StartTok < 0 || decl.StartTok >= len(file.Tokens) {
		return false
	}
	tok := decl.StartTok
	if tok > 0 && file.Tokens[tok-1].KindLine&255 == TokenType {
		tok--
	}
	start := TokenStart(file.Tokens[tok])
	if start <= 0 || start > len(file.Src) {
		return false
	}
	for start > 0 && (file.Src[start-1] == ' ' || file.Src[start-1] == '\t' || file.Src[start-1] == '\r') {
		start--
	}
	if start == 0 || file.Src[start-1] != '\n' {
		return false
	}
	end := start - 1
	for end > 0 && (file.Src[end-1] == ' ' || file.Src[end-1] == '\t' || file.Src[end-1] == '\r') {
		end--
	}
	begin := end
	for begin > 0 && file.Src[begin-1] != '\n' {
		begin--
	}
	for begin < end && (file.Src[begin] == ' ' || file.Src[begin] == '\t') {
		begin++
	}
	return bytesEqualString(file.Src[begin:end], "//renvo:reflect")
}
