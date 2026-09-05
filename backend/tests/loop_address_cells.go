package main

func appMain(args []string) int {
	var ps [3]*int
	for i, v := range [3]int{10, 20, 30} {
		ps[i] = &v
	}
	if *ps[0] != 10 || *ps[1] != 20 || *ps[2] != 30 {
		return 1
	}
	for i := 0; i < 3; i++ {
		ps[i] = &i
	}
	if *ps[0] != 0 || *ps[1] != 1 || *ps[2] != 2 {
		return 2
	}
	print("PASS\n")
	return 0
}
