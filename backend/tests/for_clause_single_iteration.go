package main

func appMain(args []string) int {
	n := 0
	for i := 0; i < 1; i = i + 1 {
		n = n + 7
	}
	if n != 7 {
		print("RENVO-0422 single for failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
