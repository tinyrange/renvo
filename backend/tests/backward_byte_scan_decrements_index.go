package main

func scanLineStart(src []byte, pos int) int {
	lineStart := pos
	for lineStart > 0 && src[lineStart-1] != '\n' {
		lineStart--
	}
	return lineStart
}

func appMain() int {
	src := []byte("package main\n\nfunc appMain")
	lineStart := scanLineStart(src, len(src))
	if lineStart == 14 {
		print("PASS\n")
	}
	return 0
}
