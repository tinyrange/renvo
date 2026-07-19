package main

func appMain(args []string) int {
	i := 0
loop:
	if i < 4 {
		i = i + 1
		goto loop
	}
	if i != 4 {
		print("RENVO-0452 backward goto failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
