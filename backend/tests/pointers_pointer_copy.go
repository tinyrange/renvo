package main

func appMain(args []string) int {
	value := 29
	p := &value
	q := p
	*q += 1
	if *p != 30 {
		print("RENVO-0644 pointer copy failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
