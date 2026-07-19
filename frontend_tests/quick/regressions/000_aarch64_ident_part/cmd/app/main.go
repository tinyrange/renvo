package main

func identStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func identPart(c byte) bool {
	return identStart(c) || (c >= '0' && c <= '9')
}

func scanIdent(src []byte, pos int) int {
	if pos >= len(src) || !identStart(src[pos]) {
		return pos
	}
	pos++
	for pos < len(src) && identPart(src[pos]) {
		pos++
	}
	return pos
}

func main() {
	if scanIdent([]byte("package main"), 0) != 7 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
