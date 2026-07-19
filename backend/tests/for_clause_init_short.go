package main

func appMain(args []string) int {
	last := 0
	for i := 2; i < 5; i = i + 1 {
		last = i
	}
	if last != 4 {
		print("RENVO-0403 init short failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
