package main

func appMain(args []string) int {
	sum := 0
	for i := 0; i < 5; i = i + 1 {
		if i == 2 {
			continue
		}
		sum += i
	}
	if sum != 8 {
		print("booleans_24 continue\n")
		return 1
	}
	print("PASS\n")
	return 0
}
