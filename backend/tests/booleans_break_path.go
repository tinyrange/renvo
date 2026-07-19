package main

func appMain(args []string) int {
	n := 0
	for {
		n += 1
		if n == 4 {
			break
		}
	}
	if n != 4 {
		print("booleans_23 break\n")
		return 1
	}
	print("PASS\n")
	return 0
}
