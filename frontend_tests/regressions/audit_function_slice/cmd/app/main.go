package main

func f() int { return 3 }
func g() int { return 5 }
func main() {
	s := []func() int{f}
	s = append(s, g, f)
	if s[0]() != 3 || s[1]() != 5 || s[2]() != 3 {
		panic("slice functions")
	}
	println("PASS")
}
