//go:build rtg

package syntax

func parseScanTokens(src []byte) ([]Token, bool) {
	return Scan(src), true
}
