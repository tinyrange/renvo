// Package c11 translates a practical C11 source subset into the Go-shaped
// source model consumed by Renvo's existing checker, lowerer, and linker.
package c11

const (
	tokenEOF = iota
	tokenIdent
	tokenNumber
	tokenString
	tokenChar
	tokenPunct
)

const tokenLineShift = 8

type token struct {
	kindLine int
	start    int
	end      int
	match    int
}

func makeToken(kind int, start int, end int, line int) token {
	return token{kindLine: kind | line<<tokenLineShift, start: start, end: end}
}

func tokenKind(tok token) int { return tok.kindLine & 255 }
func tokenLine(tok token) int { return tok.kindLine >> tokenLineShift }

func tokenText(src []byte, tok token) []byte {
	if tok.start < 0 || tok.end < tok.start || tok.end > len(src) {
		return nil
	}
	return src[tok.start:tok.end]
}

func tokenIs(src []byte, tok token, text string) bool {
	value := tokenText(src, tok)
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
