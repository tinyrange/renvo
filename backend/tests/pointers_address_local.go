package main

func appMain(args []string) int {
	value := 14
	p := &value
	if *p != 14 {
		print("RENVO-0626 address local failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
