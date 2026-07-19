package main

func appMain(args []string) int {
	b := byte(80)
	p := &b
	if *p == 'P' {
		print("PASS\n")
		return 0
	}
	print("RENVO-0631 byte dereference failed\n")
	return 1
}
