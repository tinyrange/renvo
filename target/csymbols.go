package target

import (
	"fmt"
	"sort"
)

const C89MinimumExternalName = 6

// CSymbolName records the auditable mapping from a canonical linked-unit
// symbol to the spelling used in a generated C translation unit.
type CSymbolName struct {
	Canonical string
	C         string
}

// MangleC89Symbols assigns compact external names without depending on source
// spelling or on how many identifier characters a C implementation preserves.
// ISO C89 only guarantees six significant monocase characters for external
// identifiers, so unknown historical compilers should use the minimum width.
func MangleC89Symbols(names []string, significant int) ([]CSymbolName, error) {
	if significant < C89MinimumExternalName || significant > 31 {
		return nil, fmt.Errorf("C89 external identifier width must be between %d and 31", C89MinimumExternalName)
	}
	canonical := append([]string(nil), names...)
	sort.Strings(canonical)
	for i, name := range canonical {
		if name == "" {
			return nil, fmt.Errorf("canonical C symbol name is empty")
		}
		if i > 0 && name == canonical[i-1] {
			return nil, fmt.Errorf("duplicate canonical C symbol %q", name)
		}
	}
	digits := significant - 2
	capacity := 1
	maxInt := int(^uint(0) >> 1)
	for i := 0; i < digits; i++ {
		if capacity > maxInt/36 {
			capacity = maxInt
			break
		}
		capacity *= 36
	}
	if len(canonical) > capacity {
		return nil, fmt.Errorf("%d canonical C symbols exceed the %d-character namespace", len(canonical), significant)
	}
	result := make([]CSymbolName, len(canonical))
	for i, name := range canonical {
		result[i] = CSymbolName{Canonical: name, C: c89OrdinalName(i, significant)}
	}
	return result, nil
}

func c89OrdinalName(value int, width int) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	name := make([]byte, width)
	name[0] = 'r'
	name[1] = 'g'
	for i := width - 1; i >= 2; i-- {
		name[i] = alphabet[value%36]
		value /= 36
	}
	return string(name)
}
