package main

func isDigit(b byte) bool { return b >= '0' && b <= '9' }
func hexVal(b byte) int {
	if b >= '0' && b <= '9' {
		return int(b - '0')
	}
	if b >= 'a' && b <= 'f' {
		return int(b-'a') + 10
	}
	if b >= 'A' && b <= 'F' {
		return int(b-'A') + 10
	}
	return -1
}
func isIdent(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || isDigit(b) || b == '_'
}
func tagOp(s string, i int) int {
	if i+1 < len(s) {
		if s[i] == '<' && s[i+1] == '=' {
			return 11
		}
		if s[i] == '>' && s[i+1] == '=' {
			return 12
		}
		if s[i] == '&' && s[i+1] == '&' {
			return 13
		}
		if s[i] == '|' && s[i+1] == '|' {
			return 14
		}
		if s[i] == '&' && s[i+1] == '^' {
			return 15
		}
	}
	if s[i] == '<' {
		return 1
	}
	if s[i] == '>' {
		return 2
	}
	if s[i] == '&' {
		return 3
	}
	if s[i] == '|' {
		return 4
	}
	return 0
}

func braceDepth(s string, i int, depth int) int {
	if i >= len(s) {
		return depth
	}
	if s[i] == '{' {
		return braceDepth(s, i+1, depth+1)
	}
	if s[i] == '}' {
		return braceDepth(s, i+1, depth-1)
	}
	return braceDepth(s, i+1, depth)
}
func parseExprValue(s string) int { return 2 + 3*4 }

func appMain(args []string) int {
	var seed int = 0
	text := "2048;"
	i := seed
	value := 0
	for i < len(text) && isDigit(text[i]) {
		value = value*10 + int(text[i]-'0')
		i += 1
	}
	if value != 2048 {
		print("RENVO-0872 integer convert scan failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
