package main

func appMain(args []string) int {
	a := 27
	b := int(byte(28))
	p := &a
	q := &b
	if p == q {
		print("RENVO-0643 pointer inequality failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
