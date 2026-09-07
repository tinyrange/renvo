package main

func tooShort() (ok bool) {
	defer func() { ok = recover() != nil }()
	s := []int{1, 2}
	_ = (*[3]int)(s)
	return
}

func appMain(args []string) int {
	if !tooShort() { return 1 }
	s := []int{1, 2, 3}
	p := (*[3]int)(s)
	p[0] = 9
	if s[0] != 9 { return 2 }
	print("PASS\n")
	return 0
}
