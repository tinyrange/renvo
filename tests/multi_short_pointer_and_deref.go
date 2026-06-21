package main

func appMain(args []string) int {
	value := 9
	p, copy := &value, value
	*p = 12
	if copy != 9 || value != 12 {
		print("RTG-1045 pointer deref short failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
