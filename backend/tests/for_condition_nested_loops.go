package main

func appMain(args []string) int {
	outer := 0
	total := 0
	for outer < 3 {
		inner := 0
		for inner < 2 {
			total = total + outer + inner
			inner = inner + 1
		}
		outer = outer + 1
	}
	if total != 9 {
		print("RENVO-0398 nested condition loops failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
