package main

func appMain(args []string) int {
	x := 12
	p := &x
	if *p != 12 {
		print("RENVO-0715 pointer diagnostic failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
