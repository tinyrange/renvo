package main

func less(v int) int {
	if v < 11 {
		return 11
	}
	if v < 8 {
		return 8
	}
	if v < 51 {
		return 51
	}
	return 0
}
func appMain() int {
	if less(7) != 11 {
		panic("literal compare")
	}
	print("PASS\n")
	return 0
}
