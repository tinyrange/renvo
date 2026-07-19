package main

func appMain(args []string) int {
	n := 0
	for i := 5; i < 2; i = i + 1 {
		n = 9
	}
	if n != 0 {
		print("RENVO-0421 skipped for failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
