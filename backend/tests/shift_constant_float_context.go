package main

func appMain(args []string) int {
	const value = 1 << 2 + 0.5
	if value != 4.5 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
