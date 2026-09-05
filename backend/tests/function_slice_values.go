package main

func three() int { return 3 }
func appMain(args []string) int {
	s := []func() int{three}
	for i := 0; i < 3; i++ {
		s = append(s, func() int { return i })
	}
	if s[0]() != 3 || s[1]() != 0 || s[2]() != 1 || s[3]() != 2 {
		return 1
	}
	s = append(s, three, s[2])
	if s[4]() != 3 || s[5]() != 1 {
		return 2
	}
	print("PASS\n")
	return 0
}
