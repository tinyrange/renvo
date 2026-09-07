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

func inRanges(r rune, ranges []int) bool {
	if r < 0 || r > MaxRune {
		return false
	}
	lo, hi := 0, len(ranges)/3
	for lo < hi {
		mid := (lo + hi) / 2
		at := mid * 3
		if int(r) < ranges[at] {
			hi = mid
		} else if int(r) > ranges[at+1] {
			lo = mid + 1
		} else {
			return (int(r)-ranges[at])%ranges[at+2] == 0
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
	lo, hi := 0, len(caseRanges)/5
	for lo < hi {
		mid := (lo + hi) / 2
		at := mid * 5
		if int(r) < caseRanges[at] {
			hi = mid
		} else if int(r) > caseRanges[at+1] {
			lo = mid + 1
		} else {
			delta := caseRanges[at+2+caseKind]
			if delta > MaxRune {
				return rune(caseRanges[at] + ((int(r) - caseRanges[at]) &^ 1) + caseKind%2)
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
	lo, hi := 0, len(foldPairs)/2
	for lo < hi {
		mid := (lo + hi) / 2
		at := mid * 2
		if int(r) < foldPairs[at] {
			hi = mid
		} else if int(r) > foldPairs[at] {
			lo = mid + 1
		} else {
			return rune(foldPairs[at+1])
		}
	}
	return r
}
