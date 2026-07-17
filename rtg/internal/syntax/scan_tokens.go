package syntax

func parseScanTokens(src []byte) ([]Token, bool) {
	return scanTokens(src)
}
