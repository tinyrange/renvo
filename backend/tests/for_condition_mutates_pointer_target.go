package main

func appMain(args []string) int {
	x := 0
	p := &x
	for *p < 4 {
		*p = *p + 1
	}
	if x != 4 {
		print("RENVO-0392 pointer loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
