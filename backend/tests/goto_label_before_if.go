package main

func appMain(args []string) int {
	x := 3
	goto check
check:
	if x != 3 {
		print("RENVO-0459 label before if failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
