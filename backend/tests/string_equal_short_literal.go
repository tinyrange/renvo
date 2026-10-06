package main

var literalCompareCalls int

func literalOperand(s string) string {
	literalCompareCalls++
	return s
}

func matchesShortLiteral(s string, n int) bool {
	switch n {
	case 0:
		return s == "" && "" == s && !(s != "")
	case 1:
		return s == "a" && "a" == s && !(s != "a")
	case 2:
		return s == "ab" && "ab" == s && !(s != "ab")
	case 3:
		return s == "abc" && "abc" == s && !(s != "abc")
	case 4:
		return s == "abcd" && "abcd" == s && !(s != "abcd")
	case 5:
		return s == "abcde" && "abcde" == s && !(s != "abcde")
	case 6:
		return s == "abcdef" && "abcdef" == s && !(s != "abcdef")
	case 7:
		return s == "abcdefg" && "abcdefg" == s && !(s != "abcdefg")
	case 8:
		return s == "abcdefgh" && "abcdefgh" == s && !(s != "abcdefgh")
	}
	return false
}

func appMain(args []string) int {
	buffer := make([]byte, 32)
	for alignment := 0; alignment < 8; alignment++ {
		for n := 0; n <= 8; n++ {
			for i := 0; i < len(buffer); i++ {
				buffer[i] = 255
			}
			for i := 0; i < n; i++ {
				buffer[alignment+i] = byte('a' + i)
			}
			if !matchesShortLiteral(string(buffer[alignment:alignment+n]), n) {
				print("FAIL equal\n")
				return 1
			}
			if matchesShortLiteral(string(buffer[alignment:alignment+n+1]), n) {
				print("FAIL length\n")
				return 1
			}
			for i := 0; i < n; i++ {
				buffer[alignment+i] ^= 128
				if matchesShortLiteral(string(buffer[alignment:alignment+n]), n) {
					print("FAIL byte\n")
					return 1
				}
				buffer[alignment+i] ^= 128
			}
		}
	}
	var empty []byte
	if string(empty) != "" || "" != string(empty) {
		print("FAIL nil\n")
		return 1
	}
	// UTF-8 puts a high bit in the fourth byte of a packed comparison;
	// escaped literals exercise decoded length rather than source width.
	high := literalOperand("abcé")
	zero := literalOperand("\x00ab")
	if high != "abcé" || "abcé" != high || zero != "\x00ab" {
		print("FAIL encoded\n")
		return 1
	}
	literalCompareCalls = 0
	if literalOperand("abcd") != "abcd" || "abc" != literalOperand("abc") || literalOperand("x") == "y" || literalCompareCalls != 3 {
		print("FAIL evaluation\n")
		return 1
	}
	print("PASS\n")
	return 0
}
