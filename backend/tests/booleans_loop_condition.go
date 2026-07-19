package main

func appMain(args []string) int {
	keep := true
	n := 0
	for keep {
		n += 1
		if n == 3 {
			keep = false
		}
	}
	if n != 3 {
		print("booleans_22 loop\n")
		return 1
	}
	print("PASS\n")
	return 0
}
