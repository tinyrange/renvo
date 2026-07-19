package main

func appMain(args []string) int {
	sum := 0
	for i := 0; i < 5; i = i + 1 {
		switch i {
		case 1, 3:
			continue
		}
		sum = sum + i
	}
	if sum != 6 {
		print("RENVO-SWITCH-016 continue failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
