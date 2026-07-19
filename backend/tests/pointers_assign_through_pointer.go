package main

var renvo0628Global int

func appMain(args []string) int {
	p := &renvo0628Global
	*p = 16
	if renvo0628Global != 16 {
		print("RENVO-0628 assign through pointer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
