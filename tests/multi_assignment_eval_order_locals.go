package main

func appMain(args []string) int {
	a := 1
	b := 2
	a, b = b, a+b
	if a != 2 || b != 3 {
		print("RTG-1031 local eval order failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
