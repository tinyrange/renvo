package main

func appMain(args []string) int {
	if /* inline comment */ true {
		print("PASS\n")
		return 0
	}
	print("RENVO-0809 comment condition failed\n")
	return 1
	print("PASS\n")
	return 0
}
