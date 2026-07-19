package main

func appMain(args []string) int {
	s := "az"
	b := byte(0)
	b = s[1]
	if b != byte(122) {
		print("RENVO-0348 indexed byte assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
