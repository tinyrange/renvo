// Package unicode provides Unicode character classification and simple casing.
// Tables are a versioned snapshot; the target does not depend on host Unicode.
package unicode

const (
	MaxRune         = '\U0010FFFF'
	ReplacementChar = '\uFFFD'
	MaxASCII        = '\u007F'
	MaxLatin1       = '\u00FF'
	UpperCase       = 0
	LowerCase       = 1
	TitleCase       = 2
	MaxCase         = 3
	UpperLower      = MaxRune + 1
)

// Each integer occupies four printable six-bit digits. Signed 24-bit values
// cover all code points, strides, case deltas and the UpperLower sentinel.
func tableValue(table string, index int) int {
	at := index * 4
	value := int(table[at]-32)<<18 | int(table[at+1]-32)<<12 | int(table[at+2]-32)<<6 | int(table[at+3]-32)
	if value >= 1<<23 {
		value -= 1 << 24
	}
	return value
}

func inRanges(r rune, ranges string) bool {
	if r < 0 || r > MaxRune {
		return false
	}
	lo, hi := 0, len(ranges)/12
	for lo < hi {
		mid := (lo + hi) / 2
		at := mid * 3
		if int(r) < tableValue(ranges, at) {
			hi = mid
		} else if int(r) > tableValue(ranges, at+1) {
			lo = mid + 1
		} else {
			return (int(r)-tableValue(ranges, at))%tableValue(ranges, at+2) == 0
		}
	}
	return false
}
func IsLetter(r rune) bool { return inRanges(r, letterRanges) }
func IsDigit(r rune) bool  { return inRanges(r, digitRanges) }
func IsSpace(r rune) bool  { return inRanges(r, spaceRanges) }
func IsUpper(r rune) bool  { return inRanges(r, upperRanges) }
func IsLower(r rune) bool  { return inRanges(r, lowerRanges) }
func IsTitle(r rune) bool  { return inRanges(r, titleRanges) }

func To(caseKind int, r rune) rune {
	if caseKind < 0 || caseKind >= MaxCase {
		return ReplacementChar
	}
	if r < 0 || r > MaxRune {
		return r
	}
	lo, hi := 0, len(caseRanges)/20
	for lo < hi {
		mid := (lo + hi) / 2
		at := mid * 5
		if int(r) < tableValue(caseRanges, at) {
			hi = mid
		} else if int(r) > tableValue(caseRanges, at+1) {
			lo = mid + 1
		} else {
			delta := tableValue(caseRanges, at+2+caseKind)
			if delta > MaxRune {
				return rune(tableValue(caseRanges, at) + ((int(r) - tableValue(caseRanges, at)) &^ 1) + caseKind%2)
			}
			return r + rune(delta)
		}
	}
	return r
}
func ToUpper(r rune) rune { return To(UpperCase, r) }
func ToLower(r rune) rune { return To(LowerCase, r) }
func ToTitle(r rune) rune { return To(TitleCase, r) }
func SimpleFold(r rune) rune {
	lo, hi := 0, len(foldPairs)/8
	for lo < hi {
		mid := (lo + hi) / 2
		at := mid * 2
		if int(r) < tableValue(foldPairs, at) {
			hi = mid
		} else if int(r) > tableValue(foldPairs, at) {
			lo = mid + 1
		} else {
			return rune(tableValue(foldPairs, at+1))
		}
	}
	return r
}
