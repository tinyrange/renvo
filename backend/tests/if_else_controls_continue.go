package main

func appMain(args []string) int {
	sum := 0
	for i := 0; i < 5; i = i + 1 {
		if i == 2 {
			continue
		}
		sum = sum + i
	}
	if sum != 8 {
		print("RENVO-0374 continue control failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
