package main

func appMain(args []string) int {
	s := "go"
	i := 0
	sum := 0
loop:
	if i < len(s) {
		sum = sum + int(s[i])
		i = i + 1
		goto loop
	}
	if sum != 214 {
		print("RENVO-0469 goto string read failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
