package main

func appMain() int {
	x := .5
	y := +.01
	if x != 0.5 || y != 0.01 || .5e2 != 50 || -.25 != -0.25 {
		return 1
	}
	print("PASS\n")
	return 0
}
