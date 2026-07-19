package main

func appMain(args []string) int {
	x := 9 // after assignment
	if x != 9 {
		print("RENVO-0820 trailing comment failed\n")
		return 1
	}
	// after if block
	print("PASS\n")
	return 0
}
