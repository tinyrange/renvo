package main

func appMain(args []string) int {
	a := 7
	b := 2
	pa := &a
	pb := &b
	*pa, *pb = *pb+10, *pa+10
	if a != 12 || b != 17 {
		print("RTG-1025 pointer target assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
