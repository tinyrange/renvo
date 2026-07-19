package main

func appMain(args []string) int {
	x := 0
	p := &x
	for {
		*p = *p + 1
		if *p == 4 {
			break
		}
	}
	if x != 4 {
		print("RENVO-0440 pointer infinite failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
