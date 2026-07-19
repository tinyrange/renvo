package main

func appMain(args []string) int {
	x := 0
	p := &x
	for i := 0; i < 3; i = i + 1 {
		*p = *p + 2
	}
	if x != 6 {
		print("RENVO-0419 pointer for failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
