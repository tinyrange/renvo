package main

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isIdent(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func tokenEnd(src []byte) int {
	i := 0
	if !isIdentStart(src[i]) {
		return -1
	}
	i++
	for i < len(src) && isIdent(src[i]) {
		i++
	}
	return i
}

func appMain(args []string, env []string) int {
	src := []byte("renvoTokElse")
	end := tokenEnd(src)
	if end != len(src) {
		print("bad token end\n")
		return 1
	}
	print("PASS\n")
	return 0
}
