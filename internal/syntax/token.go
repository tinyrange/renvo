package syntax

const (
	TokenEOF = iota
	TokenIdent
	TokenNumber
	TokenString
	TokenChar
	TokenOperator
	TokenPackage
	TokenImport
	TokenConst
	TokenVar
	TokenType
	TokenFunc
	TokenStruct
	TokenInterface
	TokenMap
	TokenReturn
	TokenIf
	TokenElse
	TokenFor
	TokenRange
	TokenSwitch
	TokenCase
	TokenDefault
	TokenBreak
	TokenContinue
	TokenGoto
	TokenDefer
	TokenGo
	TokenSelect
	TokenChan
	TokenFallthrough
)

const (
	TokenOperatorCharShift = 8
	TokenOperatorCharMask  = 127
	TokenOperatorLineShift = 15
	TokenLineLimit         = 65535
)

type Token struct {
	// KindLine packs the kind into the low byte, a one-byte ASCII operator in
	// bits 8..14 when present, and the source line above both. Source offsets
	// are explicitly 32-bit so tokens stay compact without depending on the
	// host's int width.
	KindLine int
	Start    int32
	End      int32
}

func MakeToken(kind int, start int, end int, line int) Token {
	return Token{KindLine: kind | line<<TokenOperatorLineShift, Start: int32(start), End: int32(end)}
}

func TokenLine(tok Token) int {
	return tok.KindLine >> TokenOperatorLineShift & TokenLineLimit
}

func TokenSize(tok Token) int {
	return int(tok.End - tok.Start)
}

func TokenEnd(tok Token) int {
	return int(tok.End)
}

func TokenStart(tok Token) int {
	return int(tok.Start)
}

func TokenText(src []byte, tok Token) []byte {
	start := int(tok.Start)
	end := int(tok.End)
	if start < 0 || end < start || end > len(src) {
		return nil
	}
	return src[start:end]
}

func NumberTokenIsFloat(src []byte, tok Token) bool {
	start := int(tok.Start)
	end := int(tok.End)
	if tok.KindLine&255 != TokenNumber || start < 0 || end < start || end > len(src) {
		return false
	}
	if end-start > 2 && src[start] == '0' {
		prefix := src[start+1]
		if prefix == 'x' || prefix == 'X' {
			for i := start + 2; i < end; i++ {
				c := src[i]
				if c == '.' || c == 'p' || c == 'P' {
					return true
				}
			}
			return false
		}
		if prefix == 'b' || prefix == 'B' || prefix == 'o' || prefix == 'O' {
			return false
		}
	}
	for i := start; i < end; i++ {
		c := src[i]
		if c == '.' || c == 'e' || c == 'E' || c == 'p' || c == 'P' {
			return true
		}
	}
	return false
}
