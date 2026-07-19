package main

func appMain(args []string) int {
	x := 0
	goto loop
loop:
	for x < 2 {
		x = x + 1
	}
	if x != 2 {
		print("RENVO-0460 label before for failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
