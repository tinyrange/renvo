package main

func appMain(args []string) int {
	total := 0
	for i, step := 0, 2; i < 4; i = i + 1 {
		total += i + step
	}
	if total != 14 {
		print("RENVO-1043 for init short pair failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
