package main

func appMain(args []string) int {
	s := "abc"
	i := 0
	sum := 0
	for i < len(s) {
		sum = sum + int(s[i])
		i = i + 1
	}
	if sum != 294 {
		print("RENVO-0384 string index loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
