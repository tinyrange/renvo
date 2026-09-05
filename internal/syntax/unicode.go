package syntax

// Source encoding has already been validated before scanner calls this helper.
func identifierRune(src []byte, start int) (int, int) {
	if start < 0 || start >= len(src) {
		return 0, 0
	}
	c := src[start]
	if c < 128 {
		return int(c), 1
	}
	width := 2
	value := int(c & 31)
	if c >= 240 {
		width = 4
		value = int(c & 7)
	} else if c >= 224 {
		width = 3
		value = int(c & 15)
	}
	if start+width > len(src) {
		return 0, 0
	}
	for i := 1; i < width; i++ {
		value = value*64 + int(src[start+i]&63)
	}
	return value, width
}

func unicodeRangeContains(ranges []int, value int) bool {
	lo, hi := 0, len(ranges)/3
	for lo < hi {
		mid := lo + (hi-lo)/2
		if ranges[mid*3+1] < value {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo*3 >= len(ranges) || value < ranges[lo*3] {
		return false
	}
	return (value-ranges[lo*3])%ranges[lo*3+2] == 0
}

func unicodeIdentifierWidth(src []byte, start int, first bool) int {
	value, width := identifierRune(src, start)
	if width == 0 {
		return 0
	}
	if unicodeRangeContains(unicodeLetterRanges, value) || !first && unicodeRangeContains(unicodeDigitRanges, value) {
		return width
	}
	return 0
}

// IdentifierExported implements Go's Unicode uppercase-letter export rule.
func IdentifierExported(src []byte, start int) bool {
	if start < 0 || start >= len(src) {
		return false
	}
	if src[start] < 128 {
		return src[start] >= 'A' && src[start] <= 'Z'
	}
	value, width := identifierRune(src, start)
	return width > 0 && unicodeRangeContains(unicodeUpperRanges, value)
}
