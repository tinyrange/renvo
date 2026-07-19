package main

func appMain(args []string) int {
	ok := true
	p := &ok
	if *p {
		print("PASS\n")
		return 0
	} else {
		print("RENVO-0632 bool address failed\n")
		return 1
	}
}
