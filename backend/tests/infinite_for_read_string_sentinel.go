package main

func appMain(args []string) int {
	s := "ab!c"
	i := 0
	for {
		if s[i] == byte(33) {
			break
		}
		i = i + 1
	}
	if i != 2 {
		print("RENVO-0437 sentinel infinite failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
