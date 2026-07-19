package main

func appMain(args []string) int {
	total := 0
	for i := 0; i < 3; i = i + 1 {
		for j := 0; j < 2; j = j + 1 {
			total = total + i + j
		}
	}
	if total != 9 {
		print("RENVO-0416 nested for failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
