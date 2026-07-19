package main

func appMain(args []string) int {
	x := 4
	if x > 3 {
		goto high
	}
	print("RENVO-0463 low path\n")
	return 1
high:
	print("PASS\n")
	return 0
}
