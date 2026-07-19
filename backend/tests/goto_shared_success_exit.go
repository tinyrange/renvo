package main

func appMain(args []string) int {
	x := 5
	if x == 5 {
		goto success
	}
	print("RENVO-0453 no success\n")
	return 1
success:
	print("PASS\n")
	return 0
}
