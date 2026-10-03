package syntax

// GeneratedTypeNames reads the semantic type and embedded-field names saved by
// specialization. An adjacent reflection directive may appear between this
// compiler annotation and the declaration. Ordinary source names are unchanged
// when there is no annotation. Cache units retain the comment in their source.
func GeneratedTypeNames(src []byte, declarationStart int) (string, string) {
	begin, end := precedingDirectiveLine(src, declarationStart)
	if begin < 0 {
		return "", ""
	}
	if bytesEqualString(src[begin:end], "//renvo:reflect") {
		begin, end = precedingDirectiveLine(src, begin)
		if begin < 0 {
			return "", ""
		}
	}
	prefix := "//renvo:typename "
	if end-begin < len(prefix) || !bytesEqualString(src[begin:begin+len(prefix)], prefix) {
		return "", ""
	}
	pos := begin + len(prefix)
	typeName, next, ok := directiveString(src, pos, end)
	if !ok || next >= end || src[next] != ' ' {
		return "", ""
	}
	fieldName, next, ok := directiveString(src, next+1, end)
	if !ok || next != end || typeName == "" || fieldName == "" {
		return "", ""
	}
	return typeName, fieldName
}

func precedingDirectiveLine(src []byte, start int) (int, int) {
	if start <= 0 || start > len(src) {
		return -1, -1
	}
	for start > 0 && (src[start-1] == ' ' || src[start-1] == '\t' || src[start-1] == '\r') {
		start--
	}
	if start == 0 || src[start-1] != '\n' {
		return -1, -1
	}
	end := start - 1
	for end > 0 && (src[end-1] == ' ' || src[end-1] == '\t' || src[end-1] == '\r') {
		end--
	}
	begin := end
	for begin > 0 && src[begin-1] != '\n' {
		begin--
	}
	for begin < end && (src[begin] == ' ' || src[begin] == '\t') {
		begin++
	}
	return begin, end
}

func directiveString(src []byte, start int, end int) (string, int, bool) {
	if start >= end || src[start] != '"' {
		return "", start, false
	}
	pos := start + 1
	for pos < end {
		if src[pos] == '"' {
			pos++
			value, ok := StringLiteralValue(src, Token{KindLine: TokenString, Start: int32(start), End: int32(pos)})
			return value, pos, ok
		}
		if src[pos] == '\\' {
			pos++
		}
		pos++
	}
	return "", start, false
}
