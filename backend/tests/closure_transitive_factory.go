package main

func returned(v int) func() int { return leaf(v) }
func leaf(v int) func() int     { return func() int { return v } }
func appMain() int {
	if returned(42)() != 42 {
		return 1
	}
	print("PASS\n")
	return 0
}
