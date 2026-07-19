package main

func renvo0550Check(a int, b int) bool {
	return a+b == 9
}

func appMain(args []string) int {
	if !renvo0550Check(4, 5) {
		print("RENVO-0550 first assertion failed\n")
		return 1
	}
	if renvo0550Check(3, 5) {
		print("RENVO-0550 second assertion failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
