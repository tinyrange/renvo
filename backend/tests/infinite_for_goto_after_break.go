package main

func appMain(args []string) int {
	x := 0
	for {
		x = 5
		break
	}
	goto check
	print("RENVO-0442 skipped\n")
	return 1
check:
	if x != 5 {
		print("RENVO-0442 goto after break failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
