package main

func appMain(args []string) int {
	sum := 0
	for i := 0; i < 4; i = i + 1 {
		if i < 2 {
			sum = sum + 1
		} else {
			sum = sum + 3
		}
	}
	if sum != 8 {
		print("RENVO-0423 if else for failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
