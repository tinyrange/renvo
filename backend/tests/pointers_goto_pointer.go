package main

func appMain(args []string) int {
	value := 36
	p := &value
	goto after
after:
	*p = *p + 1
	if value != 37 {
		print("RENVO-0649 goto pointer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
