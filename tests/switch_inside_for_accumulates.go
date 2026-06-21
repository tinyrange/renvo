package main

func appMain(args []string) int {
	sum := 0
	for i := 0; i < 5; i = i + 1 {
		switch i % 3 {
		case 0:
			sum = sum + 10
		case 1:
			sum = sum + 1
		default:
			sum = sum + 4
		}
	}
	if sum != 26 {
		print("RTG-SWITCH-021 loop accumulate failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
