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

func unicodeRangeValue(ranges string, offset int) int {
	return (int(ranges[offset])-32)<<18 | (int(ranges[offset+1])-32)<<12 | (int(ranges[offset+2])-32)<<6 | (int(ranges[offset+3]) - 32)
}

func unicodeRangeContains(ranges string, value int) bool {
	lo, hi := 0, len(ranges)/12
	for lo < hi {
		mid := lo + (hi-lo)/2
		if unicodeRangeValue(ranges, mid*12+4) < value {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo*12 >= len(ranges) || value < unicodeRangeValue(ranges, lo*12) {
		return false
	}
	return (value-unicodeRangeValue(ranges, lo*12))%unicodeRangeValue(ranges, lo*12+8) == 0
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
