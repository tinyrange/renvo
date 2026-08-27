package syntax

// ExportDirective returns the external identifier from a //export directive
// immediately preceding a package-level function. It intentionally implements
// the cgo spelling and adjacency rule without making comments part of the
// ordinary token stream.
func ExportDirective(file File, fn FuncDecl) string {
	if fn.ReceiverStart >= 0 || fn.StartTok < 0 || fn.StartTok >= len(file.Tokens) {
		return ""
	}
	start := TokenStart(file.Tokens[fn.StartTok])
	if start <= 0 || start > len(file.Src) {
		return ""
	}
	i := start
	for i > 0 && (file.Src[i-1] == ' ' || file.Src[i-1] == '\t' || file.Src[i-1] == '\r') {
		i--
	}
	if i == 0 || file.Src[i-1] != '\n' {
		return ""
	}
	end := i - 1
	if end > 0 && file.Src[end-1] == '\r' {
		end--
	}
	line := end
	for line > 0 && file.Src[line-1] != '\n' {
		line--
	}
	for line < end && (file.Src[line] == ' ' || file.Src[line] == '\t') {
		line++
	}
	const prefix = "//export "
	if end-line <= len(prefix) || !bytesEqualString(file.Src[line:line+len(prefix)], prefix) {
		return ""
	}
	name := file.Src[line+len(prefix) : end]
	if !exportIdentifier(name) {
		return ""
	}
	return string(name)
}

func exportIdentifier(name []byte) bool {
	if len(name) == 0 || !exportIdentifierStart(name[0]) {
		return false
	}
	for i := 1; i < len(name); i++ {
		if !exportIdentifierStart(name[i]) && (name[i] < '0' || name[i] > '9') {
			return false
		}
	}
	return true
}

func exportIdentifierStart(ch byte) bool {
	return ch == '_' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z'
}

func bytesEqualString(value []byte, text string) bool {
	if len(value) != len(text) {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] != text[i] {
			return false
		}
	}
	return true
}
