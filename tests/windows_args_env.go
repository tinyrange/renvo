package main

func windowsLowerASCII(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}

func windowsPathEnvironment(entry string) bool {
	return len(entry) >= 5 &&
		windowsLowerASCII(entry[0]) == 'p' &&
		windowsLowerASCII(entry[1]) == 'a' &&
		windowsLowerASCII(entry[2]) == 't' &&
		windowsLowerASCII(entry[3]) == 'h' &&
		entry[4] == '='
}

func appMain(args []string, env []string) int {
	if len(args) == 0 {
		print("FAIL\n")
		return 1
	}
	for i := 0; i < len(env); i++ {
		if windowsPathEnvironment(env[i]) {
			print("PASS\n")
			return 0
		}
	}
	print("FAIL\n")
	return 1
}
