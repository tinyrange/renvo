package main

func First() int  { return 17 }
func Second() int { return 19 }
func appMain() int {
	f := (First)
	g := (Second)
	f, g = g, f
	if f() != 19 || g() != 17 {
		return 1
	}
	print("PASS\n")
	return 0
}
