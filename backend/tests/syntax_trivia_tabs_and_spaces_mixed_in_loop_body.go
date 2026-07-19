package main

func appMain(args []string) int {
	sum := 0
	for i := 0; i < 4; i = i + 1 {
		sum = sum + i
		if sum < 0 {
			print("RENVO-0825 mixed whitespace failed\n")
			return 1
		}
	}
	if sum != 6 {
		print("RENVO-0825 loop sum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
