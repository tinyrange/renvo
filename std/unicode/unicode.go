//go:generate go run gen.go

package unicode

const MaxRune = 0x10ffff
const ReplacementChar = 0xfffd
const MaxASCII = 0x7f
const MaxLatin1 = 0xff

type rangeEntry struct{ lo, hi, stride uint32 }
type caseEntry struct{ from, to rune }

func inTable(r rune, table []rangeEntry) bool {
	if r < 0 {
		return false
	}
	value := uint32(r)
	low, high := 0, len(table)
	for low < high {
		mid := low + (high-low)/2
		entry := table[mid]
		if value < entry.lo {
			high = mid
		} else if value > entry.hi {
			low = mid + 1
		} else {
			return (value-entry.lo)%entry.stride == 0
		}
	}
	return false
}
func mapCase(r rune, table []caseEntry) rune {
	low, high := 0, len(table)
	for low < high {
		mid := low + (high-low)/2
		if r < table[mid].from {
			high = mid
		} else if r > table[mid].from {
			low = mid + 1
		} else {
			return table[mid].to
		}
	}
	return r
}
func IsLetter(r rune) bool { return inTable(r, letterTable) }
func IsDigit(r rune) bool  { return inTable(r, digitTable) }
func IsNumber(r rune) bool { return inTable(r, numberTable) }
func IsSpace(r rune) bool  { return inTable(r, spaceTable) }
func IsUpper(r rune) bool  { return inTable(r, upperTable) }
func IsLower(r rune) bool  { return inTable(r, lowerTable) }
func ToUpper(r rune) rune  { return mapCase(r, toUpperTable) }
func ToLower(r rune) rune  { return mapCase(r, toLowerTable) }
