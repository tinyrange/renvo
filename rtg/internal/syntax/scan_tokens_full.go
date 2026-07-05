//go:build !rtg

package syntax

func parseScanTokens(src []byte) ([]Token, bool) {
	var scanner Scanner
	scanner.Scan(src)
	return scanner.Tokens, scanner.Ok
}
